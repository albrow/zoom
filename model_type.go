// Copyright 2014 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File model_type.go contains code related to the ModelType type.
// This includes all of the most basic operations like Save and Find.
// The Register method and associated methods are also included here.

package zoom

import (
	"fmt"
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

// Save writes a model (a struct which satisfies the Model interface) to the redis
// database. Save throws an error if the type of model does not match the registered
// ModelType. If the Id field of the struct is empty, Save will mutate the struct by
// setting the Id. To make a struct satisfy the Model interface, you can embed
// zoom.DefaultData.
func (mt *ModelType) Save(model Model) error {
	return fmt.Errorf("Save not yet implemented!")
}

// MSave is like Save but accepts a slice of models and saves them all in
// a single transaction. See http://redis.io/topics/transactions. If there
// is an error in the middle of the transaction, any models that were saved
// before the error was encountered will still be saved.
func (mt *ModelType) MSave(models []Model) error {
	return fmt.Errorf("MSave not yet implemented!")
}

// Find gets a model from the database. It returns an error
// if a model with that id does not exist or if there was a problem
// connecting to the database.
func (mt *ModelType) Find(id string) (Model, error) {
	return nil, fmt.Errorf("Find not yet implemented!")
}

// Scan retrieves a model from redis and scans its values into model.
// model should be a pointer to a struct of a registered type. Scan
// will mutate the struct, filling in its fields. It returns an error
// if a model with the given id does not exist or if there was a problem
// connecting to the database.
func (mt *ModelType) Scan(id string, model Model) error {
	return fmt.Errorf("Scan not yet implemented!")
}

// MFind is like Find but accepts a slice of ids and returns a slice of models.
// It executes the commands needed to retrieve the models in a single transaction.
// See http://redis.io/topics/transactions.If there is an error in the middle of
// the transaction, the function will halt and return the models retrieved so
// far (as well as the error).
func (mt *ModelType) MFind(ids []string) ([]Model, error) {
	return nil, fmt.Errorf("MFind not yet implemented!")
}

// MScan is like Scan but accepts a slice of ids and a pointer to
// a slice of models. It executes the commands needed to retrieve the models
// in a single transaction. See http://redis.io/topics/transactions.
// The slice of ids and models should be properly aligned so that, e.g.,
// ids[0] corresponds to models[0]. If there is an error in the middle of the
// transaction, the function will halt and return the error. Any models that
// were scanned before the error will still be valid. If any of the models in
// the models slice are nil, MScan will use reflection to allocate memory
// for them.
func (mt *ModelType) MScan(ids []string, models interface{}) error {
	return fmt.Errorf("MScan not yet implemented!")
}

// Delete removes the model with the given id from the database. It
// will throw an error if the id is empty or if there is a problem
// connecting to the database. If the model does not exist in the
// database, Delete will not return an error; it will simply have no
// effect.
func (mt *ModelType) Delete(id string) error {
	return fmt.Errorf("Delete not yet implemented!")
}

// MDelete is like Delete but accepts a slice of ids and deletes all the
// corresponding models in a single transaction. See http://redis.io/topics/transactions.
// If there is an error in the middle of the transaction, the function will halt
// and return the error. In that case, any models which were deleted before the error
// was encountered will still be deleted.
func (mt *ModelType) MDelete(ids []string) error {
	return fmt.Errorf("MDelete not yet implemented!")
}
