// Copyright 2014 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File struct_tags_test.go tests the different options
// that may be provided in model type declarations using
// struct tags.

package zoom

import (
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

// Test that the indexes are actually created in redis for a model with primative indexes
func TestSaveIndexedTestModel(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	models, err := createAndSaveIndexedTestModels(1)
	if err != nil {
		t.Error(err)
	}
	model := models[0]

	if err := indexedTestModels.Save(model); err != nil {
		t.Error("Unexpected error in Save: %s", err.Error())
	}

	// iterate through each field using reflection and validate that the index was set properly
	numFields := indexedTestModels.spec.typ.Elem().NumField()
	for i := 0; i < numFields; i++ {
		field := indexedTestModels.spec.typ.Elem().Field(i)
		if field.Anonymous {
			continue // skip embedded structs
		}
		expectIndexExists(t, indexedTestModels, model, field.Name)
	}
}

// // Test that the indexes are actually created in redis for a model with pointer indexes
// func TestSaveIndexedPointersModel(t *testing.T) {
// 	testingSetUp()
// 	defer testingTearDown()

// 	ms, err := newIndexedPointersModels(1)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	m := ms[0]

// 	if err := Save(m); err != nil {
// 		t.Error(err)
// 	}

// 	conn := GetConn()
// 	defer conn.Close()

// 	spec, found := modelSpecs["indexedPointersModel"]
// 	if !found {
// 		t.Error("Could not find modelSpec for indexedPointersModel")
// 	}
// 	numFields := spec.modelType.Elem().NumField()

// 	// iterate through each field using reflection and validate that the index was set properly
// 	for i := 0; i < numFields; i++ {
// 		field := spec.modelType.Elem().Field(i)
// 		if field.Anonymous {
// 			continue // skip embedded structs
// 		}
// 		ptr := reflect.ValueOf(m).Elem().FieldByName(field.Name)
// 		val := ptr.Elem()
// 		switch {
// 		case typeIsNumeric(field.Type.Elem()):
// 			validateNumericIndexExists(t, "indexedPointersModel", m.Id, field.Name, val, conn)
// 		case typeIsString(field.Type.Elem()):
// 			validateAlphaIndexExists(t, "indexedPointersModel", m.Id, field.Name, val.String(), conn)
// 		case typeIsBool(field.Type.Elem()):
// 			validateBooleanIndexExists(t, "indexedPointersModel", m.Id, field.Name, val.Bool(), conn)
// 		default:
// 			t.Errorf("Unexpected type %s in struct for %s", field.Type.String(), "indexedPrimativesModel")
// 		}
// 	}
// }

// // Test that the indexes are removed from redis after a model with primative indexes is deleted
// func TestDeleteIndexedPrimativesModel(t *testing.T) {
// 	testingSetUp()
// 	defer testingTearDown()

// 	ms, err := newIndexedPrimativesModels(1)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	m := ms[0]

// 	if err := Save(m); err != nil {
// 		t.Error(err)
// 	}
// 	if err := Delete(m); err != nil {
// 		t.Error(err)
// 	}

// 	conn := GetConn()
// 	defer conn.Close()

// 	spec, found := modelSpecs["indexedPrimativesModel"]
// 	if !found {
// 		t.Error("Could not find modelSpec for indexedPrimativesModel")
// 	}
// 	numFields := spec.modelType.Elem().NumField()

// 	// iterate through each field using reflection and validate that the index was set properly
// 	for i := 0; i < numFields; i++ {
// 		field := spec.modelType.Elem().Field(i)
// 		if field.Anonymous {
// 			continue // skip embedded structs
// 		}
// 		val := reflect.ValueOf(m).Elem().FieldByName(field.Name)
// 		switch {
// 		case typeIsNumeric(field.Type):
// 			validateNumericIndexNotExists(t, "indexedPrimativesModel", m.Id, field.Name, val, conn)
// 		case typeIsString(field.Type):
// 			validateAlphaIndexNotExists(t, "indexedPrimativesModel", m.Id, field.Name, val.String(), conn)
// 		case typeIsBool(field.Type):
// 			validateBooleanIndexNotExists(t, "indexedPrimativesModel", m.Id, field.Name, val.Bool(), conn)
// 		default:
// 			t.Errorf("Unexpected type %s in struct for %s", field.Type.String(), "indexedPrimativesModel")
// 		}
// 	}
// }

// // Test that the indexes are removed from redis after a model with pointer indexes is deleted
// func TestDeleteIndexedPointersModel(t *testing.T) {
// 	testingSetUp()
// 	defer testingTearDown()

// 	ms, err := newIndexedPointersModels(1)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	m := ms[0]

// 	if err := Save(m); err != nil {
// 		t.Error(err)
// 	}
// 	if err := Delete(m); err != nil {
// 		t.Error(err)
// 	}

// 	conn := GetConn()
// 	defer conn.Close()

// 	spec, found := modelSpecs["indexedPointersModel"]
// 	if !found {
// 		t.Error("Could not find modelSpec for indexedPointersModel")
// 	}
// 	numFields := spec.modelType.Elem().NumField()

// 	// iterate through each field using reflection and validate that the index was set properly
// 	for i := 0; i < numFields; i++ {
// 		field := spec.modelType.Elem().Field(i)
// 		if field.Anonymous {
// 			continue // skip embedded structs
// 		}
// 		ptr := reflect.ValueOf(m).Elem().FieldByName(field.Name)
// 		val := ptr.Elem()
// 		switch {
// 		case typeIsNumeric(field.Type.Elem()):
// 			validateNumericIndexNotExists(t, "indexedPointersModel", m.Id, field.Name, val, conn)
// 		case typeIsString(field.Type.Elem()):
// 			validateAlphaIndexNotExists(t, "indexedPointersModel", m.Id, field.Name, val.String(), conn)
// 		case typeIsBool(field.Type.Elem()):
// 			validateBooleanIndexNotExists(t, "indexedPointersModel", m.Id, field.Name, val.Bool(), conn)
// 		default:
// 			t.Errorf("Unexpected type %s in struct for %s", field.Type.String(), "indexedPrimativesModel")
// 		}
// 	}
// }

// func TestUpdateIndexedNumericModel(t *testing.T) {
// 	testingSetUp()
// 	defer testingTearDown()

// 	conn := GetConn()
// 	defer conn.Close()

// 	m := new(indexedPrimativesModel)
// 	m.Int = 123
// 	if err := Save(m); err != nil {
// 		t.Error(err)
// 	}
// 	validateNumericIndexExists(t, "indexedPrimativesModel", m.Id, "Int", reflect.ValueOf(123), conn)

// 	// now change the Int field and make sure the index was updated
// 	m.Int = 456
// 	if err := Save(m); err != nil {
// 		t.Error(err)
// 	}
// 	// index should exist on field value 456 (the new value)
// 	validateNumericIndexExists(t, "indexedPrimativesModel", m.Id, "Int", reflect.ValueOf(456), conn)
// 	// index should not exist on field value 123 (the old value)
// 	validateNumericIndexNotExists(t, "indexedPrimativesModel", m.Id, "Int", reflect.ValueOf(123), conn)
// }

// func TestUpdateIndexedAlphaModel(t *testing.T) {
// 	testingSetUp()
// 	defer testingTearDown()

// 	conn := GetConn()
// 	defer conn.Close()

// 	m := new(indexedPrimativesModel)
// 	m.String = "aaa"
// 	if err := Save(m); err != nil {
// 		t.Error(err)
// 	}
// 	validateAlphaIndexExists(t, "indexedPrimativesModel", m.Id, "String", "aaa", conn)

// 	// now change the String field and make sure the index was updated
// 	m.String = "bbb"
// 	if err := Save(m); err != nil {
// 		t.Error(err)
// 	}
// 	// index should exist on field value "bbb" (the new value)
// 	validateAlphaIndexExists(t, "indexedPrimativesModel", m.Id, "String", "bbb", conn)
// 	// index should not exist on field value "aaa" (the old value)
// 	validateAlphaIndexNotExists(t, "indexedPrimativesModel", m.Id, "String", "aaa", conn)
// }

// func TestUpdateIndexedBooleanModel(t *testing.T) {
// 	testingSetUp()
// 	defer testingTearDown()

// 	conn := GetConn()
// 	defer conn.Close()

// 	m := new(indexedPrimativesModel)
// 	m.Bool = false
// 	if err := Save(m); err != nil {
// 		t.Error(err)
// 	}
// 	validateBooleanIndexExists(t, "indexedPrimativesModel", m.Id, "Bool", false, conn)

// 	// now change the Bool field and make sure the index was updated
// 	m.Bool = true
// 	if err := Save(m); err != nil {
// 		t.Error(err)
// 	}
// 	// index should exist on field value true (the new value)
// 	validateBooleanIndexExists(t, "indexedPrimativesModel", m.Id, "Bool", true, conn)
// 	// index should not exist on field value false (the old value)
// 	validateBooleanIndexNotExists(t, "indexedPrimativesModel", m.Id, "Bool", false, conn)
// }
