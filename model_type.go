// Copyright 2015 Alex Browne.  All rights reserved.
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

// ModelType represents a specific registered type of model. It has methods
// for saving, finding, and deleting models of a specific type. Use the
// Register and RegisterName functions to register new types.
type ModelType struct {
	spec *modelSpec
	pool *Pool
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
// Each registered model gets a name, which is a unique string identifier
// used as a prefix when storing this type of model in the database. By
// default the name is just its type without the package prefix or dereference
// operators. So for example, the default name corresponding to *models.User
// would be "User". See RegisterName if you need to specify a custom name.
func (p *Pool) Register(model Model) (*ModelType, error) {
	defaultName := getDefaultName(reflect.TypeOf(model))
	return p.RegisterName(defaultName, model)
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
func (p *Pool) RegisterName(name string, model Model) (*ModelType, error) {
	// Make sure the name and type have not been previously registered
	typ := reflect.TypeOf(model)
	switch {
	case p.typeIsRegistered(typ):
		return nil, fmt.Errorf("zoom: Error in Register or RegisterName: The type %T has already been registered.", model)
	case p.nameIsRegistered(name):
		return nil, fmt.Errorf("zoom: Error in Register or RegisterName: The name %s has already been registered.", name)
	case !typeIsPointerToStruct(typ):
		return nil, fmt.Errorf("zoom: Register and RegisterName require a pointer to a struct as an argument. Got type %T", model)
	}

	// Compile the spec for this model and store it in the maps
	spec, err := compileModelSpec(typ)
	if err != nil {
		return nil, err
	}
	spec.name = name
	p.modelTypeToSpec[typ] = spec
	p.modelNameToSpec[name] = spec

	// Return the ModelType
	return &ModelType{
		spec: spec,
		pool: p,
	}, nil
}

func (p *Pool) typeIsRegistered(typ reflect.Type) bool {
	_, found := p.modelTypeToSpec[typ]
	return found
}

func (p *Pool) nameIsRegistered(name string) bool {
	_, found := p.modelNameToSpec[name]
	return found
}

// ModelKey returns the key that identifies a hash in the database
// which contains all the fields of the model corresponding to the given
// id. It returns an error iff id is empty.
func (mt *ModelType) ModelKey(id string) (string, error) {
	return mt.spec.modelKey(id)
}

// AllIndexKey returns the key that identifies a set in the database that
// stores all the ids for models of the given type.
func (mt *ModelType) AllIndexKey() string {
	return mt.spec.allIndexKey()
}

// FieldIndexKey returns the key for the sorted set used to index the field identified
// by fieldName. It returns an error if fieldName does not identify a field in the spec
// or if the field it identifies is not an indexed field.
func (mt *ModelType) FieldIndexKey(fieldName string) (string, error) {
	return mt.spec.fieldIndexKey(fieldName)
}

// Save writes a model (a struct which satisfies the Model interface) to the redis
// database. Save throws an error if the type of model does not match the registered
// ModelType. To make a struct satisfy the Model interface, you can embed
// zoom.RandomId, which will generate pseudo-random ids for each model.
func (mt *ModelType) Save(model Model) error {
	t := mt.pool.NewTransaction()
	t.Save(mt, model)
	if err := t.Exec(); err != nil {
		return err
	}
	return nil
}

// Save writes a model (a struct which satisfies the Model interface) to the redis
// database inside an existing transaction. save will set the err property of the
// transaction if the type of model does not matched the registered ModelType, which
// will cause exec to fail immediately and return the error. To make a struct satisfy
// the Model interface, you can embed zoom.RandomId, which will generate pseudo-random
// ids for each model. Any errors encountered will be added to the transaction and
// returned as an error when the transaction is executed.
func (t *Transaction) Save(mt *ModelType, model Model) {
	if err := mt.checkModelType(model); err != nil {
		t.setError(fmt.Errorf("zoom: Error in Save or Transaction.Save: %s", err.Error()))
		return
	}
	// Create a modelRef and start a transaction
	mr := &modelRef{
		spec:  mt.spec,
		model: model,
	}
	// Save indexes
	// This must happen first, because it relies on reading the old field values
	// from the hash for string indexes (if any)
	t.saveFieldIndexes(mr)
	// Save the model fields in a hash in the database
	hashArgs, err := mr.mainHashArgs()
	if err != nil {
		t.setError(err)
	}
	if len(hashArgs) > 1 {
		// Only save the main hash if there are any fields
		// The first element in hashArgs is the model key,
		// so there are fields if the length is greater than
		// 1.
		t.Command("HMSET", hashArgs, nil)
	}
	// Add the model id to the set of all models of this type
	t.Command("SADD", redis.Args{mt.AllIndexKey(), model.ModelId()}, nil)
}

// saveFieldIndexes adds commands to the transaction for saving the indexes
// for all indexed fields.
func (t *Transaction) saveFieldIndexes(mr *modelRef) {
	for _, fs := range mr.spec.fields {
		switch fs.indexKind {
		case noIndex:
			continue
		case numericIndex:
			t.saveNumericIndex(mr, fs)
		case booleanIndex:
			t.saveBooleanIndex(mr, fs)
		case stringIndex:
			t.saveStringIndex(mr, fs)
		}
	}
}

// saveNumericIndex adds commands to the transaction for saving a numeric
// index on the given field.
func (t *Transaction) saveNumericIndex(mr *modelRef, fs *fieldSpec) {
	fieldValue := mr.fieldValue(fs.name)
	if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
		return
	}
	score := numericScore(fieldValue)
	indexKey, err := mr.spec.fieldIndexKey(fs.name)
	if err != nil {
		t.setError(err)
	}
	t.Command("ZADD", redis.Args{indexKey, score, mr.model.ModelId()}, nil)
}

// saveBooleanIndex adds commands to the transaction for saving a boolean
// index on the given field.
func (t *Transaction) saveBooleanIndex(mr *modelRef, fs *fieldSpec) {
	fieldValue := mr.fieldValue(fs.name)
	if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
		return
	}
	score := boolScore(fieldValue)
	indexKey, err := mr.spec.fieldIndexKey(fs.name)
	if err != nil {
		t.setError(err)
	}
	t.Command("ZADD", redis.Args{indexKey, score, mr.model.ModelId()}, nil)
}

// saveStringIndex adds commands to the transaction for saving a string
// index on the given field. This includes removing the old index (if any).
func (t *Transaction) saveStringIndex(mr *modelRef, fs *fieldSpec) {
	// Remove the old index (if any)
	t.deleteStringIndex(mr.spec.name, mr.model.ModelId(), fs.redisName)
	fieldValue := mr.fieldValue(fs.name)
	for fieldValue.Kind() == reflect.Ptr {
		if fieldValue.IsNil() {
			return
		}
		fieldValue = fieldValue.Elem()
	}
	member := fieldValue.String() + nullString + mr.model.ModelId()
	indexKey, err := mr.spec.fieldIndexKey(fs.name)
	if err != nil {
		t.setError(err)
	}
	t.Command("ZADD", redis.Args{indexKey, 0, member}, nil)
}

// Find retrieves a model with the given id from redis and scans its values
// into model. model should be a pointer to a struct of a registered type
// corresponding to the ModelType. Find will mutate the struct, filling in its
// fields and overwriting any previous values. It returns an error if a model
// with the given id does not exist, if the given model was the wrong type, or
// if there was a problem connecting to the database.
func (mt *ModelType) Find(id string, model Model) error {
	t := mt.pool.NewTransaction()
	t.Find(mt, id, model)
	if err := t.Exec(); err != nil {
		return err
	}
	return nil
}

// Find retrieves a model with the given id from redis and scans its values
// into model in an existing transaction. model should be a pointer to a struct
// of a registered type corresponding to the ModelType. find will mutate the struct,
// filling in its fields and overwriting any previous values. Any errors encountered
// will be added to the transaction and returned as an error when the transaction is
// executed.
func (t *Transaction) Find(mt *ModelType, id string, model Model) {
	if err := mt.checkModelType(model); err != nil {
		t.setError(fmt.Errorf("zoom: Error in Find or Transaction.Find: %s", err.Error()))
		return
	}
	model.SetModelId(id)
	mr := &modelRef{
		spec:  mt.spec,
		model: model,
	}
	// Get the fields from the main hash for this model
	args := redis.Args{mr.key()}
	for _, fieldName := range mr.spec.fieldRedisNames() {
		args = append(args, fieldName)
	}
	t.Command("HMGET", args, newScanModelHandler(mr.spec.fieldNames(), mr))
}

// FindAll finds all the models of the given type. It executes the commands needed
// to retrieve the models in a single transaction. See http://redis.io/topics/transactions.
// models must be a pointer to a slice of models with a type corresponding to the ModelType.
// FindAll will grow or shrink the models slice as needed and if any of the models in the
// models slice are nil, FindAll will use reflection to allocate memory for them.
// FindAll returns an error if models is the wrong type or if there was a problem connecting
// to the database.
func (mt *ModelType) FindAll(models interface{}) error {
	// Since this is somewhat type-unsafe, we need to verify that
	// models is the correct type
	t := mt.pool.NewTransaction()
	t.FindAll(mt, models)
	if err := t.Exec(); err != nil {
		return err
	}
	return nil
}

// FindAll finds all the models of the given type and scans the values of the models into
// models in an existing transaction. See http://redis.io/topics/transactions.
// models must be a pointer to a slice of models with a type corresponding to the ModelType.
// findAll will grow the models slice as needed and if any of the models in the
// models slice are nil, FindAll will use reflection to allocate memory for them.
// Any errors encountered will be added to the transaction and returned as an error
// when the transaction is executed.
func (t *Transaction) FindAll(mt *ModelType, models interface{}) {
	// Since this is somewhat type-unsafe, we need to verify that
	// models is the correct type
	if err := mt.checkModelsType(models); err != nil {
		t.setError(fmt.Errorf("zoom: Error in FindAll or Transaction.FindAll: %s", err.Error()))
		return
	}
	sortArgs := mt.spec.sortArgs(mt.spec.allIndexKey(), mt.spec.fieldRedisNames(), 0, 0, ascendingOrder)
	fieldNames := append(mt.spec.fieldNames(), "-")
	t.Command("SORT", sortArgs, newScanModelsHandler(mt.spec, fieldNames, models))
}

// Count returns the number of models of the given type that exist in the database.
// It returns an error if there was a problem connecting to the database.
func (mt *ModelType) Count() (int, error) {
	t := mt.pool.NewTransaction()
	count := 0
	t.Count(mt, &count)
	if err := t.Exec(); err != nil {
		return count, err
	}
	return count, nil
}

// Count counts the number of models of the given type in the database in an existing
// transaction. It sets the value of count to the number of models. Any errors
// encountered will be added to the transaction and returned as an error when the
// transaction is executed.
func (t *Transaction) Count(mt *ModelType, count *int) {
	t.Command("SCARD", redis.Args{mt.AllIndexKey()}, newScanIntHandler(count))
}

// Delete removes the model with the given type and id from the database. It will
// not return an error if the model corresponding to the given id was not
// found in the database. Instead, it will return a boolean representing whether
// or not the model was found and deleted, and will only return an error
// if there was a problem connecting to the database.
func (mt *ModelType) Delete(id string) (bool, error) {
	t := mt.pool.NewTransaction()
	deleted := false
	t.Delete(mt, id, &deleted)
	if err := t.Exec(); err != nil {
		return deleted, err
	}
	return deleted, nil
}

// Delete removes a model with the given type and id in an existing transaction.
// deleted will be set to true iff the model was successfully deleted when the
// transaction is executed. If the no model with the given type and id existed,
// the value of deleted will be set to false. Any errors encountered will be
// added to the transaction and returned as an error when the transaction is
// executed.
func (t *Transaction) Delete(mt *ModelType, id string, deleted *bool) {
	// Delete any field indexes
	// This must happen first, because it relies on reading the old field values
	// from the hash for string indexes (if any)
	t.deleteFieldIndexes(mt, id)
	// Delete the main hash
	t.Command("DEL", redis.Args{mt.Name() + ":" + id}, newScanBoolHandler(deleted))
	// Remvoe the id from the index of all models for the given type
	t.Command("SREM", redis.Args{mt.AllIndexKey(), id}, nil)
}

// deleteFieldIndexes adds commands to the transaction for deleting the field
// indexes for all indexed fields of the given model type.
func (t *Transaction) deleteFieldIndexes(mt *ModelType, id string) {
	for _, fs := range mt.spec.fields {
		switch fs.indexKind {
		case noIndex:
			continue
		case numericIndex, booleanIndex:
			t.deleteNumericOrBooleanIndex(fs, mt.spec, id)
		case stringIndex:
			// NOTE: this invokes a lua script which is defined in scripts/delete_string_index.lua
			t.deleteStringIndex(mt.Name(), id, fs.redisName)
		}
	}
}

// deleteNumericOrBooleanIndex removes the model from a numeric or boolean index for the given
// field. I.e. it removes the model id from a sorted set.
func (t *Transaction) deleteNumericOrBooleanIndex(fs *fieldSpec, ms *modelSpec, modelId string) {
	indexKey, err := ms.fieldIndexKey(fs.name)
	if err != nil {
		t.setError(err)
	}
	t.Command("ZREM", redis.Args{indexKey, modelId}, nil)
}

// DeleteAll deletes all the models of the given type in a single transaction. See
// http://redis.io/topics/transactions. It returns the number of models deleted
// and an error if there was a problem connecting to the database.
func (mt *ModelType) DeleteAll() (int, error) {
	t := mt.pool.NewTransaction()
	count := 0
	t.DeleteAll(mt, &count)
	if err := t.Exec(); err != nil {
		return count, err
	}
	return count, nil
}

// DeleteAll delets all models for the given model type in an existing transaction.
// The value of count will be set to the number of models that were successfully deleted
// when the transaction is executed. Any errors encountered will be added to the transaction
// and returned as an error when the transaction is executed.
func (t *Transaction) DeleteAll(mt *ModelType, count *int) {
	t.deleteModelsBySetIds(mt.AllIndexKey(), mt.Name(), newScanIntHandler(count))
}

// checkModelType returns an error iff model is not of the registered type that
// corresponds to mt.
func (modelType *ModelType) checkModelType(model Model) error {
	return modelType.spec.checkModelType(model)
}

// checkModelsType returns an error iff models is not a pointer to a slice of models of the
// registered type that corresponds to modelType.
func (modelType *ModelType) checkModelsType(models interface{}) error {
	return modelType.spec.checkModelsType(models)
}
