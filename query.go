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

type Query interface {
	Run() (interface{}, error)
}

type FindQuery struct {
	scannable Model
	includes  []string
	excludes  []string
	modelName string
	id        string
	err       error
}

type FindAllQuery struct {
	scannables interface{}
	includes   []string
	excludes   []string
	modelName  string
	modelType  reflect.Type
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

func (q *FindQuery) getIncludes() []string {
	return q.includes
}

func (q *FindQuery) getExcludes() []string {
	return q.excludes
}

func (q *FindAllQuery) getIncludes() []string {
	return q.includes
}

func (q *FindAllQuery) getExcludes() []string {
	return q.excludes
}

func (q *FindQuery) name() string {
	return q.modelName
}

func (q *FindAllQuery) name() string {
	return q.modelName
}

func FindById(modelName, id string) *FindQuery {

	// create a query object
	q := &FindQuery{
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

func ScanById(m Model, id string) *FindQuery {

	// create a query object
	q := &FindQuery{
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

func (q *FindQuery) Include(fields ...string) *FindQuery {
	if len(q.excludes) > 0 {
		q.setErrorIfNone(errors.New("zoom: cannot use both Include and Exclude modifiers on a query"))
		return q
	}
	q.includes = append(q.includes, fields...)
	return q
}

func (q *FindQuery) Exclude(fields ...string) *FindQuery {
	if len(q.includes) > 0 {
		q.setErrorIfNone(errors.New("zoom: cannot use both Include and Exclude modifiers on a query"))
		return q
	}
	q.excludes = append(q.excludes, fields...)
	return q
}

func (q *FindQuery) setErrorIfNone(e error) {
	if q.err == nil {
		q.err = e
	}
}

func (q *FindQuery) Run() (interface{}, error) {
	// check if the query had any prior errors
	if q.err != nil {
		return q.scannable, q.err
	}

	// start a transaction
	t := newTransaction()

	// set up includes
	if err := findModelWithIncludes(q.id, q.scannable, q, t); err != nil {
		return q.scannable, err
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return q.scannable, err
	}

	return q.scannable, nil
}

func FindAll(modelName string) *FindAllQuery {

	q := &FindAllQuery{
		modelName: modelName,
	}

	typ, err := getRegisteredTypeFromName(modelName)
	if err != nil {
		q.setErrorIfNone(err)
		return q
	}

	q.modelType = typ
	newVal := reflect.New(reflect.SliceOf(typ))
	newVal.Elem().Set(reflect.MakeSlice(reflect.SliceOf(typ), 0, 0))
	q.scannables = newVal.Interface()

	return q
}

func ScanAll(models interface{}) *FindAllQuery {

	// create a query object
	q := new(FindAllQuery)

	// get the name corresponding to the type of m
	typ := reflect.TypeOf(models).Elem()
	if !util.TypeIsSliceOrArray(typ) {
		msg := fmt.Sprintf("zoom: ScanAll requires a pointer to a slice slice or array as an argument. Got: %T", models)
		q.setErrorIfNone(errors.New(msg))
		return q
	}
	elemType := typ.Elem()
	if !util.TypeIsPointerToStruct(elemType) {
		msg := fmt.Sprintf("zoom: ScanAll requires a pointer to a slice of pointers to structs. Got: %T", models)
		q.setErrorIfNone(errors.New(msg))
		return q
	}
	q.modelType = elemType

	modelName, found := typeToName[elemType]
	if !found {
		q.setErrorIfNone(NewModelTypeNotRegisteredError(elemType))
		return q
	}
	q.modelName = modelName
	q.scannables = models

	return q
}

func (q *FindAllQuery) Include(fields ...string) *FindAllQuery {
	if len(q.excludes) > 0 {
		q.setErrorIfNone(errors.New("zoom: cannot use both Include and Exclude modifiers on a query"))
		return q
	}
	q.includes = append(q.includes, fields...)
	return q
}

func (q *FindAllQuery) Exclude(fields ...string) *FindAllQuery {
	if len(q.includes) > 0 {
		q.setErrorIfNone(errors.New("zoom: cannot use both Include and Exclude modifiers on a query"))
		return q
	}
	q.excludes = append(q.excludes, fields...)
	return q
}

func (q *FindAllQuery) SortBy(field string) *FindAllQuery {
	q.sort.fieldName = field
	return q
}

func (q *FindAllQuery) Order(order string) *FindAllQuery {
	if order == "ASC" {
		q.sort.desc = false
	} else if order == "DESC" {
		q.sort.desc = true
	} else {
		q.setErrorIfNone(errors.New("zoom: order must be either ASC or DESC"))
	}
	return q
}

func (q *FindAllQuery) Limit(amount uint) *FindAllQuery {
	q.limit = amount
	return q
}

func (q *FindAllQuery) Offset(amount uint) *FindAllQuery {
	q.offset = amount
	return q
}

func (q *FindAllQuery) setErrorIfNone(e error) {
	if q.err == nil {
		q.err = e
	}
}

func (q *FindAllQuery) Run() (interface{}, error) {

	// check if the query had any prior errors
	if q.err != nil {
		return nil, q.err
	}

	// use reflection to get a value for scannables
	scannablesVal := reflect.ValueOf(q.scannables).Elem()

	// get the ids for the models
	ids, err := q.getIds()
	if err != nil {
		return scannablesVal.Interface(), err
	}

	// start a transaction
	t := newTransaction()

	// iterate through the ids and add a find operation for each model
	for _, id := range ids {

		// instantiate a new scannable element and append it to q.scannables
		scannable := reflect.New(q.modelType.Elem())
		scannablesVal.Set(reflect.Append(scannablesVal, scannable))

		model, ok := scannable.Interface().(Model)
		if !ok {
			msg := fmt.Sprintf("zoom: could not convert val of type %s to Model\n", scannable.Type().String())
			return scannablesVal.Interface(), errors.New(msg)
		}

		// add a find operation for the model m
		findModelWithIncludes(id, model, q, t)
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return scannablesVal.Interface(), err
	}

	return scannablesVal.Interface(), nil
}

func findModelWithIncludes(id string, scannable Model, q namedIncluderExcluder, t *transaction) error {
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

func (q *FindAllQuery) getIds() ([]string, error) {

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
		field, found := q.modelType.Elem().FieldByName(q.sort.fieldName)
		if !found {
			msg := fmt.Sprintf("zoom: invalid SortBy modifier. model of type %s has no field %s\n.", q.modelType.String(), q.sort.fieldName)
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
