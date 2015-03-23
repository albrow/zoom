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

// setUpOnce is used to enforce that the setup process happens exactly once,
// no matter how many times testingSetUp is called
var setUpOnce = sync.Once{}

// testingSetUp prepares the database for testing and registers the testing types.
// The setup-related code only runs once, no matter how many times you call
// testingSetUp
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

// testModel is a model type that is used for testing
type testModel struct {
	Int    int
	String string
	Bool   bool
	DefaultData
}

// createTestModels creates and returns n testModels with
// random field values, but does not save them to the database.
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

// createAndSaveTestModels creates n testModels with random field
// values, saves them, and returns them.
func createAndSaveTestModels(n int) ([]*testModel, error) {
	models := createTestModels(n)
	t := NewTransaction()
	for _, model := range models {
		t.Save(testModels, model)
	}
	if err := t.Exec(); err != nil {
		return nil, err
	}
	return models, nil
}

// indexedTestModel is a model type used for testing indexes
// and queries.
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

// registerTestingTypes registers the common types used for testing
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

// checkDatabaseEmpty panics if the database to be used for testing
// is not empty.
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

// testingTearDown flushes the database. It should be run at the end
// of each test that toches the database, typically by using defer.
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

// expectModelExists sets an error via t.Errorf if model does not exist in
// the database. It checks for the main hash as well as the id in the index of all
// ids for a given type.
func expectModelExists(t *testing.T, mt *ModelType, model Model) {
	modelKey, err := mt.KeyForModel(model)
	if err != nil {
		t.Fatalf("Unexpected error in KeyForModel: %s", err.Error())
	}
	expectKeyExists(t, modelKey)
	expectSetContains(t, mt.AllIndexKey(), model.GetId())
}

// expectModelDoesNotExist sets an error via t.Errorf if model exists in the database.
// It checks for the main hash as well as the id in the index of all ids for a
// given type.
func expectModelDoesNotExist(t *testing.T, mt *ModelType, model Model) {
	modelKey, err := mt.KeyForModel(model)
	if err != nil {
		t.Fatalf("Unexpected error in KeyForModel: %s", err.Error())
	}
	expectKeyDoesNotExist(t, modelKey)
	expectSetDoesNotContain(t, mt.AllIndexKey(), model.GetId())
}

// expectModelsExist sets an error via t.Errorf for each model in models that
// does not exist in the database. It checks for the main hash as well as the id in
// the index of all ids for a given type.
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

// expectModelsDoNotExist sets an error via t.Errorf for each model in models that
// exists in the database. It checks for the main hash as well as the id in the index
// of all ids for a given type.
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
