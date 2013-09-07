// File contains code related to the query abstraction.
// This includes the Find* and Scan* functions and their modifiers.

package zoom

import (
	"errors"
	"fmt"
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

	// add a model find operation to the transaction
	if err := t.addModelFind(q.modelName, q.id, q.scannable); err != nil {
		return q.scannable, err
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return q.scannable, err
	}

	return q.scannable, nil
}
