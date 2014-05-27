// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File zoom.go contains glue code that connects the Model
// interface to the database. The most basic
// public-facing methods are here.

// Package zoom is a fast and lightweight ORM powered by Redis.
// It supports models of any arbitrary struct type, supports relationships
// between models, and provides basic querying functionality. It also
// supports running Redis commands directly.
package zoom

import (
	"errors"
	"fmt"
	"reflect"
)

// Save writes a model (a struct which satisfies the Model interface) to the redis
// database. Save throws an error if the type of the struct has not yet been registered
// or if there is a problem connecting to the database. If the Id field of the struct is
// empty, Save will mutate the struct by setting the Id. To make a struct satisfy the Model
// interface, you can embed zoom.DefaultData.
func Save(model Model) error {
	t := newTransaction()

	// add a save operation to the transaction
	if err := t.saveModel(model); err != nil {
		return err
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return err
	}
	return nil
}

// MSave is like Save but accepts a slice of models and saves them all in
// a single transaction. See http://redis.io/topics/transactions. If there
// is an error in the middle of the transaction, any models that were saved
// before the error was encountered will still be saved. Usually this is fine
// because saving a model a second time will have no adverse effects.
func MSave(models []Model) error {
	t := newTransaction()

	// add a save operation for each model to the transaction
	for _, m := range models {
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
// if a model with that id does not exist or if there was a problem
// connecting to the database. By default modelName should be the
// string version of the type of model (without the asterisk, ampersand,
// or package prefix). If you used RegisterName instead of Register,
// modelName should be the custom name you used.
func FindById(modelName, id string) (Model, error) {

	// create a new struct of proper type
	typ, err := getRegisteredTypeFromName(modelName)
	if err != nil {
		return nil, err
	}
	val := reflect.New(typ.Elem())
	m, ok := val.Interface().(Model)
	if !ok {
		err := fmt.Errorf("zoom: could not convert val of type %T to Model\n", val.Interface())
		return nil, err
	}

	// invoke ScanById
	if err := ScanById(id, m); err != nil {
		return m, err
	}
	return m, nil
}

// MFindById is like FindById but accepts a slice of model names and ids
// and returns a slice of models. It executes the commands needed to retrieve
// the models in a single transaction. See http://redis.io/topics/transactions.
// The slice of modelNames and ids should be properly aligned so that, e.g.,
// modelNames[0] corresponds to ids[0]. If there is an error in the middle of
// the transaction, the function will halt and return the models retrieved so
// far (as well as the error).
func MFindById(modelNames, ids []string) ([]Model, error) {

	if len(modelNames) != len(ids) {
		return nil, errors.New("Zoom: error in MFindById: modelNames and ids must be the same length")
	}

	t := newTransaction()
	results := make([]Model, 0)

	for i := 0; i < len(modelNames); i++ {
		name, id := modelNames[i], ids[i]

		// create a new struct of proper type
		typ, err := getRegisteredTypeFromName(name)
		if err != nil {
			return results, err
		}
		val := reflect.New(typ.Elem())
		m, ok := val.Interface().(Model)
		if !ok {
			err := fmt.Errorf("zoom: could not convert val of type %T to Model\n", val.Interface())
			return results, err
		}
		results = append(results, m)

		// create a modelRef
		mr, err := newModelRefFromModel(m)
		if err != nil {
			return results, err
		}
		mr.model.SetId(id)

		// add a find operation to the transaction
		t.findModel(mr, nil)
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return results, err
	}
	return results, nil
}

// ScanById retrieves a model from redis and scans it into model.
// model should be a pointer to a struct of a registered type. ScanById
// will mutate the struct, filling in its fields. It returns an error
// if a model with that id does not exist or if there was a problem
// connecting to the database.
func ScanById(id string, model Model) error {

	// create a modelRef
	mr, err := newModelRefFromModel(model)
	if err != nil {
		return err
	}
	mr.model.SetId(id)

	// start a transaction
	t := newTransaction()
	t.findModel(mr, nil)

	// execute the transaction and return the result
	if err := t.exec(); err != nil {
		return err
	}
	return nil
}

// MScanById is like ScanById but accepts a slice of ids and a pointer to
// a slice of models. It executes the commands needed to retrieve the models
// in a single transaction. See http://redis.io/topics/transactions.
// The slice of ids and models should be properly aligned so that, e.g.,
// ids[0] corresponds to models[0]. If there is an error in the middle of the
// transaction, the function will halt and return the error. Any models that
// were scanned before the error will still be valid. If any of the models in
// the models slice are nil, MScanById will use reflection to allocate memory
// for them.
func MScanById(ids []string, models interface{}) error {

	// since this is somewhat type-unsafe, we need to verify that
	// models is the correct type
	modelsVal := reflect.ValueOf(models).Elem()
	modelType := modelsVal.Type().Elem()

	if len(ids) != modelsVal.Len() {
		return errors.New("Zoom: error in MScanById: ids and models must be the same length")
	} else if !typeIsSliceOrArray(modelsVal.Type()) {
		return errors.New("Zoom: error in MScanById: models should be a pointer to a slice or array of models")
	} else if !typeIsPointerToStruct(modelType) {
		return errors.New("Zoom: error in MScanById: the elements in models should be pointers to structs")
	} else if !modelTypeIsRegistered(modelType) {
		return fmt.Errorf("Zoom: error in MScanById: the elements in models should be of a registered type\nType %s has not been registered.", modelType.String())
	}

	t := newTransaction()
	for i := 0; i < len(ids); i++ {
		id, mVal := ids[i], modelsVal.Index(i)

		if mVal.IsNil() {
			mVal.Set(reflect.New(modelType.Elem()))
		}

		// create a modelRef
		mr, err := newModelRefFromInterface(mVal.Interface())
		if err != nil {
			return err
		}
		mr.model.SetId(id)

		// start a transaction
		t.findModel(mr, nil)
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return err
	}
	return nil
}

// Delete removes a model from the database. It will throw an error if
// the type of the model has not yet been registered, if the Id field
// of the model is empty, or if there is a problem connecting to the
// database. If the model does not exist in the database, Delete will
// not return an error; it will simply have no effect.
func Delete(model Model) error {
	t := newTransaction()

	if model.GetId() == "" {
		return errors.New("zoom: cannot delete because model Id field is empty")
	}
	mr, err := newModelRefFromModel(model)
	if err != nil {
		return err
	}
	t.deleteModel(mr)

	// execute the transaction
	if err := t.exec(); err != nil {
		return err
	}
	return nil
}

// MDelete is like Delete but accepts a slice of models and
// deletes them all in a single transaction. See
// http://redis.io/topics/transactions. If an error is encountered
// in the middle of the transaction, the function will halt and return
// the error. In that case, any models which were deleted before the
// error was encountered will still be deleted. Usually this is fine
// because calling Delete on a model a second time will have no adverse
// effects.
func MDelete(models []Model) error {
	t := newTransaction()
	for _, m := range models {
		if m.GetId() == "" {
			return errors.New("zoom: cannot delete because model Id field is empty")
		}
		mr, err := newModelRefFromModel(m)
		if err != nil {
			return err
		}
		t.deleteModel(mr)
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return err
	}
	return nil
}

// DeleteById removes a model from the database by its registered name and id.
// By default modelName should be the string version of the type of model (without
// the asterisk, ampersand, or package prefix). If you used RegisterName instead of
// Register, modelName should be the custom name you used. DeleteById will throw an error
// if modelName is invalid or if there is a problem connecting to the database. If
// the model does not exist, DeleteById will not return an error; it will simply have
// no effect.
func DeleteById(modelName string, id string) error {
	t := newTransaction()
	if err := t.deleteModelById(modelName, id); err != nil {
		return err
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return err
	}
	return nil
}

// MDeleteById is like DeleteById but accepts a slice of modelNames and ids
// and deletes them all in a single transaction. See http://redis.io/topics/transactions.
// The slice of modelNames and ids should be properly aligned so that, e.g.,
// modelNames[0] corresponds to ids[0]. If there is an error in the middle of
// the transaction, the function will halt and return the error. In that case,
// any models which were deleted before the error was encountered will still be
// deleted. Usually this is fine because calling Delete on a model a second time
// will have no adverse effects.
func MDeleteById(modelNames []string, ids []string) error {
	if len(modelNames) != len(ids) {
		return errors.New("Zoom: error in MDeleteById: modelNames and ids must be the same length")
	}

	t := newTransaction()
	for i := 0; i < len(modelNames); i++ {
		name, id := modelNames[i], ids[i]
		if err := t.deleteModelById(name, id); err != nil {
			return err
		}
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return err
	}
	return nil
}
