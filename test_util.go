// Copyright 2014 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

package zoom

import (
	"flag"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"reflect"
	"sync"
	"testing"
)

var (
	address  *string = flag.String("address", "localhost:6379", "the address of a redis server to connect to")
	network  *string = flag.String("network", "tcp", "the network to use for the database connection (e.g. 'tcp' or 'unix')")
	database *int    = flag.Int("database", 9, "the redis database number to use for testing")
)

var setUpOnce = sync.Once{}

func testingSetUp() {
	setUpOnce.Do(func() {
		Init(&Configuration{
			Address:  *address,
			Network:  *network,
			Database: *database,
		})
		checkDatabaseEmpty()
		registerTestingTypes()
	})
}

type testModel struct {
	Int    int
	String string
	Bool   bool
	DefaultData
}

func createTestModels(n int) []*testModel {
	models := make([]*testModel, n)
	for i := 0; i < n; i++ {
		models[i] = &testModel{
			Int:    randomInt(),
			String: randomString(),
			Bool:   randomBool(),
		}
	}
	return models
}

func createAndSaveTestModels(n int) ([]*testModel, error) {
	models := createTestModels(n)
	t := newTransaction()
	for _, model := range models {
		t.save(testModels, model)
	}
	if err := t.exec(); err != nil {
		return nil, err
	}
	return models, nil
}

type indexedTestModel struct {
	Int    int    `zoom:"index"`
	String string `zoom:"index"`
	Bool   bool   `zoom:"index"`
	DefaultData
}

var (
	testModels        *ModelType
	indexedTestModels *ModelType
)

func registerTestingTypes() {
	testModelTypes := []struct {
		modelType **ModelType
		model     Model
	}{
		{
			modelType: &testModels,
			model:     &testModel{},
		},
		{
			modelType: &indexedTestModels,
			model:     &indexedTestModel{},
		},
	}
	for _, m := range testModelTypes {
		modelType, err := Register(m.model)
		if err != nil {
			panic(err)
		}
		*m.modelType = modelType
	}
}

func checkDatabaseEmpty() {
	conn := GetConn()
	defer conn.Close()
	n, err := redis.Int(conn.Do("DBSIZE"))
	if err != nil {
		panic(err.Error())
	}
	if n != 0 {
		err := fmt.Errorf("Database #%d is not empty! Testing can not continue.", *database)
		panic(err)
	}
}

func testingTearDown() {
	// flush and close the database
	conn := GetConn()
	_, err := conn.Do("flushdb")
	if err != nil {
		panic(err)
	}
	conn.Close()
}

// expectSetContains sets an error via t.Errorf if member is not in the set
func expectSetContains(t *testing.T, setName string, member interface{}) {
	conn := GetConn()
	defer conn.Close()
	contains, err := redis.Bool(conn.Do("SISMEMBER", setName, member))
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	if !contains {
		t.Errorf("Expected set %s to contain %s but it did not.", setName, member)
	}
}

// expectSetDoesNotContain sets an error via t.Errorf if member is in the set
func expectSetDoesNotContain(t *testing.T, setName string, member interface{}) {
	conn := GetConn()
	defer conn.Close()
	contains, err := redis.Bool(conn.Do("SISMEMBER", setName, member))
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	if contains {
		t.Errorf("Expected set %s to not contain %s but it did.", setName, member)
	}
}

// expectFieldEquals sets an error via t.Errorf if the the field identified by fieldName does
// not equal expected according to the database.
func expectFieldEquals(t *testing.T, key string, fieldName string, expected interface{}) {
	conn := GetConn()
	defer conn.Close()
	reply, err := conn.Do("HGET", key, fieldName)
	if err != nil {
		t.Errorf("Unexpected error in HGET: %s", err.Error())
	}
	srcBytes, ok := reply.([]byte)
	if !ok {
		t.Fatalf("Unexpected error: could not convert %v of type %T to []byte.\n", reply, reply)
	}
	typ := reflect.TypeOf(expected)
	dest := reflect.New(typ).Elem()
	switch {
	case typeIsPrimative(typ):
		err = scanPrimativeVal(srcBytes, dest)
	case typ.Kind() == reflect.Ptr:
		err = scanPointerVal(srcBytes, dest)
	default:
		err = scanInconvertibleVal(srcBytes, dest)
	}
	if err != nil {
		t.Errorf("Unexpected error scanning value for field %s: %s", fieldName, err)
	}
	got := dest.Interface()
	if !reflect.DeepEqual(expected, got) {
		t.Errorf("Field %s for %s was not saved correctly.\n\tExpected: %v\n\tBut got:  %v", fieldName, key, expected, got)
	}
}

// expectKeyExists sets an error via t.Errorf if key does not exist in the database.
func expectKeyExists(t *testing.T, key string) {
	conn := GetConn()
	defer conn.Close()
	if exists, err := redis.Bool(conn.Do("EXISTS", key)); err != nil {
		t.Errorf("Unexpected error in EXISTS: %s", err.Error())
	} else if !exists {
		t.Errorf("Expected key %s to exist, but it did not.", key)
	}
}

// expectKeyDoesNotExist sets an error via t.Errorf if key does exist in the database.
func expectKeyDoesNotExist(t *testing.T, key string) {
	conn := GetConn()
	defer conn.Close()
	if exists, err := redis.Bool(conn.Do("EXISTS", key)); err != nil {
		t.Errorf("Unexpected error in EXISTS: %s", err.Error())
	} else if exists {
		t.Errorf("Expected key %s to not exist, but it did exist.", key)
	}
}

func expectModelExists(t *testing.T, mt *ModelType, model Model) {
	modelKey, err := mt.KeyForModel(model)
	if err != nil {
		t.Fatalf("Unexpected error in KeyForModel: %s", err.Error())
	}
	expectKeyExists(t, modelKey)
	expectSetContains(t, mt.AllIndexKey(), model.GetId())
}

func expectModelDoesNotExist(t *testing.T, mt *ModelType, model Model) {
	modelKey, err := mt.KeyForModel(model)
	if err != nil {
		t.Fatalf("Unexpected error in KeyForModel: %s", err.Error())
	}
	expectKeyDoesNotExist(t, modelKey)
	expectSetDoesNotContain(t, mt.AllIndexKey(), model.GetId())
}

func expectModelsExist(t *testing.T, mt *ModelType, models []Model) {
	for _, model := range models {
		modelKey, err := mt.KeyForModel(model)
		if err != nil {
			t.Fatalf("Unexpected error in KeyForModel: %s", err.Error())
		}
		expectKeyExists(t, modelKey)
		expectSetContains(t, mt.AllIndexKey(), model.GetId())
	}
}

func expectModelsDoNotExist(t *testing.T, mt *ModelType, models []Model) {
	for _, model := range models {
		modelKey, err := mt.KeyForModel(model)
		if err != nil {
			t.Fatalf("Unexpected error in KeyForModel: %s", err.Error())
		}
		expectKeyDoesNotExist(t, modelKey)
		expectSetDoesNotContain(t, mt.AllIndexKey(), model.GetId())
	}
}
