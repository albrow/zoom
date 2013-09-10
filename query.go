// File contains code related to the query abstraction.
// This includes the Find* and Scan* functions and their modifiers.

package zoom

import (
	"errors"
	"fmt"
	"github.com/stephenalexbrowne/zoom/redis"
	"github.com/stephenalexbrowne/zoom/util"
	"reflect"
)

type Query struct {
	scannable Model
	includes  []string
	excludes  []string
	modelName string
	id        string
	err       error
}

type MultiQuery struct {
	scannables []Model
	includes   []string
	excludes   []string
	modelName  string
	sort       sort
	limit      uint
	offset     uint
	err        error
}

type sort struct {
	fieldName string
	desc      bool
	alpha     bool
}

type includerExcluder interface {
	getIncludes() []string
	getExcludes() []string
}

type namedIncluderExcluder interface {
	includerExcluder
	name() string
}

func (q *Query) getIncludes() []string {
	return q.includes
}

func (q *Query) getExcludes() []string {
	return q.excludes
}

func (q *MultiQuery) getIncludes() []string {
	return q.includes
}

func (q *MultiQuery) getExcludes() []string {
	return q.excludes
}

func (q *Query) name() string {
	return q.modelName
}

func (q *MultiQuery) name() string {
	return q.modelName
}

func FindById(modelName, id string) *Query {

	// create a query object
	q := &Query{
		modelName: modelName,
		id:        id,
	}

	// get the type corresponding to the modelName
	typ, err := getRegisteredTypeFromName(modelName)
	if err != nil {
		q.setErrorIfNone(err)
		return q
	}

	// create a new struct of type typ
	val := reflect.New(typ.Elem())
	m, ok := val.Interface().(Model)
	if !ok {
		msg := fmt.Sprintf("zoom: could not convert val of type %T to Model", val.Interface())
		q.setErrorIfNone(errors.New(msg))
		return q
	}

	// set scannable and return the query
	q.scannable = m
	return q
}

func ScanById(m Model, id string) *Query {

	// create a query object
	q := &Query{
		id:        id,
		scannable: m,
	}

	// get the name corresponding to the type of m
	modelName, err := getRegisteredNameFromInterface(m)
	if err != nil {
		q.setErrorIfNone(err)
		return q
	}

	// set modelName and return the query
	q.modelName = modelName
	return q
}

func (q *Query) Include(fields ...string) *Query {
	if len(q.excludes) > 0 {
		q.setErrorIfNone(errors.New("zoom: cannot use both Include and Exclude modifiers on a query"))
		return q
	}
	q.includes = append(q.includes, fields...)
	return q
}

func (q *Query) Exclude(fields ...string) *Query {
	if len(q.includes) > 0 {
		q.setErrorIfNone(errors.New("zoom: cannot use both Include and Exclude modifiers on a query"))
		return q
	}
	q.excludes = append(q.excludes, fields...)
	return q
}

func (q *Query) setErrorIfNone(e error) {
	if q.err == nil {
		q.err = e
	}
}

func (q *Query) Exec() (Model, error) {
	// check if the query had any prior errors
	if q.err != nil {
		return q.scannable, q.err
	}

	// start a transaction
	t := newTransaction()

	// set up includes
	if err := executeModelFindWithIncludes(q.id, q.scannable, q, t); err != nil {
		return q.scannable, err
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return q.scannable, err
	}

	return q.scannable, nil
}

func executeModelFindWithIncludes(id string, scannable Model, q namedIncluderExcluder, t *transaction) error {
	ms := modelSpecs[q.name()]
	includes := ms.fieldNames
	if len(q.getIncludes()) != 0 {
		// add a model find operation to the transaction
		if err := t.findModel(q.name(), id, scannable, q.getIncludes()); err != nil {
			return err
		}
	} else if len(q.getExcludes()) != 0 {
		for _, name := range q.getExcludes() {
			includes = util.RemoveElementFromStringSlice(includes, name)
		}
		// add a model find operation to the transaction
		if err := t.findModel(q.name(), id, scannable, includes); err != nil {
			return err
		}
	} else {
		// add a model find operation to the transaction
		if err := t.findModel(q.name(), id, scannable, nil); err != nil {
			return err
		}
	}
	return nil
}

func FindAll(modelName string) *MultiQuery {
	// create and return a query object
	return &MultiQuery{
		modelName: modelName,
	}
}

func (q *MultiQuery) Include(fields ...string) *MultiQuery {
	if len(q.excludes) > 0 {
		q.setErrorIfNone(errors.New("zoom: cannot use both Include and Exclude modifiers on a query"))
		return q
	}
	q.includes = append(q.includes, fields...)
	return q
}

func (q *MultiQuery) Exclude(fields ...string) *MultiQuery {
	if len(q.includes) > 0 {
		q.setErrorIfNone(errors.New("zoom: cannot use both Include and Exclude modifiers on a query"))
		return q
	}
	q.excludes = append(q.excludes, fields...)
	return q
}

func (q *MultiQuery) SortBy(field string) *MultiQuery {
	q.sort.fieldName = field
	return q
}

func (q *MultiQuery) Order(order string) *MultiQuery {
	if order == "ASC" {
		q.sort.desc = false
	} else if order == "DESC" {
		q.sort.desc = true
	} else {
		q.setErrorIfNone(errors.New("zoom: order must be either ASC or DESC"))
	}
	return q
}

func (q *MultiQuery) Limit(amount uint) *MultiQuery {
	q.limit = amount
	return q
}

func (q *MultiQuery) Offset(amount uint) *MultiQuery {
	q.offset = amount
	return q
}

func (q *MultiQuery) setErrorIfNone(e error) {
	if q.err == nil {
		q.err = e
	}
}

func (q *MultiQuery) Exec() ([]Model, error) {
	// check if the query had any prior errors
	if q.err != nil {
		return q.scannables, q.err
	}

	// get the type corresponding to the modelName
	typ, err := getRegisteredTypeFromName(q.modelName)
	if err != nil {
		return nil, err
	}

	// get the ids for the models
	ids, err := q.getIds(typ)
	if err != nil {
		return nil, err
	}

	// start a transaction
	t := newTransaction()

	// allocate a slice of Model
	q.scannables = make([]Model, len(ids))

	// iterate through the ids and add a find operation for each model
	for i, id := range ids {

		// instantiate a model using reflection
		modelVal := reflect.New(typ.Elem())
		m, ok := modelVal.Interface().(Model)
		if !ok {
			msg := fmt.Sprintf("zoom: could not convert val of type %T to Model", modelVal.Interface())
			return nil, errors.New(msg)
		}

		// set the ith element of scannables
		q.scannables[i] = m

		// add a find operation for the model m
		executeModelFindWithIncludes(id, q.scannables[i], q, t)
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return nil, err
	}

	return q.scannables, nil
}

func (q *MultiQuery) getIds(typ reflect.Type) ([]string, error) {

	// get a connection
	conn := GetConn()
	defer conn.Close()

	// construct a redis command to get the ids
	indexKey := q.modelName + ":index"
	args := redis.Args{}
	var command string
	if q.sort.fieldName == "" {
		// without sorting sorting
		command = "SMEMBERS"
		args = args.Add(indexKey)
	} else {
		// with sorting
		command = "SORT"
		weight := q.modelName + ":*->" + q.sort.fieldName
		args = args.Add(indexKey).Add("BY").Add(weight)

		// figure out if we need the alpha option
		field, found := typ.Elem().FieldByName(q.sort.fieldName)
		if !found {
			msg := fmt.Sprintf("zoom: invalid SortBy modifier. model of type %s has no field %s\n.", typ.String(), q.sort.fieldName)
			return nil, errors.New(msg)
		}
		if field.Type.Kind() == reflect.String {
			args = args.Add("ALPHA")
		}

		// add either ASC or DESC
		if q.sort.desc {
			args = args.Add("DESC")
		} else {
			args = args.Add("ASC")
		}

		// add limit if applicable
		if q.limit != 0 {
			args = args.Add("LIMIT").Add(q.offset).Add(q.limit)
		}
	}
	return redis.Strings(conn.Do(command, args...))
}
