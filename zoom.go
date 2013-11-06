// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File zoom.go contains glue code that connects the Model
// interface to the database. The most basic
// public-facing methods are here.

// Package zoom is a lightweight, blazing-fast ORM powered by Redis.
package zoom

import (
	"errors"
	"fmt"
	"reflect"
)

// Save writes an arbitrary struct (or structs) to the redis database.
// Save throws an error if the type of the struct has not yet been registered.
// If the Id field of the struct is empty, Save will mutate the struct by setting
// the Id. To make a struct satisfy the Model interface, you can embed zoom.DefaultData.
func Save(models ...Model) error {
	t := newTransaction()
	for _, m := range models {

		// make sure we'll dealing with a pointer to a struct
		if !typeIsPointerToStruct(reflect.TypeOf(m)) {
			msg := fmt.Sprintf("zoom: Save() requires a pointer to a struct as an argument.\nThe type %T is not a pointer to a struct.", m)
			return errors.New(msg)
		}

		if err := t.saveModel(m); err != nil {
			return err
		}
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return err
	}
	return nil
}

// FindById gets a model from the database. It returns an error
// if a model with that id does not exist.
func FindById(modelName, id string) (Model, error) {

	// create a new struct of proper type
	typ, err := getRegisteredTypeFromName(modelName)
	if err != nil {
		return nil, err
	}
	val := reflect.New(typ.Elem())
	m, ok := val.Interface().(Model)
	if !ok {
		msg := fmt.Sprintf("zoom: could not convert val of type %T to Model", val.Interface())
		return nil, errors.New(msg)
	}

	// invoke ScanById
	if err := ScanById(id, m); err != nil {
		return m, err
	}
	return m, nil
}

// ScanById returns a ModelQuery which can be chained with additional modifiers.
// It expects Model as an argument, which should be a pointer to a struct of a
// registered type. ScanById will mutate the struct, filling in its fields.
func ScanById(id string, m Model) error {

	// create a modelRef
	mr, err := newModelRefFromInterface(m)
	if err != nil {
		return err
	}
	mr.model.setId(id)

	// start a transaction
	t := newTransaction()
	t.findModel(mr, nil)

	// execute the transaction and return the result
	if err := t.exec(); err != nil {
		return err
	}
	return nil
}

// Delete removes a struct (or structs) from the database. Will
// throw an error if the type of the struct has not yet been
// registered, or if the Id field of the struct is empty.
func Delete(models ...Model) error {
	t := newTransaction()
	for _, m := range models {
		if m.getId() == "" {
			return errors.New("zoom: cannot delete because model Id field is empty")
		}
		modelName, err := getRegisteredNameFromInterface(m)
		if err != nil {
			return err
		}
		if err := t.deleteModel(modelName, m.getId()); err != nil {
			return err
		}
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return err
	}
	return nil
}

// DeleteById removes a struct (or structs) from the database by its
// registered name and id. The modelName argument should be the same
// string name that was used in the Register function. If using variadic
// paramaters, you can only delete models of the same registered name and type.
func DeleteById(modelName string, ids ...string) error {
	t := newTransaction()
	for _, id := range ids {
		if err := t.deleteModel(modelName, id); err != nil {
			return err
		}
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return err
	}
	return nil
}
