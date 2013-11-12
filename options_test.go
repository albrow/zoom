// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File options_test.go tests the different options that may be provided
// in model type declarations using struct tags

package zoom

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"reflect"
	"testing"
)

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
				if index.iType != indexNumeric {
					t.Errorf("Expected iType to be numeric (%d) but got: %d", indexNumeric, index.iType)
				}
			case typeIsString(field.Type):
				if index.iType != indexAlpha {
					t.Errorf("Expected iType to be alpha (%d) but got: %d", indexAlpha, index.iType)
				}
			case typeIsBool(field.Type):
				if index.iType != indexBoolean {
					t.Errorf("Expected iType to be bool (%d) but got: %d", indexBoolean, index.iType)
				}
			}
		}
	}
}

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
				if index.iType != indexNumeric {
					t.Errorf("Expected iType to be numeric (%d) but got: %d", indexNumeric, index.iType)
				}
			case typeIsString(field.Type):
				if index.iType != indexAlpha {
					t.Errorf("Expected iType to be alpha (%d) but got: %d", indexAlpha, index.iType)
				}
			case typeIsBool(field.Type):
				fmt.Println("checking that boolean shit")
				if index.iType != indexBoolean {
					t.Errorf("Expected iType to be bool (%d) but got: %d", indexBoolean, index.iType)
				}
			}
		}
	}
}

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
			validateNumericIndex(t, "indexedPrimativesModel", m.Id, field.Name, val, conn)
		case typeIsString(field.Type):
			validateAlphaIndex(t, "indexedPrimativesModel", m.Id, field.Name, val.String(), conn)
		case typeIsBool(field.Type):
			validateBooleanIndex(t, "indexedPrimativesModel", m.Id, field.Name, val.Bool(), conn)
		default:
			t.Errorf("Unexpected type %s in struct for %s", field.Type.String(), "indexedPrimativesModel")
		}
	}
}

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
			validateNumericIndex(t, "indexedPointersModel", m.Id, field.Name, val, conn)
		case typeIsString(field.Type.Elem()):
			validateAlphaIndex(t, "indexedPointersModel", m.Id, field.Name, val.String(), conn)
		case typeIsBool(field.Type.Elem()):
			validateBooleanIndex(t, "indexedPointersModel", m.Id, field.Name, val.Bool(), conn)
		default:
			t.Errorf("Unexpected type %s in struct for %s", field.Type.String(), "indexedPrimativesModel")
		}
	}
}

// validate index on a numeric type
func validateNumericIndex(t *testing.T, modelName string, modelId string, fieldName string, fieldValue reflect.Value, conn redis.Conn) {
	indexKey := modelName + ":" + fieldName
	score, err := convertNumericToFloat64(fieldValue)
	if err != nil {
		t.Error(err)
	}
	results, err := redis.Strings(conn.Do("ZRANGEBYSCORE", indexKey, score, score))
	if err != nil {
		t.Error(err)
	}
	if len(results) == 0 {
		t.Errorf("numeric index was not set.\nExpected %s:%s to have score %f\n", modelName, modelId, score)
	}
}

// validate index on a string type
func validateAlphaIndex(t *testing.T, modelName string, modelId string, fieldName string, fieldValue string, conn redis.Conn) {
	indexKey := modelName + ":" + fieldName
	memberKey := fieldValue + " " + modelId
	_, err := redis.Int(conn.Do("ZRANK", indexKey, memberKey))
	if err != nil {
		t.Errorf("alpha index was not set\nExpected to find member %s\n%s", memberKey, err)
	}
}

// validate index on a boolean type
func validateBooleanIndex(t *testing.T, modelName string, modelId string, fieldName string, fieldValue bool, conn redis.Conn) {
	var indexKey string
	if fieldValue == true {
		indexKey = modelName + ":" + fieldName + ":true"
	} else {
		indexKey = modelName + ":" + fieldName + ":false"
	}
	if found, err := redis.Bool(conn.Do("SISMEMBER", indexKey, modelId)); err != nil {
		t.Error(err)
	} else if !found {
		t.Errorf("bool index was not set\nExpected to find member %s in set %s\n", modelId, indexKey)
	}
}
