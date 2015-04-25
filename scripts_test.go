package zoom

import (
	"github.com/garyburd/redigo/redis"
	"reflect"
	"strconv"
	"testing"
)

func TestDeleteModelsBySetIdsScript(t *testing.T) {
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
		ids = append(ids, model.Id())
	}
	ids = append(ids, "foo", "bar")
	tempSetKey := "testModelIds"
	conn := Conn()
	defer conn.Close()
	saddArgs := redis.Args{tempSetKey}
	saddArgs = saddArgs.Add(Interfaces(ids)...)
	if _, err = conn.Do("SADD", saddArgs...); err != nil {
		t.Errorf("Unexpected error in SADD: %s", err.Error())
	}

	// Run the script
	tx := NewTransaction()
	count := 0
	tx.deleteModelsBySetIds(tempSetKey, testModels.Name(), newScanIntHandler(&count))
	if err := tx.Exec(); err != nil {
		t.Fatalf("Unexected error in tx.Exec: %s", err.Error())
	}

	// Check that the return value is correct
	if count != 3 {
		t.Errorf("Expected count to be 3 but got %d", count)
	}

	// Make sure the first three models were deleted
	for _, model := range models[:3] {
		modelKey, err := testModels.ModelKey(model)
		if err != nil {
			t.Errorf("Unexpected error in ModelKey: %s", err.Error())
		}
		expectKeyDoesNotExist(t, modelKey)
		expectSetDoesNotContain(t, testModels.AllIndexKey(), model.Id())
	}
	// Make sure the last two models were not deleted
	for _, model := range models[3:] {
		modelKey, err := testModels.ModelKey(model)
		if err != nil {
			t.Errorf("Unexpected error in ModelKey: %s", err.Error())
		}
		expectKeyExists(t, modelKey)
		expectSetContains(t, testModels.AllIndexKey(), model.Id())
	}
}

func TestDeleteStringIndexScript(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// Register a new model type with an indexed string field
	type stringIndexModel struct {
		String string `zoom:"index"`
		DefaultData
	}
	stringIndexModels, err := Register(&stringIndexModel{})
	if err != nil {
		t.Errorf("Unexpected error registering stringIndexModel: %s", err.Error())
	}

	// Create a new model (but don't save it yet)
	model := &stringIndexModel{
		String: "foo",
	}
	model.SetId("testId")

	// Run the script before saving the hash, to make sure it does not cause an error
	tx := NewTransaction()
	tx.deleteStringIndex(stringIndexModels.Name(), model.Id(), "String")
	if err := tx.Exec(); err != nil {
		t.Fatalf("Unexected error in tx.Exec: %s", err.Error())
	}

	// Set the field value in the main hash
	conn := Conn()
	defer conn.Close()
	modelKey, _ := stringIndexModels.ModelKey(model)
	if _, err := conn.Do("HSET", modelKey, "String", model.String); err != nil {
		t.Errorf("Unexpected error in HSET")
	}

	// Add the model to the index for the string field
	fieldIndexKey, err := stringIndexModels.FieldIndexKey("String")
	if err != nil {
		t.Fatalf("Unexpected error in FieldIndexKey: %s", err.Error())
	}
	member := model.String + " " + model.Id()
	if _, err := conn.Do("ZADD", fieldIndexKey, 0, member); err != nil {
		t.Fatalf("Unexpected error in ZADD: %s", err.Error())
	}

	// Run the script again. This time we expect the index to be removed
	tx = NewTransaction()
	tx.deleteStringIndex(stringIndexModels.Name(), model.Id(), "String")
	if err := tx.Exec(); err != nil {
		t.Fatalf("Unexected error in tx.Exec: %s", err.Error())
	}

	// Check that the index was removed
	expectIndexDoesNotExist(t, stringIndexModels, model, "String")
}

func TestExtractIdsFromStringIndexScript(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// Create and save some test models with increasing String values
	models := createIndexedTestModels(5)
	tx := NewTransaction()
	for i, model := range models {
		model.String = strconv.Itoa(i)
		tx.Save(indexedTestModels, model)
	}
	if err := tx.Exec(); err != nil {
		t.Errorf("Unexpected error saving models in tx.Exec: %s", err.Error())
	}

	// Test in both ascending and descending order
	for _, order := range []orderKind{ascendingOrder, descendingOrder} {
		expectedIds := []string{}
		switch order {
		case ascendingOrder:
			expectedIds = modelIds(Models(models))
		case descendingOrder:
			expectedIds = modelIds(Models(reverseModels(models)))
		}
		gotIds := []string{}
		tx := NewTransaction()
		fieldIndexKey, _ := indexedTestModels.FieldIndexKey("String")
		tx.extractIdsFromStringIndex(fieldIndexKey, order, newScanStringsHandler(&gotIds))
		if err := tx.Exec(); err != nil {
			t.Errorf("Unexpected error in tx.Exec: %s", err.Error())
		}
		if !reflect.DeepEqual(gotIds, expectedIds) {
			t.Errorf("Script results were incorrect.\nExpected: %v\nGot:  %v", expectedIds, gotIds)
		}
	}
}
