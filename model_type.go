// Copyright 2014 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File model_type.go contains code related to the ModelType type.
// This includes all of the most basic operations like Save and Find.
// The Register method and associated methods are also included here.

package zoom

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"reflect"
	"strings"
)

var (
	// modelTypeToSpec maps a registered model type to a modelSpec
	modelTypeToSpec map[reflect.Type]*modelSpec = map[reflect.Type]*modelSpec{}
	// modelNameToSpec maps a registered model name to a modelSpec
	modelNameToSpec map[string]*modelSpec = map[string]*modelSpec{}
)

type ModelType struct {
	spec *modelSpec
}

// Name returns the name for the given ModelType. The name is a unique
// string identifier which is used as a prefix when storing this type of
// model in the database.
func (mt *ModelType) Name() string {
	return mt.spec.name
}

// Register adds a model type to the list of registered types. Any model
// you wish to save must be registered first. The type of model must be
// unique, i.e., not already registered, and must be a pointer to a struct.
// Each registered model gets a name, a unique string identifier, which is
// used as a prefix when storing this type of model in the database. By
// default the name is just its type without the package prefix or dereference
// operators. So for example, the default name corresponding to *models.User
// would be "User". See RegisterName if you need to specify a custom name.
func Register(model Model) (*ModelType, error) {
	defaultName := getDefaultName(reflect.TypeOf(model))
	return RegisterName(defaultName, model)
}

// getDefaultName returns the default name for the given type, which is
// simply the name of the type without the package prefix or dereference
// operators.
func getDefaultName(typ reflect.Type) string {
	// Strip any dereference operators
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	nameWithPackage := typ.String()
	// Strip the package name
	return strings.Join(strings.Split(nameWithPackage, ".")[1:], "")
}

// RegisterName is like Register but allows you to specify a custom
// name to use for the model type. The custom name will be used as
// a prefix for all models of this type that are stored in the
// database. Both the name and the model must be unique, i.e., not
// already registered. The type of model must be a pointer to a struct.
func RegisterName(name string, model Model) (*ModelType, error) {
	// Make sure the name and type have not been previously registered
	typ := reflect.TypeOf(model)
	if _, found := modelTypeToSpec[typ]; found {
		return nil, NewTypeAlreadyRegisteredError(typ)
	} else if _, found := modelNameToSpec[name]; found {
		return nil, NewNameAlreadyRegisteredError(name)
	} else if !typeIsPointerToStruct(typ) {
		return nil, fmt.Errorf("zoom: Register and RegisterName require a pointer to a struct as an argument. Got type %T", model)
	}

	// Compile the spec for this model and store it in the maps
	spec, err := compileModelSpec(typ)
	if err != nil {
		return nil, err
	}
	spec.name = name
	modelTypeToSpec[typ] = spec
	modelNameToSpec[name] = spec

	// Return the ModelType
	return &ModelType{spec}, nil
}

// KeyForModel returns the key that identifies a hash in the database
// which contains all the fields of the given model. It returns an error
// iff the model does not have an id.
func (mt *ModelType) KeyForModel(model Model) (string, error) {
	if model.GetId() == "" {
		return "", fmt.Errorf("zoom: Error in KeyForModel: model does not have an id and therefore cannot have a valid key")
	}
	return mt.Name() + ":" + model.GetId(), nil
}

// KeyForAll returns the key that identifies a set in the database that
// stores all the ids for models of the given type.
func (mt *ModelType) KeyForAll() string {
	return mt.Name() + ":all"
}

// Save writes a model (a struct which satisfies the Model interface) to the redis
// database. Save throws an error if the type of model does not match the registered
// ModelType. If the Id field of the struct is empty, Save will mutate the struct by
// setting the Id. To make a struct satisfy the Model interface, you can embed
// zoom.DefaultData.
func (mt *ModelType) Save(model Model) error {
	t := newTransaction()
	t.save(mt, model)
	if err := t.exec(); err != nil {
		return err
	}
	return nil
}

// MSave is like Save but accepts a slice of models and saves them all in
// a single transaction. See http://redis.io/topics/transactions. If there
// is an error in the middle of the transaction, any models that were saved
// before the error was encountered will still be saved.
func (mt *ModelType) MSave(models []Model) error {
	t := newTransaction()
	for _, model := range models {
		t.save(mt, model)
	}
	if err := t.exec(); err != nil {
		return err
	}
	return nil
}

// save writes a model (a struct which satisfies the Model interface) to the redis
// database inside an existing transaction. save will set the err property of the
// transaction if the type of model does not matched the registered ModelType, which
// will cause exec to fail immediately and return the error. If the Id field of the
// struct is empty, save will mutate the struct by setting the Id. To make a struct
// satisfy the Model interface, you can embed zoom.DefaultData.
func (t *transaction) save(mt *ModelType, model Model) {
	// Generate id if needed
	if model.GetId() == "" {
		model.SetId(generateRandomId())
	}

	// Create a modelRef and start a transaction
	mr := &modelRef{
		spec:  mt.spec,
		model: model,
	}

	// Save the model fields in a hash in the database
	hashArgs, err := mr.mainHashArgs()
	if err != nil {
		t.setError(err)
	}
	t.command("HMSET", hashArgs, nil)

	// Add the model id to the set of all models of this type
	t.command("SADD", redis.Args{mt.KeyForAll(), model.GetId()}, nil)

	// TODO: save indexes
}

// Find retrieves a model with the given id from redis and scans its values
// into model. model should be a pointer to a struct of a registered type
// corresponding to the ModelType. Find will mutate the struct, filling in its
// fields and overwriting any previous values. It returns an error if a model
// with the given id does not exist, if the given model was the wrong type, or
// if there was a problem connecting to the database.
func (mt *ModelType) Find(id string, model Model) error {
	t := newTransaction()
	t.find(mt, id, model)
	if err := t.exec(); err != nil {
		return err
	}
	return nil
}

// MFind is like Find but accepts a slice of ids and a pointer to
// a slice of models. It executes the commands needed to retrieve the models
// in a single transaction. See http://redis.io/topics/transactions. models must
// be a pointer to a slice of models with a type corresponding to the ModelType.
// MFind will grow the models slice as needed and if any of the models in the
// models slice are nil, MFind will use reflection to allocate memory for them.
// MFind returns an error if the model corresponding to any of the given ids did
// not exist, if models is the wrong type, or if there was a problem connecting
// to the database.
func (mt *ModelType) MFind(ids []string, models interface{}) error {
	// Since this is somewhat type-unsafe, we need to verify that
	// models is the correct type
	if reflect.TypeOf(models).Kind() != reflect.Ptr {
		return fmt.Errorf("Zoom: error in MScanById: models should be a pointer to a slice or array of models")
	}
	modelsVal := reflect.ValueOf(models).Elem()
	modelType := modelsVal.Type().Elem()
	if !typeIsSliceOrArray(modelsVal.Type()) {
		return fmt.Errorf("Zoom: error in MScanById: models should be a pointer to a slice or array of models")
	} else if !typeIsPointerToStruct(modelType) {
		return fmt.Errorf("Zoom: error in MScanById: the elements in models should be pointers to structs")
	} else if _, found := modelTypeToSpec[modelType]; !found {
		return fmt.Errorf("Zoom: error in MScanById: the elements in models should be of a registered type\nType %s has not been registered.", modelType.String())
	}

	// Start a new transaction and add an action to find the model for each id
	t := newTransaction()
	for i, id := range ids {
		var modelVal reflect.Value
		if modelsVal.Len() > i {
			// Use the pre-existing value at index i
			modelVal = modelsVal.Index(i)
			if modelVal.IsNil() {
				// If the value is nil, allocate space for it
				modelsVal.Index(i).Set(reflect.New(mt.spec.typ.Elem()))
			}
		} else {
			// Index i is out of range of the existing slice. Create a
			// new modelVal and append it to modelsVal
			modelVal = reflect.New(mt.spec.typ.Elem())
			modelsVal.Set(reflect.Append(modelsVal, modelVal))
		}
		t.find(mt, id, modelVal.Interface().(Model))
	}
	// Trim the length in case the original slice had a length greater
	// than the number of models
	modelsVal.SetLen(len(ids))

	// Execute the transaction
	if err := t.exec(); err != nil {
		return err
	}
	return nil
}

// FindAll finds all the models of the given type. It executes the commands needed
// to retrieve the models in a single transaction. See http://redis.io/topics/transactions.
// models must be a pointer to a slice of models with a type corresponding to the ModelType.
// FindAll will grow the models slice as needed and if any of the models in the
// models slice are nil, FindAll will use reflection to allocate memory for them.
// FindAll returns an error if models is the wrong type or if there was a problem connecting
// to the database.
func (mt *ModelType) FindAll(models interface{}) error {
	// TODO: Use a lua script to get the fields for all models corresponding to
	// the ids in the set of all ids.
	return fmt.Errorf("FindAll not yet implemented!")
}

// find retrieves a model with the given id from redis and scans its values
// into model in an existing transaction. model should be a pointer to a struct
// of a registered type corresponding to the ModelType. find will mutate the struct,
// filling in its fields and overwriting any previous values. If a model
// with the given id does not exist, the given model was the wrong type, or
// there was a problem connecting to the database, find will set the error field
// of the transaction, which will call exec to fail immediately and return the error.
func (t *transaction) find(mt *ModelType, id string, model Model) {
	model.SetId(id)

	// Create a modelRef and start a transaction
	mr := &modelRef{
		spec:  mt.spec,
		model: model,
	}

	// Get the fields from the main hash for this model
	t.command("HGETALL", redis.Args{mr.key()}, newScanModelHandler(mr))
}

// Count returns the number of models of the given type that exist in the database.
// It returns an error if there was a problem connecting to the database.
func (mt *ModelType) Count() (int, error) {
	return 0, fmt.Errorf("Count not yet implemented!")
}

// Delete removes the model with the given type and id from the database. It will
// not return an error if the model corresponding to the given id was not
// found in the database. Instead, it will return a boolean representing whether
// or not the model was found and deleted, and will only return an error
// if there was a problem connecting to the database.
func (mt *ModelType) Delete(id string) (bool, error) {
	return false, fmt.Errorf("Delete not yet implemented!")
}

// MDelete is like Delete but accepts a slice of ids and deletes all the
// corresponding models in a single transaction. See http://redis.io/topics/transactions.
// MDelete will not return an error if it can't find a model corresponding
// to a given id. It return the number of models deleted and an error if there
// was a problem connecting to the database.
func (mt *ModelType) MDelete(ids []string) (int, error) {
	return 0, fmt.Errorf("MDelete not yet implemented!")
}

// DeleteAll all the models of the given type in a single transaction. See
// http://redis.io/topics/transactions. It returns the number of models deleted
// and an error if there was a problem connecting to the database.
func (mt *ModelType) DeleteAll() (int, error) {
	return 0, fmt.Errorf("DeleteAll not yet implemented!")
}
