// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File query.go contains code related to the query abstraction.
// This includes the Find* and Scan* functions and their modifiers.

package zoom

import (
	"errors"
	"fmt"
	"github.com/stephenalexbrowne/zoom/redis"
	"github.com/stephenalexbrowne/zoom/util"
	"reflect"
)

// A Query is an interface which is intended encapsulate sophisticated requests to
// the database. All queries are executed as redis transactions when possible. You
// must call Run on the query when you are ready for the query to be run. Depending
// on the type of the query, certain "modifier" methods may be chained to the
// constructor. Modifiers are Include, Exclude, Sort, Limit, and Offset. A query
// will remember any errors that occur and return the first one when you call Run.
type Query interface {
	Run() (interface{}, error)
}

// A FindQuery is a query which returns a single item from the database
type FindQuery struct {
	scannable Model
	includes  []string
	excludes  []string
	modelName string
	id        string
	err       error
}

// A FindAllQuery is a query wich returns one or more items from the database
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

// FindById returns a FindQuery which can be chained with additional modifiers.
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

// ScanById returns a FindQuery which can be chained with additional modifiers.
// It expects Model as an argument, which should be a pointer to a struct of a
// registered type. ScanById will mutate the struct, filling in its fields.
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

// Include specifies fields to be filled in. Any fields which are included will be unchanged.
func (q *FindQuery) Include(fields ...string) *FindQuery {
	if len(q.excludes) > 0 {
		q.setErrorIfNone(errors.New("zoom: cannot use both Include and Exclude modifiers on a query"))
		return q
	}
	q.includes = append(q.includes, fields...)
	return q
}

// Exclude specifies fields to *not* be filled in. Excluded fields will remain unchanged.
// Any other fields *will* be filled in with the values stored in redis.
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

// Run executes the query, using a transaction if possible. The first return
// value is a Model, i.e. a pointer to a struct. When using the ScanById
// constructor, the first return value is typically not needed. The second
// return value is the first error (if any) that occured in the query constructor
// or modifier methods.
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

// FindAll returns a FindAllQuery which can be chained with additional modifiers.
func FindAll(modelName string) *FindAllQuery {

	// create a query object
	q := &FindAllQuery{
		modelName: modelName,
	}

	// get the registered type corresponding to the modelName
	typ, err := getRegisteredTypeFromName(modelName)
	if err != nil {
		q.setErrorIfNone(err)
		return q
	}

	// instantiate a new slice and set it as scannables
	q.modelType = typ
	newVal := reflect.New(reflect.SliceOf(typ))
	newVal.Elem().Set(reflect.MakeSlice(reflect.SliceOf(typ), 0, 0))
	q.scannables = newVal.Interface()
	return q
}

// ScanAll returns a FindAllQuery which can be chained with additional modifiers.
// It expects a pointer to a slice (or array) of Models as an argument, which should be
// a pointer to a slice (or array) of pointers to structs of a registered type. ScanAll
// will mutate the slice or array by appending to it.
func ScanAll(models interface{}) *FindAllQuery {

	// create a query object
	q := new(FindAllQuery)

	// make sure models is the right type
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

	// get the registered name corresponding to the type of models
	modelName, found := typeToName[elemType]
	if !found {
		q.setErrorIfNone(NewModelTypeNotRegisteredError(elemType))
		return q
	}
	q.modelName = modelName
	q.scannables = models
	return q
}

// Include specifies fields to be filled in for each model. Any fields which are included
// will be unchanged.
func (q *FindAllQuery) Include(fields ...string) *FindAllQuery {
	if len(q.excludes) > 0 {
		q.setErrorIfNone(errors.New("zoom: cannot use both Include and Exclude modifiers on a query"))
		return q
	}
	q.includes = append(q.includes, fields...)
	return q
}

// Exclude specifies fields to *not* be filled in for each struct. Excluded fields will
// remain unchanged. Any other fields *will* be filled in with the values stored in redis.
func (q *FindAllQuery) Exclude(fields ...string) *FindAllQuery {
	if len(q.includes) > 0 {
		q.setErrorIfNone(errors.New("zoom: cannot use both Include and Exclude modifiers on a query"))
		return q
	}
	q.excludes = append(q.excludes, fields...)
	return q
}

// SortBy specifies a field to sort by. The field argument should be exactly the name of
// an exported field. Will cause an error if the field is not found.
func (q *FindAllQuery) SortBy(field string) *FindAllQuery {
	q.sort.fieldName = field
	return q
}

// Order specifies the order in which records should be sorted. It should be either ASC
// or DESC. Any other argument will cause an error.
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

// Limit specifies an upper limit on the number of records to return.
func (q *FindAllQuery) Limit(amount uint) *FindAllQuery {
	q.limit = amount
	return q
}

// Offset specifies a starting index from which to start counting records that
// will be returned.
func (q *FindAllQuery) Offset(amount uint) *FindAllQuery {
	q.offset = amount
	return q
}

func (q *FindAllQuery) setErrorIfNone(e error) {
	if q.err == nil {
		q.err = e
	}
}

// Run executes the query, using a transaction if possible. The first return value
// of Run will be a slice of Models, i.e. a slice of pointers to structs. When
// using the ScanAll constructor, the first return value is typically not needed.
// The second return value is the first error (if any) that occured in the query
// constructor or modifier methods.
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
	conn := GetConn()
	defer conn.Close()

	// construct a redis command to get the ids
	indexKey := q.modelName + ":index"
	args := redis.Args{}
	var command string
	if q.sort.fieldName == "" {
		// without sorting
		command = "SMEMBERS"
		args = args.Add(indexKey)
	} else {
		// with sorting
		command = "SORT"
		weight := q.modelName + ":*->" + q.sort.fieldName
		args = args.Add(indexKey).Add("BY").Add(weight)

		// check if the field is sortable and if we need the alpha option
		field, found := q.modelType.Elem().FieldByName(q.sort.fieldName)
		if !found {
			msg := fmt.Sprintf("zoom: invalid SortBy modifier. model of type %s has no field %s\n.", q.modelType.String(), q.sort.fieldName)
			return nil, errors.New(msg)
		}
		fieldType := field.Type
		if !typeIsSortable(fieldType) {
			msg := fmt.Sprintf("zoom: invalid SortBy modifier. field of type %s is not sortable.\nmust be string, int, uint, float, byte, or bool.", fieldType.String())
			return nil, errors.New(msg)
		}
		if util.TypeIsString(fieldType) {
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

func typeIsSortable(typ reflect.Type) bool {
	return util.TypeIsString(typ) || util.TypeIsNumeric(typ) || util.TypeIsBool(typ)
}
