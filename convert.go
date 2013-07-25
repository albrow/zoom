package zoom

// File contains code that converts between various data formats.
// It makes heavy use of the reflection package. This is the biggest
// speed bottleneck of the package, and should be constantly watched
// for opportunities for optimization.

import (
	"code.google.com/p/tcgl/redis"
	"log"
	"reflect"
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
// [string]string to an interface{} (a struct)
// the keys of the map are the names of the fields in a struct of type typ
// the values of the map are the values of those corresponding fields
func convertKeyValuesToModelInterface(keyValues []*redis.KeyValue, typ reflect.Type) (interface{}, error) {
	typ = typ.Elem()
	val := reflect.New(typ).Elem()

	// iterate through each item in keyValues and add it to the struct
	for _, keyValue := range keyValues {
		key := keyValue.Key
		field := val.FieldByName(key)
		if field.CanSet() {
			value := keyValue.Value
			switch field.Type().Kind() {
			case reflect.String:
				field.SetString(value.String())
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				valInt, err := value.Int64()
				if err != nil {
					return nil, err
				}
				field.SetInt(valInt)
			default:
				log.Panic("Don't know how to add %s to a struct!", field.Type().Kind().String())
				// TODO: add more cases
			}
		}
	}

	// Set and allocate the embedded *Model attribute
	// This is so we can call SetId() later
	val.FieldByName("Model").Set(reflect.ValueOf(new(Model)))

	// Typecast and return the result
	model := val.Addr().Interface()
	return model, nil
}

// converts an interface and a given name to a slice of interface{}
// the slice can then be passed directly to the redis driver
func convertInterfaceToArgSlice(key string, in interface{}) ([]interface{}, error) {

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
		// there's a special case for relational attributes
		// a.k.a. those which include Id in the name and are
		// tagged with `refersTo:*`
		if fieldIsRelational(field) {
			//fmt.Println("Detected relational field: ", field.Name)
			err := validateRelationalField(field)
			if err != nil {
				return nil, err
			}
		}

		// the field name will the name of the member in redis
		args = append(args, field.Name)

		// the field value is the value of that member
		fieldVal := val.Field(i)
		args = append(args, fieldVal.Interface())
	}

	return args, nil
}
