// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File query.go contains code related to the query abstraction.
// This includes the Find* and Scan* functions and their modifiers.

package zoom

import (
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
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

// A ModelQuery is a query which returns a single item from the database.
// It can be chained with query modifiers.
type ModelQuery struct {
	modelRef modelRef
	includes []string
	excludes []string
	err      error
}

// A MultiModelQuery is a query wich returns one or more items from the database.
// It can be chained with query modifiers.
type MultiModelQuery struct {
	modelSpec modelSpec
	models    interface{}
	includes  []string
	excludes  []string
	sort      sort
	limit     uint
	offset    uint
	err       error
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

func (q *ModelQuery) getIncludes() []string {
	return q.includes
}

func (q *ModelQuery) getExcludes() []string {
	return q.excludes
}

func (q *MultiModelQuery) getIncludes() []string {
	return q.includes
}

func (q *MultiModelQuery) getExcludes() []string {
	return q.excludes
}

func (q *ModelQuery) name() string {
	return q.modelRef.modelSpec.modelName
}

func (q *MultiModelQuery) name() string {
	return q.modelSpec.modelName
}

// FindById returns a ModelQuery which can be chained with additional modifiers.
func FindById(modelName, id string) *ModelQuery {

	q := &ModelQuery{}

	mr, err := newModelRefFromName(modelName)
	if err != nil {
		q.setErrorIfNone(err)
	}
	q.modelRef = mr

	// create a new struct of type typ
	val := reflect.New(mr.modelSpec.modelType.Elem())
	m, ok := val.Interface().(Model)
	if !ok {
		msg := fmt.Sprintf("zoom: could not convert val of type %T to Model", val.Interface())
		q.setErrorIfNone(errors.New(msg))
		return q
	}

	// set model and return the query
	q.modelRef.model = m
	q.modelRef.model.setId(id)
	return q
}

// ScanById returns a ModelQuery which can be chained with additional modifiers.
// It expects Model as an argument, which should be a pointer to a struct of a
// registered type. ScanById will mutate the struct, filling in its fields.
func ScanById(id string, m Model) *ModelQuery {

	// create and return a query object
	q := &ModelQuery{}
	mr, err := newModelRefFromInterface(m)
	if err != nil {
		q.setErrorIfNone(err)
	}
	q.modelRef = mr
	q.modelRef.model.setId(id)
	return q
}

// Include specifies fields to be filled in. Any fields which are included will be unchanged.
func (q *ModelQuery) Include(fields ...string) *ModelQuery {
	if len(q.excludes) > 0 {
		q.setErrorIfNone(errors.New("zoom: cannot use both Include and Exclude modifiers on a query"))
		return q
	}
	q.includes = append(q.includes, fields...)
	return q
}

// Exclude specifies fields to *not* be filled in. Excluded fields will remain unchanged.
// Any other fields *will* be filled in with the values stored in redis.
func (q *ModelQuery) Exclude(fields ...string) *ModelQuery {
	if len(q.includes) > 0 {
		q.setErrorIfNone(errors.New("zoom: cannot use both Include and Exclude modifiers on a query"))
		return q
	}
	q.excludes = append(q.excludes, fields...)
	return q
}

func (q *ModelQuery) setErrorIfNone(e error) {
	if q.err == nil {
		q.err = e
	}
}

// Run executes the query, using a transaction if possible. The first return
// value is a Model, i.e. a pointer to a struct. When using the ScanById
// constructor, the first return value is typically not needed. The second
// return value is the first error (if any) that occured in the query constructor
// or modifier methods.
func (q *ModelQuery) Run() (interface{}, error) {
	// check if the query had any prior errors
	if q.err != nil {
		return q.modelRef.model, q.err
	}

	// start a transaction
	t := newTransaction()

	// set up includes
	if err := findModelWithIncludes(q.modelRef, q, t); err != nil {
		return q.modelRef.model, err
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return q.modelRef.model, err
	}
	return q.modelRef.model, nil
}

// FindAll returns a MultiModelQuery which can be chained with additional modifiers.
func FindAll(modelName string) *MultiModelQuery {

	// create a query object
	q := new(MultiModelQuery)

	// get the registered type corresponding to the modelName
	typ, err := getRegisteredTypeFromName(modelName)
	if err != nil {
		q.setErrorIfNone(err)
		return q
	}
	q.modelSpec = newModelSpec(modelName, typ)

	modelsVal := reflect.New(reflect.SliceOf(typ))
	modelsVal.Elem().Set(reflect.MakeSlice(reflect.SliceOf(typ), 0, 0))
	q.models = modelsVal.Interface()

	return q
}

// ScanAll returns a MultiModelQuery which can be chained with additional modifiers.
// It expects a pointer to a slice (or array) of Models as an argument, which should be
// a pointer to a slice (or array) of pointers to structs of a registered type. ScanAll
// will mutate the slice or array by appending to it.
func ScanAll(models interface{}) *MultiModelQuery {

	// create a query object
	q := new(MultiModelQuery)

	// make sure models is the right type
	typ := reflect.TypeOf(models).Elem()
	if !util.TypeIsSliceOrArray(typ) {
		msg := fmt.Sprintf("zoom: ScanAll requires a pointer to a slice or array as an argument. Got: %T", models)
		q.setErrorIfNone(errors.New(msg))
		return q
	}
	modelType := typ.Elem()
	if !util.TypeIsPointerToStruct(modelType) {
		msg := fmt.Sprintf("zoom: ScanAll requires a pointer to a slice of pointers to structs. Got: %T", models)
		q.setErrorIfNone(errors.New(msg))
		return q
	}
	// get the registered name corresponding to the type of models
	modelName, err := getRegisteredNameFromType(modelType)
	if err != nil {
		q.setErrorIfNone(err)
	}
	q.modelSpec = newModelSpec(modelName, modelType)
	q.models = models

	return q
}

// Include specifies fields to be filled in for each model. Any fields which are included
// will be unchanged.
func (q *MultiModelQuery) Include(fields ...string) *MultiModelQuery {
	if len(q.excludes) > 0 {
		q.setErrorIfNone(errors.New("zoom: cannot use both Include and Exclude modifiers on a query"))
		return q
	}
	q.includes = append(q.includes, fields...)
	return q
}

// Exclude specifies fields to *not* be filled in for each struct. Excluded fields will
// remain unchanged. Any other fields *will* be filled in with the values stored in redis.
func (q *MultiModelQuery) Exclude(fields ...string) *MultiModelQuery {
	if len(q.includes) > 0 {
		q.setErrorIfNone(errors.New("zoom: cannot use both Include and Exclude modifiers on a query"))
		return q
	}
	q.excludes = append(q.excludes, fields...)
	return q
}

// SortBy specifies a field to sort by. The field argument should be exactly the name of
// an exported field. Will cause an error if the field is not found.
func (q *MultiModelQuery) SortBy(field string) *MultiModelQuery {
	q.sort.fieldName = field
	return q
}

// Order specifies the order in which records should be sorted. It should be either ASC
// or DESC. Any other argument will cause an error.
func (q *MultiModelQuery) Order(order string) *MultiModelQuery {
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
func (q *MultiModelQuery) Limit(amount uint) *MultiModelQuery {
	q.limit = amount
	return q
}

// Offset specifies a starting index from which to start counting records that
// will be returned.
func (q *MultiModelQuery) Offset(amount uint) *MultiModelQuery {
	q.offset = amount
	return q
}

func (q *MultiModelQuery) setErrorIfNone(e error) {
	if q.err == nil {
		q.err = e
	}
}

// Run executes the query, using a transaction if possible. The first return value
// of Run will be a slice of Models, i.e. a slice of pointers to structs. When
// using the ScanAll constructor, the first return value is typically not needed.
// The second return value is the first error (if any) that occured in the query
// constructor or modifier methods.
func (q *MultiModelQuery) Run() (interface{}, error) {

	// check if the query had any prior errors
	if q.err != nil {
		return nil, q.err
	}

	// use reflection to get a value for models
	modelsVal := reflect.ValueOf(q.models).Elem()

	// get the ids for the models
	ids, err := q.getIds()
	if err != nil {
		return modelsVal.Interface(), err
	}

	// start a transaction
	t := newTransaction()

	// iterate through the ids and add a find operation for each model
	for _, id := range ids {

		// instantiate a new scannable element and append it to q.models
		scannable := reflect.New(q.modelSpec.modelType.Elem())
		modelsVal.Set(reflect.Append(modelsVal, scannable))

		model, ok := scannable.Interface().(Model)
		if !ok {
			msg := fmt.Sprintf("zoom: could not convert val of type %s to Model\n", scannable.Type().String())
			return modelsVal.Interface(), errors.New(msg)
		}
		mr, err := newModelRefFromInterface(model)
		if err != nil {
			return modelsVal, err
		}
		mr.model.setId(id)

		// add a find operation for the model m
		findModelWithIncludes(mr, q, t)
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return modelsVal.Interface(), err
	}

	return modelsVal.Interface(), nil
}

func findModelWithIncludes(mr modelRef, q namedIncluderExcluder, t *transaction) error {
	//ms := modelSpecs[q.name()]
	//includes := ms.fieldNames
	includes := make([]string, 0)
	if len(q.getIncludes()) != 0 {
		// add a model find operation to the transaction
		if err := t.findModel(mr, q.getIncludes()); err != nil {
			return err
		}
	} else if len(q.getExcludes()) != 0 {
		for _, name := range q.getExcludes() {
			includes = util.RemoveElementFromStringSlice(includes, name)
		}
		// add a model find operation to the transaction
		if err := t.findModel(mr, includes); err != nil {
			return err
		}
	} else {
		// add a model find operation to the transaction
		if err := t.findModel(mr, nil); err != nil {
			return err
		}
	}
	return nil
}

func (q *MultiModelQuery) getIds() ([]string, error) {
	conn := GetConn()
	defer conn.Close()

	// construct a redis command to get the ids
	args := redis.Args{}
	var command string
	if q.sort.fieldName == "" {
		// without sorting
		command = "SMEMBERS"
		args = args.Add(q.modelSpec.indexKey())
	} else {
		// with sorting
		command = "SORT"
		weight := q.modelSpec.modelName + ":*->" + q.sort.fieldName
		args = args.Add(q.modelSpec.indexKey()).Add("BY").Add(weight)

		// check if the field is sortable and if we need the alpha option
		field, found := q.modelSpec.field(q.sort.fieldName)
		if !found {
			msg := fmt.Sprintf("zoom: invalid SortBy modifier. model of type %s has no field %s\n.", q.modelSpec.modelType.String(), q.sort.fieldName)
			return nil, errors.New(msg)
		}
		if !typeIsSortable(field.Type) {
			msg := fmt.Sprintf("zoom: invalid SortBy modifier. field of type %s is not sortable.\nmust be string, int, uint, float, byte, or bool.", field.Type.String())
			return nil, errors.New(msg)
		}
		if util.TypeIsString(field.Type) {
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
	if typ.Kind() == reflect.Ptr {
		return util.TypeIsPrimative(typ.Elem())
	} else {
		return util.TypeIsPrimative(typ)
	}
}
