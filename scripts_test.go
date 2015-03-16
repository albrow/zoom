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
	tx := newTransaction()
	var gotReply interface{}
	tx.findModelsBySetIds(testModels.KeyForAll(), testModels.Name(), func(reply interface{}) error {
		gotReply = reply
		return nil
	})
	if err := tx.exec(); err != nil {
		t.Fatalf("Unexected error in tx.exec: %s", err.Error())
	}
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
