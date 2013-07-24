package zoom

import (
	"fmt"
	"reflect"
)

func fieldIsRelational(field reflect.StructField) bool {
	n := field.Name
	if n[len(n)-2:] == "Id" || n[len(n)-3:] == "Ids" {
		tag := field.Tag
		if tag.Get("refersTo") != "" {
			return true
		}
	}
	return false
}

func relationalModelName(field reflect.StructField) string {
	return field.Tag.Get("refersTo")
}

// Takes as arguments: a field, the reflect.Value of the struct which contains field,
// and the index of that field.
// Verifies that:
// 		(1) the refersTo tag is a valid model name and has been registered
// 		(2) the value of the field is a valid id (the key exists)
// A return value of nil means that the relational field is valid
// Any other return value will be the error that was caused
func validateRelationalField(field reflect.StructField, val reflect.Value, i int) error {
	fieldVal := val.Field(i)
	relateName := relationalModelName(field)
	if !alreadyRegisteredName(relateName) {
		return NewModelNameNotRegisteredError(relateName)
	}
	if fieldVal.String() != "" {
		key := relateName + ":" + fieldVal.String()
		exists, err := keyExists(key)
		if err != nil {
			return err
		}
		if !exists {
			msg := fmt.Sprintf("Couldn't find %s with id = %s\n", relateName, fieldVal.String())
			return NewKeyNotFoundError(msg)
		}
	} else {
		// fmt.Println("relational field was empty")
		return nil
	}
	return nil
}
