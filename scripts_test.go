// Copyright 2015 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File scripts_test.go tests all the lua scripts.

package zoom

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/garyburd/redigo/redis"
)

func TestDeleteModelsBySetIDsScript(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// Create and save some test models
	models, err := createAndSaveTestModels(5)
	if err != nil {
		t.Fatalf("Unexpected error saving test models: %s", err.Error())
	}

	// The set of ids will contain three valid ids and two invalid ones
	ids := []string{}
	for _, model := range models[:3] {
		ids = append(ids, model.ModelID())
	}
	ids = append(ids, "foo", "bar")
	tempSetKey := "testModelIDs"
	conn := testPool.NewConn()
	defer func() {
		_ = conn.Close()
	}()
	saddArgs := redis.Args{tempSetKey}
	saddArgs = saddArgs.Add(Interfaces(ids)...)
	if _, err = conn.Do("SADD", saddArgs...); err != nil {
		t.Errorf("Unexpected error in SADD: %s", err.Error())
	}

	// Run the script
	tx := testPool.NewTransaction()
	count := 0
	tx.DeleteModelsBySetIDs(tempSetKey, testModels.Name(), NewScanIntHandler(&count))
	if err := tx.Exec(); err != nil {
		t.Fatalf("Unexected error in tx.Exec: %s", err.Error())
	}

	// Check that the return value is correct
	if count != 3 {
		t.Errorf("Expected count to be 3 but got %d", count)
	}

	// Make sure the first three models were deleted
	for _, model := range models[:3] {
		modelKey := testModels.ModelKey(model.ModelID())
		expectKeyDoesNotExist(t, modelKey)
		expectSetDoesNotContain(t, testModels.IndexKey(), model.ModelID())
	}
	// Make sure the last two models were not deleted
	for _, model := range models[3:] {
		modelKey := testModels.ModelKey(model.ModelID())
		expectKeyExists(t, modelKey)
		expectSetContains(t, testModels.IndexKey(), model.ModelID())
	}
}

func TestDeleteStringIndexScript(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// Create a new collection with an indexed string field
	type stringIndexModel struct {
		String string `zoom:"index"`
		RandomID
	}
	options := DefaultCollectionOptions.WithIndex(true)
	stringIndexModels, err := testPool.NewCollectionWithOptions(&stringIndexModel{}, options)
	if err != nil {
		t.Errorf("Unexpected error registering stringIndexModel: %s", err.Error())
	}

	// Create a new model (but don't save it yet)
	model := &stringIndexModel{
		String: "foo",
	}
	model.SetModelID("testID")

	// Run the script before saving the hash, to make sure it does not cause an error
	tx := testPool.NewTransaction()
	tx.deleteStringIndex(stringIndexModels.Name(), model.ModelID(), "String")
	if err := tx.Exec(); err != nil {
		t.Fatalf("Unexected error in tx.Exec: %s", err.Error())
	}

	// Set the field value in the main hash
	conn := testPool.NewConn()
	defer func() {
		_ = conn.Close()
	}()
	modelKey := stringIndexModels.ModelKey(model.ModelID())
	if _, err := conn.Do("HSET", modelKey, "String", model.String); err != nil {
		t.Errorf("Unexpected error in HSET")
	}

	// Add the model to the index for the string field
	fieldIndexKey, err := stringIndexModels.FieldIndexKey("String")
	if err != nil {
		t.Fatalf("Unexpected error in FieldIndexKey: %s", err.Error())
	}
	member := model.String + " " + model.ModelID()
	if _, err := conn.Do("ZADD", fieldIndexKey, 0, member); err != nil {
		t.Fatalf("Unexpected error in ZADD: %s", err.Error())
	}

	// Run the script again. This time we expect the index to be removed
	tx = testPool.NewTransaction()
	tx.deleteStringIndex(stringIndexModels.Name(), model.ModelID(), "String")
	if err := tx.Exec(); err != nil {
		t.Fatalf("Unexected error in tx.Exec: %s", err.Error())
	}

	// Check that the index was removed
	expectIndexDoesNotExist(t, stringIndexModels, model, "String")
}

func TestExtractIDsFromFieldIndexScript(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// Create and save some test models with increasing Int values
	models := createIndexedTestModels(5)
	tx := testPool.NewTransaction()
	for i, model := range models {
		model.Int = i
		tx.Save(indexedTestModels, model)
	}
	if err := tx.Exec(); err != nil {
		t.Errorf("Unexpected error saving models in tx.Exec: %s", err.Error())
	}

	// Create a few test cases
	testCases := []struct {
		min         interface{}
		max         interface{}
		expectedIDs []string
	}{
		{
			min:         "-inf",
			max:         "+inf",
			expectedIDs: modelIDs(Models(models)),
		},
		{
			min:         "2",
			max:         "+inf",
			expectedIDs: modelIDs(Models(models[2:])),
		},
		{
			min:         "(2",
			max:         "+inf",
			expectedIDs: modelIDs(Models(models[3:])),
		},
	}

	// Run the script for each test case and check the result
	fieldIndexKey, _ := indexedTestModels.FieldIndexKey("Int")
	for i, tc := range testCases {
		gotIDs := []string{}
		destKey := "TestExtractIDsFromFieldIndexScript:" + strconv.Itoa(i)
		tx = testPool.NewTransaction()
		tx.ExtractIDsFromFieldIndex(fieldIndexKey, destKey, tc.min, tc.max)
		tx.Command("ZRANGE", redis.Args{destKey, 0, -1}, NewScanStringsHandler(&gotIDs))
		if err := tx.Exec(); err != nil {
			t.Errorf("Unexpected error in tx.Exec: %s", err.Error())
		}
		if !reflect.DeepEqual(gotIDs, tc.expectedIDs) {
			t.Errorf("Script results were incorrect.\nExpected: %v\nGot:      %v", tc.expectedIDs, gotIDs)
		}
	}
}

func TestExtractIDsFromStringIndexScript(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// Create and save some test models with increasing String values
	models := createIndexedTestModels(5)
	tx := testPool.NewTransaction()
	for i, model := range models {
		model.String = strconv.Itoa(i)
		tx.Save(indexedTestModels, model)
	}
	if err := tx.Exec(); err != nil {
		t.Errorf("Unexpected error saving models in tx.Exec: %s", err.Error())
	}

	// Create a few test cases
	testCases := []struct {
		min         string
		max         string
		expectedIDs []string
	}{
		{
			min:         "-",
			max:         "+",
			expectedIDs: modelIDs(Models(models)),
		},
		{
			min:         "[2",
			max:         "+",
			expectedIDs: modelIDs(Models(models[2:])),
		},
		{
			min:         "(2" + delString,
			max:         "+",
			expectedIDs: modelIDs(Models(models[3:])),
		},
	}

	// Run the script for each test case and check the result
	fieldIndexKey, _ := indexedTestModels.FieldIndexKey("String")
	for i, tc := range testCases {
		gotIDs := []string{}
		destKey := "ExtractIDsFromStringIndexScript:" + strconv.Itoa(i)
		tx = testPool.NewTransaction()
		tx.ExtractIDsFromStringIndex(fieldIndexKey, destKey, tc.min, tc.max)
		tx.Command("ZRANGE", redis.Args{destKey, 0, -1}, NewScanStringsHandler(&gotIDs))
		if err := tx.Exec(); err != nil {
			t.Errorf("Unexpected error in tx.Exec: %s", err.Error())
		}
		if !reflect.DeepEqual(gotIDs, tc.expectedIDs) {
			t.Errorf("Script results for test case %d were incorrect.\nExpected: %v\nGot:      %v", i, tc.expectedIDs, gotIDs)
		}
	}
}
