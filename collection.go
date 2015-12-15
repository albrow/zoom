// Copyright 2015 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File collection.go contains code related to the Collection type.
// This includes all of the most basic operations like Save and Find.
// The Register method and associated methods are also included here.

package zoom

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/garyburd/redigo/redis"
)

// Collection represents a specific registered type of model. It has methods
// for saving, finding, and deleting models of a specific type. Use the
// NewCollection method to create a new collection.
type Collection struct {
	spec  *modelSpec
	pool  *Pool
	index bool
}

// CollectionOptions contains various options for a pool.
type CollectionOptions struct {
	// FallbackMarshalerUnmarshaler is used to marshal/unmarshal any type
	// into a slice of bytes which is suitable for storing in the database. If
	// Zoom does not know how to directly encode a certain type into bytes, it
	// will use the FallbackMarshalerUnmarshaler. By default, the value is
	// GobMarshalerUnmarshaler which uses the builtin gob package. Zoom also
	// provides JSONMarshalerUnmarshaler to support json encoding out of the box.
	// Default: GobMarshalerUnmarshaler.
	FallbackMarshalerUnmarshaler MarshalerUnmarshaler
	// Iff Index is true, any model in the collection that is saved will be added
	// to a set in redis which acts as an index. The default value is false. The
	// key for the set is exposed via the IndexKey method. Queries and the
	// FindAll, Count, and DeleteAll methods will not work for unindexed
	// collections. This may change in future versions. Default: false.
	Index bool
	// Name is a unique string identifier to use for the collection in redis. All
	// models in this collection that are saved in the database will use the
	// collection name as a prefix. If not provided, the default name will be the
	// name of the model type without the package prefix or pointer declarations.
	// So for example, the default name corresponding to *models.User would be
	// "User". If a custom name is provided, it cannot contain a colon.
	// Default: The name of the model type, excluding package prefix and pointer
	// declarations.
	Name string
}

// NewCollection registers and returns a new collection of the given model type.
// You must create a collection for each model type you want to save. The type
// of model must be unique, i.e., not already registered, and must be a pointer
// to a struct. To use the default options, pass in nil as the options argument.
func (p *Pool) NewCollection(model Model, options *CollectionOptions) (*Collection, error) {
	// Parse the options
	fullOptions, err := parseCollectionOptions(model, options)
	if err != nil {
		return nil, err
	}

	// Make sure the name and type have not been previously registered
	typ := reflect.TypeOf(model)
	switch {
	case p.typeIsRegistered(typ):
		return nil, fmt.Errorf("zoom: Error in NewCollection: The type %T has already been registered", model)
	case p.nameIsRegistered(fullOptions.Name):
		return nil, fmt.Errorf("zoom: Error in NewCollection: The name %s has already been registered", fullOptions.Name)
	case !typeIsPointerToStruct(typ):
		return nil, fmt.Errorf("zoom: NewCollection requires a pointer to a struct as an argument. Got type %T", model)
	}

	// Compile the spec for this model and store it in the maps
	spec, err := compileModelSpec(typ)
	if err != nil {
		return nil, err
	}
	spec.name = fullOptions.Name
	spec.fallback = fullOptions.FallbackMarshalerUnmarshaler
	p.modelTypeToSpec[typ] = spec
	p.modelNameToSpec[fullOptions.Name] = spec

	// Return the Collection
	return &Collection{
		spec:  spec,
		pool:  p,
		index: fullOptions.Index,
	}, nil
}

// Name returns the name for the given collection. The name is a unique string
// identifier to use for the collection in redis. All models in this collection
// that are saved in the database will use the collection name as a prefix.
func (c *Collection) Name() string {
	return c.spec.name
}

// parseCollectionOptions returns a well-formed CollectionOptions struct. If
// passedOptions is nil, it uses all the default options. Else, for each zero
// value field in passedOptions, it uses the default value for that field.
func parseCollectionOptions(model Model, passedOptions *CollectionOptions) (*CollectionOptions, error) {
	// If passedOptions is nil, use all the default values
	if passedOptions == nil {
		return &CollectionOptions{
			FallbackMarshalerUnmarshaler: GobMarshalerUnmarshaler,
			Name: getDefaultModelSpecName(reflect.TypeOf(model)),
		}, nil
	}
	// Copy and validate the passedOptions
	newOptions := *passedOptions
	if newOptions.Name == "" {
		newOptions.Name = getDefaultModelSpecName(reflect.TypeOf(model))
	} else if strings.Contains(newOptions.Name, ":") {
		return nil, fmt.Errorf("zoom: CollectionOptions.Name cannot contain a colon. Got: %s", newOptions.Name)
	}
	if newOptions.FallbackMarshalerUnmarshaler == nil {
		newOptions.FallbackMarshalerUnmarshaler = GobMarshalerUnmarshaler
	}
	// NOTE: we don't need to modify the Index field because the default value,
	// false, is also the zero value.
	return &newOptions, nil
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
func (c *Collection) ModelKey(id string) (string, error) {
	return c.spec.modelKey(id)
}

// IndexKey returns the key that identifies a set in the database that
// stores all the ids for models in the given collection.
func (c *Collection) IndexKey() string {
	return c.spec.indexKey()
}

// FieldIndexKey returns the key for the sorted set used to index the field
// identified by fieldName. It returns an error if fieldName does not identify a
// field in the spec or if the field it identifies is not an indexed field.
func (c *Collection) FieldIndexKey(fieldName string) (string, error) {
	return c.spec.fieldIndexKey(fieldName)
}

// newNilCollectionError returns an error with a message describing that
// methodName was called on a nil collection.
func newNilCollectionError(methodName string) error {
	return fmt.Errorf("zoom: Called %s on nil collection. You must initialize the collection with Pool.NewCollection", methodName)
}

// newUnindexedCollectionError returns an error with a message describing that
// methodName was called on a collection that was not indexed. Certain methods
// can only be called on indexed collections.
func newUnindexedCollectionError(methodName string) error {
	return fmt.Errorf("zoom: %s only works for indexed collections. To index the collection, set the Index property to true in CollectionOptions when calling Pool.NewCollection", methodName)
}

// Save writes a model (a struct which satisfies the Model interface) to the
// redis database. Save returns an error if the type of model does not match the
// registered Collection. To make a struct satisfy the Model interface, you can
// embed zoom.RandomId, which will generate pseudo-random ids for each model.
func (c *Collection) Save(model Model) error {
	t := c.pool.NewTransaction()
	t.Save(c, model)
	if err := t.Exec(); err != nil {
		return err
	}
	return nil
}

// Save writes a model (a struct which satisfies the Model interface) to the
// redis database inside an existing transaction. save will set the err property
// of the transaction if the type of model does not match the registered
// Collection, which will cause exec to fail immediately and return the error.
// To make a struct satisfy the Model interface, you can embed zoom.RandomId,
// which will generate pseudo-random ids for each model. Any errors encountered
// will be added to the transaction and returned as an error when the
// transaction is executed.
func (t *Transaction) Save(c *Collection, model Model) {
	if c == nil {
		t.setError(newNilCollectionError("Save"))
		return
	}
	if err := c.checkModelType(model); err != nil {
		t.setError(fmt.Errorf("zoom: Error in Save or Transaction.Save: %s", err.Error()))
		return
	}
	// Create a modelRef and start a transaction
	mr := &modelRef{
		spec:  c.spec,
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
	// Add the model id to the set of all models for this collection
	if c.index {
		t.Command("SADD", redis.Args{c.IndexKey(), model.ModelId()}, nil)
	}
}

// saveFieldIndexes adds commands to the transaction for saving the indexes
// for all indexed fields.
func (t *Transaction) saveFieldIndexes(mr *modelRef) {
	t.saveFieldIndexesForFields(mr.spec.fieldNames(), mr)
}

// saveFieldIndexesForFields works like saveFieldIndexes, but only saves the
// indexes for the given fieldNames.
func (t *Transaction) saveFieldIndexesForFields(fieldNames []string, mr *modelRef) {
	for _, fs := range mr.spec.fields {
		// Skip fields whose names do not appear in fieldNames.
		if !stringSliceContains(fieldNames, fs.name) {
			continue
		}
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

// UpdateFields updates only the given fields of the model. UpdateFields uses
// "last write wins" semantics. If another caller updates the the same fields
// concurrently, your updates may be overwritten. It will return an error if
// the type of model does not match the registered Collection, or if any of
// the given fieldNames are not found in the registered Collection. If
// UpdateFields is called on a model that has not yet been saved, it will not
// return an error. Instead, only the given fields will be saved in the
// database.
func (c *Collection) UpdateFields(fieldNames []string, model Model) error {
	t := c.pool.NewTransaction()
	t.UpdateFields(c, fieldNames, model)
	if err := t.Exec(); err != nil {
		return err
	}
	return nil
}

// UpdateFields updates only the given fields of the model inside an existing
// transaction. UpdateFields will set the err property of the transaction if the
// type of model does not match the registered Collection, or if any of the
// given fieldNames are not found in the model type. In either case, the
// transaction will return the error when you call Exec. UpdateFields uses "last
// write wins" semantics. If another caller updates the the same fields
// concurrently, your updates may be overwritten. If UpdateFields is called on a
// model that has not yet been saved, it will not return an error. Instead, only
// the given fields will be saved in the database.
func (t *Transaction) UpdateFields(c *Collection, fieldNames []string, model Model) {
	// Check the model type
	if err := c.checkModelType(model); err != nil {
		t.setError(fmt.Errorf("zoom: Error in UpdateFields or Transaction.UpdateFields: %s", err.Error()))
		return
	}
	// Check the given field names
	for _, fieldName := range fieldNames {
		if !stringSliceContains(c.spec.fieldNames(), fieldName) {
			t.setError(fmt.Errorf("zoom: Error in UpdateFields or Transaction.UpdateFields: Collection %s does not have field named %s", c.Name(), fieldName))
			return
		}
	}
	// Create a modelRef and start a transaction
	mr := &modelRef{
		spec:  c.spec,
		model: model,
	}
	// Update indexes
	// This must happen first, because it relies on reading the old field values
	// from the hash for string indexes (if any)
	t.saveFieldIndexesForFields(fieldNames, mr)
	// Get the main hash args.
	hashArgs, err := mr.mainHashArgsForFields(fieldNames)
	if err != nil {
		t.setError(err)
	}
	//
	if len(hashArgs) > 1 {
		// Only save the main hash if there are any fields
		// The first element in hashArgs is the model key,
		// so there are fields if the length is greater than
		// 1.
		t.Command("HMSET", hashArgs, nil)
	}
}

// Find retrieves a model with the given id from redis and scans its values
// into model. model should be a pointer to a struct of a registered type
// corresponding to the Collection. Find will mutate the struct, filling in its
// fields and overwriting any previous values. It returns an error if a model
// with the given id does not exist, if the given model was the wrong type, or
// if there was a problem connecting to the database.
func (c *Collection) Find(id string, model Model) error {
	t := c.pool.NewTransaction()
	t.Find(c, id, model)
	if err := t.Exec(); err != nil {
		return err
	}
	return nil
}

// Find retrieves a model with the given id from redis and scans its values
// into model in an existing transaction. model should be a pointer to a struct
// of a registered type corresponding to the Collection. find will mutate the struct,
// filling in its fields and overwriting any previous values. Any errors encountered
// will be added to the transaction and returned as an error when the transaction is
// executed.
func (t *Transaction) Find(c *Collection, id string, model Model) {
	if c == nil {
		t.setError(newNilCollectionError("Find"))
		return
	}
	if err := c.checkModelType(model); err != nil {
		t.setError(fmt.Errorf("zoom: Error in Find or Transaction.Find: %s", err.Error()))
		return
	}
	model.SetModelId(id)
	mr := &modelRef{
		spec:  c.spec,
		model: model,
	}
	// Get the fields from the main hash for this model
	args := redis.Args{mr.key()}
	for _, fieldName := range mr.spec.fieldRedisNames() {
		args = append(args, fieldName)
	}
	t.Command("HMGET", args, newScanModelHandler(mr.spec.fieldNames(), mr))
}

// FindFields is like Find but finds and sets only the specified fields. Any
// fields of the model which are not in the given fieldNames are not mutated.
// FindFields will return an error if any of the given fieldNames are not found
// in the model type.
func (c *Collection) FindFields(id string, fieldNames []string, model Model) error {
	t := c.pool.NewTransaction()
	t.FindFields(c, id, fieldNames, model)
	if err := t.Exec(); err != nil {
		return err
	}
	return nil
}

// FindFields is like Find but finds and sets only the specified fields. Any
// fields of the model which are not in the given fieldNames are not mutated.
// FindFields will return an error if any of the given fieldNames are not found
// in the model type.
func (t *Transaction) FindFields(c *Collection, id string, fieldNames []string, model Model) {
	if err := c.checkModelType(model); err != nil {
		t.setError(fmt.Errorf("zoom: Error in FindFields or Transaction.FindFields: %s", err.Error()))
		return
	}
	// Set the model id and create a modelRef
	model.SetModelId(id)
	mr := &modelRef{
		spec:  c.spec,
		model: model,
	}
	// Check the given field names and append the corresponding redis field names
	// to args.
	args := redis.Args{mr.key()}
	for _, fieldName := range fieldNames {
		if !stringSliceContains(c.spec.fieldNames(), fieldName) {
			t.setError(fmt.Errorf("zoom: Error in FindFields or Transaction.FindFields: Collection %s does not have field named %s", c.Name(), fieldName))
			return
		}
		// args is an array of arguments passed to the HMGET command. We want to
		// use the redis names corresponding to each field name. The redis names
		// may be customized via struct tags.
		args = append(args, c.spec.fieldsByName[fieldName].redisName)
	}
	// Get the fields from the main hash for this model
	t.Command("HMGET", args, newScanModelHandler(fieldNames, mr))
}

// FindAll finds all the models of the given type. It executes the commands needed
// to retrieve the models in a single transaction. See http://redis.io/topics/transactions.
// models must be a pointer to a slice of models with a type corresponding to the Collection.
// FindAll will grow or shrink the models slice as needed and if any of the models in the
// models slice are nil, FindAll will use reflection to allocate memory for them.
// FindAll returns an error if models is the wrong type or if there was a problem connecting
// to the database.
func (c *Collection) FindAll(models interface{}) error {
	// Since this is somewhat type-unsafe, we need to verify that
	// models is the correct type
	t := c.pool.NewTransaction()
	t.FindAll(c, models)
	if err := t.Exec(); err != nil {
		return err
	}
	return nil
}

// FindAll finds all the models of the given type and scans the values of the models into
// models in an existing transaction. See http://redis.io/topics/transactions.
// models must be a pointer to a slice of models with a type corresponding to the Collection.
// findAll will grow the models slice as needed and if any of the models in the
// models slice are nil, FindAll will use reflection to allocate memory for them.
// Any errors encountered will be added to the transaction and returned as an error
// when the transaction is executed.
func (t *Transaction) FindAll(c *Collection, models interface{}) {
	if c == nil {
		t.setError(newNilCollectionError("FindAll"))
		return
	}
	if !c.index {
		t.setError(newUnindexedCollectionError("FindAll"))
		return
	}
	// Since this is somewhat type-unsafe, we need to verify that
	// models is the correct type
	if err := c.checkModelsType(models); err != nil {
		t.setError(fmt.Errorf("zoom: Error in FindAll or Transaction.FindAll: %s", err.Error()))
		return
	}
	sortArgs := c.spec.sortArgs(c.spec.indexKey(), c.spec.fieldRedisNames(), 0, 0, ascendingOrder)
	fieldNames := append(c.spec.fieldNames(), "-")
	t.Command("SORT", sortArgs, newScanModelsHandler(c.spec, fieldNames, models))
}

// Count returns the number of models of the given type that exist in the database.
// It returns an error if there was a problem connecting to the database.
func (c *Collection) Count() (int, error) {
	t := c.pool.NewTransaction()
	count := 0
	t.Count(c, &count)
	if err := t.Exec(); err != nil {
		return count, err
	}
	return count, nil
}

// Count counts the number of models of the given type in the database in an existing
// transaction. It sets the value of count to the number of models. Any errors
// encountered will be added to the transaction and returned as an error when the
// transaction is executed.
func (t *Transaction) Count(c *Collection, count *int) {
	if c == nil {
		t.setError(newNilCollectionError("Count"))
		return
	}
	if !c.index {
		t.setError(newUnindexedCollectionError("Count"))
		return
	}
	t.Command("SCARD", redis.Args{c.IndexKey()}, newScanIntHandler(count))
}

// Delete removes the model with the given type and id from the database. It will
// not return an error if the model corresponding to the given id was not
// found in the database. Instead, it will return a boolean representing whether
// or not the model was found and deleted, and will only return an error
// if there was a problem connecting to the database.
func (c *Collection) Delete(id string) (bool, error) {
	t := c.pool.NewTransaction()
	deleted := false
	t.Delete(c, id, &deleted)
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
// executed. You may pass in nil for deleted if you do not care whether or not
// the model was deleted.
func (t *Transaction) Delete(c *Collection, id string, deleted *bool) {
	if c == nil {
		t.setError(newNilCollectionError("Delete"))
		return
	}
	// Delete any field indexes
	// This must happen first, because it relies on reading the old field values
	// from the hash for string indexes (if any)
	t.deleteFieldIndexes(c, id)
	var handler ReplyHandler
	if deleted == nil {
		handler = nil
	} else {
		handler = newScanBoolHandler(deleted)
	}
	// Delete the main hash
	t.Command("DEL", redis.Args{c.Name() + ":" + id}, handler)
	// Remvoe the id from the index of all models for the given type
	t.Command("SREM", redis.Args{c.IndexKey(), id}, nil)
}

// deleteFieldIndexes adds commands to the transaction for deleting the field
// indexes for all indexed fields of the given model type.
func (t *Transaction) deleteFieldIndexes(c *Collection, id string) {
	for _, fs := range c.spec.fields {
		switch fs.indexKind {
		case noIndex:
			continue
		case numericIndex, booleanIndex:
			t.deleteNumericOrBooleanIndex(fs, c.spec, id)
		case stringIndex:
			// NOTE: this invokes a lua script which is defined in scripts/delete_string_index.lua
			t.deleteStringIndex(c.Name(), id, fs.redisName)
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
func (c *Collection) DeleteAll() (int, error) {
	t := c.pool.NewTransaction()
	count := 0
	t.DeleteAll(c, &count)
	if err := t.Exec(); err != nil {
		return count, err
	}
	return count, nil
}

// DeleteAll delets all models for the given model type in an existing transaction.
// The value of count will be set to the number of models that were successfully deleted
// when the transaction is executed. Any errors encountered will be added to the transaction
// and returned as an error when the transaction is executed. You may pass in nil
// for count if you do not care about the number of models that were deleted.
func (t *Transaction) DeleteAll(c *Collection, count *int) {
	if c == nil {
		t.setError(newNilCollectionError("DeleteAll"))
		return
	}
	if !c.index {
		t.setError(newUnindexedCollectionError("DeleteAll"))
		return
	}
	var handler ReplyHandler
	if count == nil {
		handler = nil
	} else {
		handler = newScanIntHandler(count)
	}
	t.deleteModelsBySetIds(c.IndexKey(), c.Name(), handler)
}

// checkModelType returns an error iff model is not of the registered type that
// corresponds to c.
func (c *Collection) checkModelType(model Model) error {
	return c.spec.checkModelType(model)
}

// checkModelsType returns an error iff models is not a pointer to a slice of models of the
// registered type that corresponds to the collection.
func (c *Collection) checkModelsType(models interface{}) error {
	return c.spec.checkModelsType(models)
}
