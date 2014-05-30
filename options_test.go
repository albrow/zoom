// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File options_test.go tests the different options that may be provided
// in model type declarations using struct tags

package zoom

import (
	"github.com/garyburd/redigo/redis"
	"reflect"
	"testing"
)

// Test that the redis ignore struct tag causes a field to be ignored
func TestRedisIgnoreOption(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	type ignored struct {
		Attr string `redis:"-"`
		DefaultData
	}
	Register(&ignored{})
	defer Unregister(&ignored{})

	// check the spec
	spec, found := modelSpecs["ignored"]
	if !found {
		t.Error("Could not find spec for model of type ignored")
	}
	if len(spec.primatives) != 0 {
		t.Errorf("Expected no primatives in model spec. Got %d", len(spec.primatives))
	}

	// save a new model and check redis
	m := &ignored{
		Attr: "this should be ignored",
	}
	if err := Save(m); err != nil {
		t.Error(err)
	}

	conn := GetConn()
	defer conn.Close()
	key := "ignored:" + m.Id
	gotAttr, err := redis.String(conn.Do("HGET", key, "Attr"))
	if err != nil && err != redis.ErrNil {
		t.Error(err)
	}
	if gotAttr != "" {
		t.Errorf("Expected empty attr but got: %s", gotAttr)
	}
}

// Test that the redis name struct tag causes a field's name in redis to be changed
func TestRedisNameOption(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	type customized struct {
		Attr string `redis:"a"`
		DefaultData
	}
	Register(&customized{})
	defer Unregister(&customized{})

	// check the spec
	spec, found := modelSpecs["customized"]
	if !found {
		t.Error("Could not find spec for model of type customized")
	}
	if spec.primatives["Attr"].redisName != "a" {
		t.Errorf("Redis name was incorrect.\nExpected: 'a'\nGot: %s\n", spec.primatives["Attr"].redisName)
	}

	// save a new model and check redis
	m := &customized{
		Attr: "test",
	}
	if err := Save(m); err != nil {
		t.Error(err)
	}

	conn := GetConn()
	defer conn.Close()
	key := "customized:" + m.Id
	gotAttr, err := redis.String(conn.Do("HGET", key, "a"))
	if err != nil {
		t.Error(err)
	}
	if gotAttr != "test" {
		t.Errorf("Expected 'test' but got: %s", gotAttr)
	}
}

func TestInvalidOptionThrowsError(t *testing.T) {
	testingSetUp()
	testingTearDown()

	type invalid struct {
		Attr string `zoom:"index,poop"`
		DefaultData
	}
	if err := Register(&invalid{}); err == nil {
		t.Error("Expected error when registering struct with invalid tag")
	}
	Unregister(&invalid{})
}

// Test that a model spec for a model type with primative indexes is created properly
func TestIndexedPrimativesModelSpec(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms, found := modelSpecs["indexedPrimativesModel"]
	if !found {
		t.Error("Could not find modelSpec for indexedPrimativesModel")
	}
	numFields := ms.modelType.Elem().NumField()
	if len(ms.primativeIndexes) != numFields-1 {
		t.Errorf("Expected %d primative fields to be indexed but got %d", numFields-1, len(ms.primativeIndexes))
	} else {
		for i := 0; i < numFields; i++ {
			field := ms.modelType.Elem().Field(i)
			index := ms.primativeIndexes[field.Name]
			switch {
			case typeIsNumeric(field.Type):
				if index.indexType != indexNumeric {
					t.Errorf("Expected indexType to be numeric (%d) but got: %d", indexNumeric, index.indexType)
				}
			case typeIsString(field.Type):
				if index.indexType != indexAlpha {
					t.Errorf("Expected indexType to be boolean (%d) but got: %d", indexAlpha, index.indexType)
				}
			case typeIsBool(field.Type):
				if index.indexType != indexBoolean {
					t.Errorf("Expected indexType to be bool (%d) but got: %d", indexBoolean, index.indexType)
				}
			}
		}
	}
}

// Test that a model spec for a model type with pointer indexes is created properly
func TestIndexedPointersModelSpec(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms, found := modelSpecs["indexedPointersModel"]
	if !found {
		t.Error("Could not find modelSpec for indexedPointersModel")
	}

	numFields := ms.modelType.Elem().NumField()
	if len(ms.pointerIndexes) != numFields-1 {
		t.Errorf("Expected %d primative fields to be indexed but got %d", numFields-1, len(ms.pointerIndexes))
	} else {
		// iterate throough each field and validate the index struct inside the modelSpec
		for i := 0; i < numFields; i++ {
			field := ms.modelType.Elem().Field(i)
			index := ms.pointerIndexes[field.Name]
			switch {
			case typeIsNumeric(field.Type):
				if index.indexType != indexNumeric {
					t.Errorf("Expected indexType to be numeric (%d) but got: %d", indexNumeric, index.indexType)
				}
			case typeIsString(field.Type):
				if index.indexType != indexAlpha {
					t.Errorf("Expected indexType to be alpha (%d) but got: %d", indexAlpha, index.indexType)
				}
			case typeIsBool(field.Type):
				if index.indexType != indexBoolean {
					t.Errorf("Expected indexType to be bool (%d) but got: %d", indexBoolean, index.indexType)
				}
			}
		}
	}
}

// Test that the indexes are actually created in redis for a model with primative indexes
func TestSaveIndexedPrimativesModel(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms, err := newIndexedPrimativesModels(1)
	if err != nil {
		t.Error(err)
	}
	m := ms[0]

	if err := Save(m); err != nil {
		t.Error(err)
	}

	conn := GetConn()
	defer conn.Close()

	spec, found := modelSpecs["indexedPrimativesModel"]
	if !found {
		t.Error("Could not find modelSpec for indexedPrimativesModel")
	}
	numFields := spec.modelType.Elem().NumField()

	// iterate through each field using reflection and validate that the index was set properly
	for i := 0; i < numFields; i++ {
		field := spec.modelType.Elem().Field(i)
		if field.Anonymous {
			continue // skip embedded structs
		}
		val := reflect.ValueOf(m).Elem().FieldByName(field.Name)
		switch {
		case typeIsNumeric(field.Type):
			validateNumericIndexExists(t, "indexedPrimativesModel", m.Id, field.Name, val, conn)
		case typeIsString(field.Type):
			validateAlphaIndexExists(t, "indexedPrimativesModel", m.Id, field.Name, val.String(), conn)
		case typeIsBool(field.Type):
			validateBooleanIndexExists(t, "indexedPrimativesModel", m.Id, field.Name, val.Bool(), conn)
		default:
			t.Errorf("Unexpected type %s in struct for %s", field.Type.String(), "indexedPrimativesModel")
		}
	}
}

// Test that the indexes are actually created in redis for a model with pointer indexes
func TestSaveIndexedPointersModel(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms, err := newIndexedPointersModels(1)
	if err != nil {
		t.Error(err)
	}
	m := ms[0]

	if err := Save(m); err != nil {
		t.Error(err)
	}

	conn := GetConn()
	defer conn.Close()

	spec, found := modelSpecs["indexedPointersModel"]
	if !found {
		t.Error("Could not find modelSpec for indexedPointersModel")
	}
	numFields := spec.modelType.Elem().NumField()

	// iterate through each field using reflection and validate that the index was set properly
	for i := 0; i < numFields; i++ {
		field := spec.modelType.Elem().Field(i)
		if field.Anonymous {
			continue // skip embedded structs
		}
		ptr := reflect.ValueOf(m).Elem().FieldByName(field.Name)
		val := ptr.Elem()
		switch {
		case typeIsNumeric(field.Type.Elem()):
			validateNumericIndexExists(t, "indexedPointersModel", m.Id, field.Name, val, conn)
		case typeIsString(field.Type.Elem()):
			validateAlphaIndexExists(t, "indexedPointersModel", m.Id, field.Name, val.String(), conn)
		case typeIsBool(field.Type.Elem()):
			validateBooleanIndexExists(t, "indexedPointersModel", m.Id, field.Name, val.Bool(), conn)
		default:
			t.Errorf("Unexpected type %s in struct for %s", field.Type.String(), "indexedPrimativesModel")
		}
	}
}

// Test that the indexes are removed from redis after a model with primative indexes is deleted
func TestDeleteIndexedPrimativesModel(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms, err := newIndexedPrimativesModels(1)
	if err != nil {
		t.Error(err)
	}
	m := ms[0]

	if err := Save(m); err != nil {
		t.Error(err)
	}
	if err := Delete(m); err != nil {
		t.Error(err)
	}

	conn := GetConn()
	defer conn.Close()

	spec, found := modelSpecs["indexedPrimativesModel"]
	if !found {
		t.Error("Could not find modelSpec for indexedPrimativesModel")
	}
	numFields := spec.modelType.Elem().NumField()

	// iterate through each field using reflection and validate that the index was set properly
	for i := 0; i < numFields; i++ {
		field := spec.modelType.Elem().Field(i)
		if field.Anonymous {
			continue // skip embedded structs
		}
		val := reflect.ValueOf(m).Elem().FieldByName(field.Name)
		switch {
		case typeIsNumeric(field.Type):
			validateNumericIndexNotExists(t, "indexedPrimativesModel", m.Id, field.Name, val, conn)
		case typeIsString(field.Type):
			validateAlphaIndexNotExists(t, "indexedPrimativesModel", m.Id, field.Name, val.String(), conn)
		case typeIsBool(field.Type):
			validateBooleanIndexNotExists(t, "indexedPrimativesModel", m.Id, field.Name, val.Bool(), conn)
		default:
			t.Errorf("Unexpected type %s in struct for %s", field.Type.String(), "indexedPrimativesModel")
		}
	}
}

// Test that the indexes are removed from redis after a model with pointer indexes is deleted
func TestDeleteIndexedPointersModel(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms, err := newIndexedPointersModels(1)
	if err != nil {
		t.Error(err)
	}
	m := ms[0]

	if err := Save(m); err != nil {
		t.Error(err)
	}
	if err := Delete(m); err != nil {
		t.Error(err)
	}

	conn := GetConn()
	defer conn.Close()

	spec, found := modelSpecs["indexedPointersModel"]
	if !found {
		t.Error("Could not find modelSpec for indexedPointersModel")
	}
	numFields := spec.modelType.Elem().NumField()

	// iterate through each field using reflection and validate that the index was set properly
	for i := 0; i < numFields; i++ {
		field := spec.modelType.Elem().Field(i)
		if field.Anonymous {
			continue // skip embedded structs
		}
		ptr := reflect.ValueOf(m).Elem().FieldByName(field.Name)
		val := ptr.Elem()
		switch {
		case typeIsNumeric(field.Type.Elem()):
			validateNumericIndexNotExists(t, "indexedPointersModel", m.Id, field.Name, val, conn)
		case typeIsString(field.Type.Elem()):
			validateAlphaIndexNotExists(t, "indexedPointersModel", m.Id, field.Name, val.String(), conn)
		case typeIsBool(field.Type.Elem()):
			validateBooleanIndexNotExists(t, "indexedPointersModel", m.Id, field.Name, val.Bool(), conn)
		default:
			t.Errorf("Unexpected type %s in struct for %s", field.Type.String(), "indexedPrimativesModel")
		}
	}
}

func TestUpdateIndexedNumericModel(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	conn := GetConn()
	defer conn.Close()

	m := new(indexedPrimativesModel)
	m.Int = 123
	if err := Save(m); err != nil {
		t.Error(err)
	}
	validateNumericIndexExists(t, "indexedPrimativesModel", m.Id, "Int", reflect.ValueOf(123), conn)

	// now change the Int field and make sure the index was updated
	m.Int = 456
	if err := Save(m); err != nil {
		t.Error(err)
	}
	// index should exist on field value 456 (the new value)
	validateNumericIndexExists(t, "indexedPrimativesModel", m.Id, "Int", reflect.ValueOf(456), conn)
	// index should not exist on field value 123 (the old value)
	validateNumericIndexNotExists(t, "indexedPrimativesModel", m.Id, "Int", reflect.ValueOf(123), conn)
}

func TestUpdateIndexedAlphaModel(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	conn := GetConn()
	defer conn.Close()

	m := new(indexedPrimativesModel)
	m.String = "aaa"
	if err := Save(m); err != nil {
		t.Error(err)
	}
	validateAlphaIndexExists(t, "indexedPrimativesModel", m.Id, "String", "aaa", conn)

	// now change the String field and make sure the index was updated
	m.String = "bbb"
	if err := Save(m); err != nil {
		t.Error(err)
	}
	// index should exist on field value "bbb" (the new value)
	validateAlphaIndexExists(t, "indexedPrimativesModel", m.Id, "String", "bbb", conn)
	// index should not exist on field value "aaa" (the old value)
	validateAlphaIndexNotExists(t, "indexedPrimativesModel", m.Id, "String", "aaa", conn)
}

func TestUpdateIndexedBooleanModel(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	conn := GetConn()
	defer conn.Close()

	m := new(indexedPrimativesModel)
	m.Bool = false
	if err := Save(m); err != nil {
		t.Error(err)
	}
	validateBooleanIndexExists(t, "indexedPrimativesModel", m.Id, "Bool", false, conn)

	// now change the Bool field and make sure the index was updated
	m.Bool = true
	if err := Save(m); err != nil {
		t.Error(err)
	}
	// index should exist on field value true (the new value)
	validateBooleanIndexExists(t, "indexedPrimativesModel", m.Id, "Bool", true, conn)
	// index should not exist on field value false (the old value)
	validateBooleanIndexNotExists(t, "indexedPrimativesModel", m.Id, "Bool", false, conn)
}

// returns true if the numeric index exists
// if err is not nil there was an unexpected error
func numericIndexExists(modelName string, modelId string, fieldName string, fieldValue reflect.Value, conn redis.Conn) (bool, error) {
	indexKey := modelName + ":" + fieldName
	score, err := convertNumericToFloat64(fieldValue)
	if err != nil {
		return false, err
	}
	results, err := redis.Strings(conn.Do("ZRANGEBYSCORE", indexKey, score, score))
	if err != nil {
		return false, err
	}
	return len(results) != 0, nil
}

// make sure an alpha index exists
// uses t.Error or t.Errorf to report an error if the index does not exist
func validateNumericIndexExists(t *testing.T, modelName string, modelId string, fieldName string, fieldValue reflect.Value, conn redis.Conn) {
	if found, err := numericIndexExists(modelName, modelId, fieldName, fieldValue, conn); err != nil {
		t.Errorf("unexpected error:\n%s", err)
	} else if !found {
		t.Errorf("numeric index was not set.\nExpected %s:%s to have valid score\n", modelName, modelId)
	}
}

// make sure an numeric index DOES NOT exist
// uses t.Error or t.Errorf to report an error if the index DOES exist
func validateNumericIndexNotExists(t *testing.T, modelName string, modelId string, fieldName string, fieldValue reflect.Value, conn redis.Conn) {
	if found, err := numericIndexExists(modelName, modelId, fieldName, fieldValue, conn); err != nil {
		t.Errorf("unexpected error:\n%s", err)
	} else if found {
		t.Errorf("numeric index was set.\nExpected %s:%s to be gone, but it was there\n", modelName, modelId)
	}
}

// returns true if the alpha index exists
// if err is not nil there was an unexpected error
func alphaIndexExists(modelName string, modelId string, fieldName string, fieldValue string, conn redis.Conn) (bool, error) {
	indexKey := modelName + ":" + fieldName
	memberKey := fieldValue + " " + modelId
	_, err := redis.Int(conn.Do("ZRANK", indexKey, memberKey))
	if err != nil {
		if err == redis.ErrNil {
			return false, nil
		} else {
			return false, err
		}
	} else {
		return true, nil
	}
}

// make sure an alpha index exists
// uses t.Error or t.Errorf to report an error if the index does not exist
func validateAlphaIndexExists(t *testing.T, modelName string, modelId string, fieldName string, fieldValue string, conn redis.Conn) {
	if found, err := alphaIndexExists(modelName, modelId, fieldName, fieldValue, conn); err != nil {
		t.Errorf("unexpected error:\n%s", err)
	} else if !found {
		t.Errorf("alpha index was not set\nExpected to find member %s %s\n%s", fieldValue, modelId, err)
	}
}

// make sure an alpha index DOES NOT exist
// uses t.Error or t.Errorf to report an error if the index DOES exist
func validateAlphaIndexNotExists(t *testing.T, modelName string, modelId string, fieldName string, fieldValue string, conn redis.Conn) {
	if found, err := alphaIndexExists(modelName, modelId, fieldName, fieldValue, conn); err != nil {
		t.Errorf("unexpected error:\n%s", err)
	} else if found {
		t.Errorf("alpha index was set.\nExpected member %s %s to be gone.\n", fieldValue, modelId)
	}
}

// returns true if the boolean index exists
// if err is not nil there was an unexpected error
func booleanIndexExists(modelName string, modelId string, fieldName string, fieldValue bool, conn redis.Conn) (bool, error) {
	indexKey := modelName + ":" + fieldName
	var score float64
	if fieldValue == true {
		score = 1.0
	} else {
		score = 0.0
	}
	results, err := redis.Strings(conn.Do("ZRANGEBYSCORE", indexKey, score, score))
	if err != nil {
		return false, err
	}
	return len(results) != 0, nil
}

// make sure an boolean index exists
// uses t.Error or t.Errorf to report an error if the index does not exist
func validateBooleanIndexExists(t *testing.T, modelName string, modelId string, fieldName string, fieldValue bool, conn redis.Conn) {
	if found, err := booleanIndexExists(modelName, modelId, fieldName, fieldValue, conn); err != nil {
		t.Errorf("unexpected error:\n%s", err)
	} else if !found {
		t.Errorf("bool index was not set\nExpected to find member %s\n", modelId)
	}
}

// make sure an boolean index DOES NOT exist
// uses t.Error or t.Errorf to report an error if the index DOES exist
func validateBooleanIndexNotExists(t *testing.T, modelName string, modelId string, fieldName string, fieldValue bool, conn redis.Conn) {
	if found, err := booleanIndexExists(modelName, modelId, fieldName, fieldValue, conn); err != nil {
		t.Errorf("unexpected error:\n%s", err)
	} else if found {
		t.Errorf("bool index was set\nExpected member %s to be gone.\n", modelId)
	}
}
