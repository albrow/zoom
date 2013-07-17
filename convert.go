package zoom

// File contains code that converts between various data formats.
// It makes heavy use of the reflection package. This is the biggest
// speed bottleneck of the package, and should be constantly watched
// for opportunities for optimization.

import (
	"code.google.com/p/tcgl/redis"
	"fmt"
	"log"
	"reflect"
	"strconv"
)

// Converts a slice of redis.KeyValues into a map and returns it
func convertKeyValuesToMap(slice []*redis.KeyValue) map[string]string {
	m := make(map[string]string)
	for _, elem := range slice {
		key := elem.Key
		val := elem.Value.String()
		m[key] = val
	}
	return m
}

// Uses reflect to dynamically convert a map of
// [string]string to a ModelInterface (a struct)
// the keys of the map are the names of the fields in a struct of type typ
// the values of the map are the values of those corresponding fields
func convertMapToModelInterface(m map[string]string, typ reflect.Type) (ModelInterface, error) {
	typ = typ.Elem()
	val := reflect.New(typ).Elem()
	numFields := val.NumField()

	// iterate through each of the fields in the struct of type typ
	for i := 0; i < numFields; i++ {

		field := typ.Field(i)        // for getting field name/type/kind
		mutableField := val.Field(i) // for actually setting the field
		mapVal, ok := m[field.Name]  // the value from the map (will become field value)

		if ok {
			// convert each string into the appropriate type and then
			// add it to the struct.
			fieldKind := val.Field(i).Kind()
			fmt.Println("fieldKind: ", fieldKind)

			switch fieldKind {
			case reflect.String:
				mutableField.SetString(mapVal)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				valInt, err := strconv.Atoi(mapVal)
				if err != nil {
					return nil, err
				}
				mutableField.SetInt(int64(valInt))
			default:
				log.Panic("Don't know how to convert from string to " + fieldKind.String())
				// TODO: add more cases
			}
		}
	}

	// Set and allocate the embedded *Model attribute
	// This is so we can call SetId() later
	val.FieldByName("Model").Set(reflect.ValueOf(new(Model)))

	// Typecast and return the result
	model := val.Addr().Interface().(ModelInterface)
	return model, nil
}

// converts an interface and a given name to a slice of interface{}
// the slice can then be passed directly to the redis driver
func convertInterfaceToArgSlice(key string, in ModelInterface) []interface{} {

	// get the number of fields
	elem := reflect.ValueOf(in).Elem().Interface() // Get the actual element from the pointer
	val := reflect.ValueOf(elem)                   // for getting the actual field value
	typ := reflect.TypeOf(elem)                    // for name/type/kind information
	numFields := val.NumField()

	// init/allocate a slice of arguments
	args := make([]interface{}, 0, numFields*2+1)

	// the first arg is the key for the redis set
	args = append(args, key)

	// the remaining args are members of the redis set and their values.
	// iterate through fields and add each one to the slice
	for i := 0; i < numFields; i++ {
		field := typ.Field(i)
		// skip the embedded Model struct
		// that's used internally and doesn't belong in redis
		if field.Name == "Model" {
			continue
		}
		// the field name will the name of the member in redis
		args = append(args, field.Name)

		// the field value is the value of that member
		fieldVal := val.Field(i)
		valString := fmt.Sprint(fieldVal.Interface())
		args = append(args, valString)
	}

	return args
}
