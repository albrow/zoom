// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File query.go contains code related to the query abstraction.

package zoom

import (
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"reflect"
)

type RunScanner interface {
	Run() (interface{}, error)
	Scan(interface{}) error
}

type Query struct {
	modelSpec modelSpec
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

func NewQuery(modelName string) *Query {
	q := &Query{}
	spec, found := modelSpecs[modelName]
	if !found {
		q.setErrorIfNone(NewModelNameNotRegisteredError(modelName))
	} else {
		q.modelSpec = spec
	}
	return q
}

func (q *Query) Run() (interface{}, error) {
	if q.err != nil {
		return nil, q.err
	}

	ids, err := q.getIds()
	if err != nil {
		return nil, err
	}

	// create a slice in which to store results using reflection the
	// type of the slice whill match the type of the model being queried
	resultsVal := reflect.New(reflect.SliceOf(q.modelSpec.modelType))
	resultsVal.Elem().Set(reflect.MakeSlice(reflect.SliceOf(q.modelSpec.modelType), 0, 0))

	if err := scanModelsByIds(resultsVal, q.modelSpec.modelName, ids); err != nil {
		return resultsVal.Elem().Interface(), err
	}
	return resultsVal.Elem().Interface(), nil
}

func (q *Query) Scan(in interface{}) error {
	if q.err != nil {
		return q.err
	}

	// make sure we are dealing with the right type
	typ := reflect.TypeOf(in).Elem()
	if !(typ.Kind() == reflect.Slice) {
		msg := fmt.Sprintf("zoom: Query.Scan requires a pointer to a slice or array as an argument. Got: %T", in)
		return errors.New(msg)
	}
	elemType := typ.Elem()
	if !typeIsPointerToStruct(elemType) {
		msg := fmt.Sprintf("zoom: Query.Scan requires a pointer to a slice of pointers to model structs. Got: %T", in)
		return errors.New(msg)
	}
	if elemType != q.modelSpec.modelType {
		msg := fmt.Sprintf("zoom: argument for Query.Scan did not match the type corresponding to the model name given in the NewQuery constructor.\nExpected %T but got %T", reflect.SliceOf(q.modelSpec.modelType), in)
		return errors.New(msg)
	}

	ids, err := q.getIds()
	if err != nil {
		return err
	}

	resultsVal := reflect.ValueOf(in)
	resultsVal.Elem().Set(reflect.MakeSlice(reflect.SliceOf(q.modelSpec.modelType), 0, 0))

	return scanModelsByIds(resultsVal, q.modelSpec.modelName, ids)
}

func (q *Query) Count() (int, error) {
	if q.err != nil {
		return 0, q.err
	}
	return q.getIdCount()
}

func (q *Query) IdsOnly() ([]string, error) {
	if q.err != nil {
		return nil, q.err
	}
	return q.getIds()
}

func (q *Query) setErrorIfNone(e error) {
	if q.err == nil {
		q.err = e
	}
}

func (q *Query) getIds() ([]string, error) {
	conn := GetConn()
	defer conn.Close()

	// construct a redis command to get the ids
	args := redis.Args{}
	var command string
	if q.sort.fieldName == "" {
		// without sorting or filters
		command = "SMEMBERS"
		indexKey := q.modelSpec.modelName + ":all"
		args = args.Add(indexKey)
	} else {
		// TODO: with sorting and/or filters
		return nil, errors.New("zoom: sorting and filters not implemented yet!")
	}
	return redis.Strings(conn.Do(command, args...))
}

// NOTE: sliceVal should be the value of a pointer to a slice of pointer to models.
// It's type should be *[]*<T>, where <T> is some type which satisfies the Model
// interface. The type *[]*Model is not equivalent and will not work.
func scanModelsByIds(sliceVal reflect.Value, modelName string, ids []string) error {
	t := newTransaction()
	for _, id := range ids {
		mr, err := newModelRefFromName(modelName)
		if err != nil {
			return err
		}
		mr.model.setId(id)
		if err := t.findModel(mr, nil); err != nil {
			if _, ok := err.(*KeyNotFoundError); ok {
				continue // key not found errors are fine
				// TODO: update the index in this case? Or maybe if it keeps happening?
			}
		}
		sliceVal.Elem().Set(reflect.Append(sliceVal.Elem(), mr.modelVal()))
	}
	return t.exec()
}

func (q *Query) getIdCount() (int, error) {
	conn := GetConn()
	defer conn.Close()

	// construct a redis command to get the ids
	args := redis.Args{}
	var command string
	if q.sort.fieldName == "" {
		// without filtering
		command = "SCARD"
		indexKey := q.modelSpec.modelName + ":all"
		args = args.Add(indexKey)
	} else {
		// TODO: with filtering
		return 0, errors.New("zoom: filters not implemented yet!")
	}
	return redis.Int(conn.Do(command, args...))
}
