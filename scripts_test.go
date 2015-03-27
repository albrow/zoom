package zoom

import (
	"github.com/garyburd/redigo/redis"
	"reflect"
	"testing"
)

func TestFindModelsBySetIdsScript(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// Create and save some test models
	models, err := createAndSaveTestModels(3)
	if err != nil {
		t.Fatalf("Unexpected error saving test models: %s", err.Error())
	}
	modelsById := map[string]*testModel{}
	for _, model := range models {
		modelsById[model.Id] = model
	}

	// Run the script
	tx := NewTransaction()
	var gotReply interface{}
	tx.findModelsBySetIds(testModels.AllIndexKey(), testModels.Name(), func(reply interface{}) error {
		gotReply = reply
		return nil
	})
	if err := tx.Exec(); err != nil {
		t.Fatalf("Unexected error in tx.Exec: %s", err.Error())
	}

	// Check that the return value is correct
	modelsReplies, err := redis.Values(gotReply, nil)
	if err != nil {
		t.Fatalf("Unexpected error in redis.Values: %s", err.Error())
	}
	for i, reply := range modelsReplies {
		replies, err := redis.Values(reply, nil)
		if err != nil {
			t.Fatalf("Unexpected error in redis.Values: %s", err.Error())
		}
		gotFields := map[string]interface{}{}
		for i := 0; i < len(replies); i += 2 {
			fieldName, err := redis.String(replies[i], nil)
			if err != nil {
				t.Fatalf("Unexpected error in redis.String: %s", err.Error())
			}
			fieldValue := replies[i+1]
			gotFields[fieldName] = fieldValue
		}
		if _, found := gotFields["Id"]; !found {
			t.Errorf("reply %d did not have an Id field", i)
		}
		id := string(gotFields["Id"].([]byte))
		expectedModel, found := modelsById[id]
		if !found {
			t.Errorf("reply had incorrect id. Could not find id %s in %v", id, modelsById)
		}
		expectedModelVal := reflect.ValueOf(expectedModel).Elem()
		for fieldName, gotVal := range gotFields {
			var convertedVal interface{}
			var err error
			switch fieldName {
			case "Id":
				continue // We already checked the id field
			case "Int":
				convertedVal, err = redis.Int(gotVal, nil)
			case "String":
				convertedVal, err = redis.String(gotVal, nil)
			case "Bool":
				convertedVal, err = redis.Bool(gotVal, nil)
			}
			if err != nil {
				t.Errorf("Unexpected error converting field %s: %s", fieldName, err.Error())
			}
			expectedField := expectedModelVal.FieldByName(fieldName)
			if !expectedField.IsValid() {
				t.Errorf("Could not find field %s in %T", fieldName, expectedModel)
			}
			if !reflect.DeepEqual(expectedField.Interface(), convertedVal) {
				t.Errorf("Field %s was incorrect. Expected %v but got %v", fieldName, expectedField.Interface(), convertedVal)
			}
		}
	}
}

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
		ids = append(ids, model.Id)
	}
	ids = append(ids, "foo", "bar")
	tempSetKey := "testModelIds"
	conn := GetConn()
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
		modelKey, err := testModels.KeyForModel(model)
		if err != nil {
			t.Errorf("Unexpected error in KeyForModel: %s", err.Error())
		}
		expectKeyDoesNotExist(t, modelKey)
		expectSetDoesNotContain(t, testModels.AllIndexKey(), model.Id)
	}
	// Make sure the last two models were not deleted
	for _, model := range models[3:] {
		modelKey, err := testModels.KeyForModel(model)
		if err != nil {
			t.Errorf("Unexpected error in KeyForModel: %s", err.Error())
		}
		expectKeyExists(t, modelKey)
		expectSetContains(t, testModels.AllIndexKey(), model.Id)
	}
}

func TestSaveStringIndexScript(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	type stringIndexModel struct {
		String string `zoom:"index"`
		DefaultData
	}
	stringIndexModels, err := Register(&stringIndexModel{})
	if err != nil {
		t.Errorf("Unexpected error registering stringIndexModel: %s", err.Error())
	}
	model := &stringIndexModel{
		String: "foo",
	}
	model.Id = "testId"

	// Run the script without an old field value
	tx := NewTransaction()
	tx.saveStringIndex(stringIndexModels.Name(), model.Id, "String", model.String)
	if err := tx.Exec(); err != nil {
		t.Fatalf("Unexected error in tx.Exec: %s", err.Error())
	}

	// Check that the index was set correctly
	expectIndexExists(t, stringIndexModels, model, "String")

	// Set the field value in the main hash
	conn := GetConn()
	defer conn.Close()
	modelKey, _ := stringIndexModels.KeyForModel(model)
	if _, err := conn.Do("HSET", modelKey, "String", model.String); err != nil {
		t.Errorf("Unexpected error in HSET")
	}

	// Create a new model with the same id and change the value of the field
	newModel := &stringIndexModel{
		String: "bar",
	}
	newModel.Id = model.Id

	// Run the script again. This time we expect the old index to be removed
	tx = NewTransaction()
	tx.saveStringIndex(stringIndexModels.Name(), newModel.Id, "String", newModel.String)
	if err := tx.Exec(); err != nil {
		t.Fatalf("Unexected error in tx.Exec: %s", err.Error())
	}

	// Check that the index was set correctly
	expectIndexDoesNotExist(t, stringIndexModels, model, "String")
	expectIndexExists(t, stringIndexModels, newModel, "String")
}
