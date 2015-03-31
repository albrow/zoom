// Copyright 2014 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File struct_tags_test.go tests the different options
// that may be provided in model type declarations using
// struct tags.

package zoom

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"testing"
)

// Test that the redis ignore struct tag causes a field to be ignored
func TestRedisIgnoreOption(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	type ignoredFieldModel struct {
		Attr string `redis:"-"`
		DefaultData
	}
	ignoredFieldModels, err := Register(&ignoredFieldModel{})
	if err != nil {
		t.Errorf("Unexpected error in Register: %s", err)
	}

	// check the spec
	spec, found := modelNameToSpec["ignoredFieldModel"]
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
	conn := GetConn()
	defer conn.Close()
	key, _ := ignoredFieldModels.KeyForModel(model)
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
		DefaultData
	}
	customFieldModels, err := Register(&customFieldModel{})
	if err != nil {
		t.Errorf("Unexpected error in Register: %s", err.Error())
	}

	// check the spec
	spec, found := modelNameToSpec["customFieldModel"]
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
	modelKey, _ := customFieldModels.KeyForModel(model)
	expectFieldEquals(t, modelKey, "a", "test")
}

func TestInvalidOptionThrowsError(t *testing.T) {
	testingSetUp()
	testingTearDown()

	type invalid struct {
		Attr string `zoom:"index,poop"`
		DefaultData
	}
	if _, err := Register(&invalid{}); err == nil {
		t.Error("Expected error when registering struct with invalid tag")
	}
}

var (
	indexedPrimativesModels *ModelType
	indexedPointersModels   *ModelType
)

type indexedPrimativesModel struct {
	Uint    uint    `zoom:"index"`
	Uint8   uint8   `zoom:"index"`
	Uint16  uint16  `zoom:"index"`
	Uint32  uint32  `zoom:"index"`
	Uint64  uint64  `zoom:"index"`
	Int     int     `zoom:"index"`
	Int8    int8    `zoom:"index"`
	Int16   int16   `zoom:"index"`
	Int32   int32   `zoom:"index"`
	Int64   int64   `zoom:"index"`
	Float32 float32 `zoom:"index"`
	Float64 float64 `zoom:"index"`
	Byte    byte    `zoom:"index"`
	Rune    rune    `zoom:"index"`
	String  string  `zoom:"index"`
	Bool    bool    `zoom:"index"`
	DefaultData
}

type indexedPointersModel struct {
	Uint    *uint    `zoom:"index"`
	Uint8   *uint8   `zoom:"index"`
	Uint16  *uint16  `zoom:"index"`
	Uint32  *uint32  `zoom:"index"`
	Uint64  *uint64  `zoom:"index"`
	Int     *int     `zoom:"index"`
	Int8    *int8    `zoom:"index"`
	Int16   *int16   `zoom:"index"`
	Int32   *int32   `zoom:"index"`
	Int64   *int64   `zoom:"index"`
	Float32 *float32 `zoom:"index"`
	Float64 *float64 `zoom:"index"`
	Byte    *byte    `zoom:"index"`
	Rune    *rune    `zoom:"index"`
	String  *string  `zoom:"index"`
	Bool    *bool    `zoom:"index"`
	DefaultData
}

// registerIndexedPrimativesModel registers the indexedPrimativesModel type and sets the value
// of indexedPrimativesModels the first time it is called. Successive calls have no effect.
func registerIndexedPrimativesModel() {
	if indexedPrimativesModels == nil {
		var err error
		indexedPrimativesModels, err = Register(&indexedPrimativesModel{})
		if err != nil {
			msg := fmt.Sprintf("Unexpected error in Register: %s", err.Error())
			panic(msg)
		}
	}
}

// registerIndexedPointersModel registers the indexedPointersModel type and sets the value
// of indexedPointersModels the first time it is called. Successive calls have no effect.
func registerIndexedPointersModel() {
	if indexedPointersModels == nil {
		var err error
		indexedPointersModels, err = Register(&indexedPointersModel{})
		if err != nil {
			msg := fmt.Sprintf("Unexpected error in Register: %s", err.Error())
			panic(msg)
		}
	}
}

// createIndexedPrimativesModel instantiates and returns an indexedPrimativesModel with
// random values for all fields.
func createIndexedPrimativesModel() *indexedPrimativesModel {
	return &indexedPrimativesModel{
		Uint:    uint(randomInt()),
		Uint8:   uint8(randomInt()),
		Uint16:  uint16(randomInt()),
		Uint32:  uint32(randomInt()),
		Uint64:  uint64(randomInt()),
		Int:     randomInt(),
		Int8:    int8(randomInt()),
		Int16:   int16(randomInt()),
		Int32:   int32(randomInt()),
		Int64:   int64(randomInt()),
		Float32: float32(randomInt()),
		Float64: float64(randomInt()),
		Byte:    []byte(randomString())[0],
		Rune:    []rune(randomString())[0],
		String:  randomString(),
		Bool:    randomBool(),
	}
}

// createIndexedPointersModel instantiates and returns an indexedPointersModel with
// random values for all fields.
func createIndexedPointersModel() *indexedPointersModel {
	Uint := uint(randomInt())
	Uint8 := uint8(randomInt())
	Uint16 := uint16(randomInt())
	Uint32 := uint32(randomInt())
	Uint64 := uint64(randomInt())
	Int := randomInt()
	Int8 := int8(randomInt())
	Int16 := int16(randomInt())
	Int32 := int32(randomInt())
	Int64 := int64(randomInt())
	Float32 := float32(randomInt())
	Float64 := float64(randomInt())
	Byte := []byte(randomString())[0]
	Rune := []rune(randomString())[0]
	String := randomString()
	Bool := randomBool()
	return &indexedPointersModel{
		Uint:    &Uint,
		Uint8:   &Uint8,
		Uint16:  &Uint16,
		Uint32:  &Uint32,
		Uint64:  &Uint64,
		Int:     &Int,
		Int8:    &Int8,
		Int16:   &Int16,
		Int32:   &Int32,
		Int64:   &Int64,
		Float32: &Float32,
		Float64: &Float64,
		Byte:    &Byte,
		Rune:    &Rune,
		String:  &String,
		Bool:    &Bool,
	}
}

// Test that the indexes are actually created in redis for a model with all
// the different indexed primative fields
func TestSaveIndexedPrimativesModel(t *testing.T) {
	testingSetUp()
	defer testingTearDown()
	registerIndexedPrimativesModel()

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
	registerIndexedPointersModel()

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
	registerIndexedPrimativesModel()

	// Create and save a new model with random primative fields
	model := createIndexedPrimativesModel()
	if err := indexedPrimativesModels.Save(model); err != nil {
		t.Fatalf("Unexpected error in Save: %s", err.Error())
	}
	if _, err := indexedPrimativesModels.Delete(model.Id); err != nil {
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
	registerIndexedPointersModel()

	// Create and save a new model with random pointer to primative fields
	model := createIndexedPointersModel()
	if err := indexedPointersModels.Save(model); err != nil {
		t.Fatalf("Unexpected error in Save: %s", err.Error())
	}
	if _, err := indexedPointersModels.Delete(model.Id); err != nil {
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
