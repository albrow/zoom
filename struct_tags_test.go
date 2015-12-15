// Copyright 2015 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File struct_tags_test.go tests the different options
// that may be provided in model type declarations using
// struct tags.

package zoom

import (
	"testing"

	"github.com/garyburd/redigo/redis"
)

// Test that the redis ignore struct tag causes a field to be ignored
func TestRedisIgnoreOption(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	type ignoredFieldModel struct {
		Attr string `redis:"-"`
		RandomId
	}
	ignoredFieldModels, err := testPool.NewCollection(&ignoredFieldModel{}, nil)
	if err != nil {
		t.Errorf("Unexpected error in Register: %s", err)
	}

	// check the spec
	spec, found := testPool.modelNameToSpec["ignoredFieldModel"]
	if !found {
		t.Error("Could not find spec for model name ignoredFieldModel")
	}
	if fs, found := spec.fieldsByName["Attr"]; found {
		t.Errorf("Expected to not find the Attr field in the spec, but found: %v", fs)
	}

	// save a new model
	model := &ignoredFieldModel{
		Attr: "this should be ignored",
	}
	if err := ignoredFieldModels.Save(model); err != nil {
		t.Errorf("Unexpected error in Save: %s", err.Error())
	}

	// Check the database to make sure the field is not there
	conn := testPool.NewConn()
	defer conn.Close()
	key, _ := ignoredFieldModels.ModelKey(model.ModelId())
	gotAttr, err := redis.String(conn.Do("HGET", key, "Attr"))
	if err != nil && err != redis.ErrNil {
		t.Errorf("Unexpected error in HGET command: %s", err.Error())
	}
	if gotAttr != "" {
		t.Errorf("Expected empty attr but got: %s", gotAttr)
	}
}

// Test that the redis name struct tag causes a field's name in redis to be changed
func TestRedisNameOption(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	type customFieldModel struct {
		Attr string `redis:"a"`
		RandomId
	}
	customFieldModels, err := testPool.NewCollection(&customFieldModel{}, nil)
	if err != nil {
		t.Errorf("Unexpected error in Register: %s", err.Error())
	}

	// check the spec
	spec, found := testPool.modelNameToSpec["customFieldModel"]
	if !found {
		t.Error("Could not find spec for model name customFieldModel")
	}
	if fs, found := spec.fieldsByName["Attr"]; !found {
		t.Error("Expected to find Attr field in the spec, but got nil")
	} else if fs.redisName != "a" {
		t.Errorf("Expected fs.redisName to be `a` but got %s", fs.redisName)
	}

	// save a new model and check redis
	model := &customFieldModel{
		Attr: "test",
	}
	if err := customFieldModels.Save(model); err != nil {
		t.Errorf("Unexpected error in Save: %s", err.Error())
	}
	modelKey, _ := customFieldModels.ModelKey(model.ModelId())
	expectFieldEquals(t, modelKey, "a", customFieldModels.spec.fallback, "test")
}

func TestInvalidOptionThrowsError(t *testing.T) {
	testingSetUp()
	testingTearDown()

	type invalid struct {
		Attr string `zoom:"index,poop"`
		RandomId
	}
	if _, err := testPool.NewCollection(&invalid{}, nil); err == nil {
		t.Error("Expected error when registering struct with invalid tag")
	}
}

// Test that the indexes are actually created in redis for a model with all
// the different indexed primative fields
func TestSaveIndexedPrimativesModel(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// Create and save a new model with random primative fields
	model := createIndexedPrimativesModel()
	if err := indexedPrimativesModels.Save(model); err != nil {
		t.Fatalf("Unexpected error in Save: %s", err.Error())
	}

	// Iterate through each field using reflection and validate that the index was set properly
	numFields := indexedPrimativesModels.spec.typ.Elem().NumField()
	for i := 0; i < numFields; i++ {
		field := indexedPrimativesModels.spec.typ.Elem().Field(i)
		if field.Anonymous {
			continue // Skip embedded structs
		}
		expectIndexExists(t, indexedPrimativesModels, model, field.Name)
	}
}

// Test that the indexes are actually created in redis for a model with all
// the different indexed pointer to primative fields
func TestSaveIndexedPointersModel(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// Create and save a new model with random pointer to primative fields
	model := createIndexedPointersModel()
	if err := indexedPointersModels.Save(model); err != nil {
		t.Fatalf("Unexpected error in Save: %s", err.Error())
	}

	// Iterate through each field using reflection and validate that the index was set properly
	numFields := indexedPointersModels.spec.typ.Elem().NumField()
	for i := 0; i < numFields; i++ {
		field := indexedPointersModels.spec.typ.Elem().Field(i)
		if field.Anonymous {
			continue // Skip embedded structs
		}
		expectIndexExists(t, indexedPointersModels, model, field.Name)
	}
}

// Test that the indexes are removed from redis after a model with primative indexes is deleted
func TestDeleteIndexedPrimativesModel(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// Create and save a new model with random primative fields
	model := createIndexedPrimativesModel()
	if err := indexedPrimativesModels.Save(model); err != nil {
		t.Fatalf("Unexpected error in Save: %s", err.Error())
	}
	if _, err := indexedPrimativesModels.Delete(model.ModelId()); err != nil {
		t.Fatalf("Unexpected error in Delete: %s", err.Error())
	}

	// Iterate through each field using reflection and validate that the index was set properly
	numFields := indexedPrimativesModels.spec.typ.Elem().NumField()
	for i := 0; i < numFields; i++ {
		field := indexedPrimativesModels.spec.typ.Elem().Field(i)
		if field.Anonymous {
			continue // Skip embedded structs
		}
		expectIndexDoesNotExist(t, indexedPrimativesModels, model, field.Name)
	}
}

// Test that the indexes are removed from redis after a model with indexed pointer to primative
// fields is deleted
func TestDeleteIndexedPointersModel(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// Create and save a new model with random pointer to primative fields
	model := createIndexedPointersModel()
	if err := indexedPointersModels.Save(model); err != nil {
		t.Fatalf("Unexpected error in Save: %s", err.Error())
	}
	if _, err := indexedPointersModels.Delete(model.ModelId()); err != nil {
		t.Fatalf("Unexpected error in Delete: %s", err.Error())
	}

	// Iterate through each field using reflection and validate that the index was set properly
	numFields := indexedPointersModels.spec.typ.Elem().NumField()
	for i := 0; i < numFields; i++ {
		field := indexedPointersModels.spec.typ.Elem().Field(i)
		if field.Anonymous {
			continue // Skip embedded structs
		}
		expectIndexDoesNotExist(t, indexedPointersModels, model, field.Name)
	}
}

// Test that the indexes are actually created in redis for a model with all
// the different indexed primative fields
func TestIndexAndCustomName(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	type customIndexModel struct {
		Int    int    `zoom:"index" redis:"integer"`
		String string `zoom:"index" redis:"str"`
		Bool   bool   `zoom:"index" redis:"boolean"`
		RandomId
	}
	customIndexModels, err := testPool.NewCollection(&customIndexModel{}, nil)
	if err != nil {
		t.Fatalf("Unexpected error in Register: %s", err.Error())
	}
	model := &customIndexModel{
		Int:    randomInt(),
		String: randomString(),
		Bool:   randomBool(),
	}
	if err := customIndexModels.Save(model); err != nil {
		t.Fatalf("Unexpected error in Save: %s", err.Error())
	}

	// Iterate through each field using reflection and validate that the index was set properly
	numFields := customIndexModels.spec.typ.Elem().NumField()
	for i := 0; i < numFields; i++ {
		field := customIndexModels.spec.typ.Elem().Field(i)
		if field.Anonymous {
			continue // Skip embedded structs
		}
		expectIndexExists(t, customIndexModels, model, field.Name)
	}
}
