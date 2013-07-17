package zoom

// File contains glue code that connects the Model
// abstraction to the database. In other words,
// this is where the magic happens.

import (
	"code.google.com/p/tcgl/redis"
	"fmt"
	"github.com/dchest/uniuri"
	"log"
	"reflect"
	"strconv"
	"time"
)

// writes the interface to the redis database
// throws an error if the type has not yet been
// registered
func Save(m ModelInterface) (ModelInterface, error) {
	fmt.Println("models.Save() was called")

	// make sure we're dealing with a pointer
	if reflect.TypeOf(m).Kind() != reflect.Ptr {
		return nil, NewInterfaceIsNotPointerError(m)
	}

	// get the registered name
	name, err := getRegisteredNameFromInterface(m)
	if err != nil {
		return nil, err
	}

	// prepare the arguments for redis driver
	id := generateRandomId()
	key := name + ":" + id
	args := convertInterfaceToArgSlice(key, m)

	// invoke redis driver to commit to database
	result := db.Command("hmset", args...)
	if result.Error() != nil {
		return nil, result.Error()
	}

	// return as a ModelInterface with the id set
	m.SetId(id)
	return m, nil
}

// TODO: remove the record from the database
func Delete(m ModelInterface) (ModelInterface, error) {
	fmt.Println("models.Delete() was called")

	// make sure we're dealing with a pointer
	if reflect.TypeOf(m).Kind() != reflect.Ptr {
		return nil, NewInterfaceIsNotPointerError(m)
	}

	// get the registered name
	name, err := getRegisteredNameFromInterface(m)
	if err != nil {
		return nil, err
	}

	// invoke redis driver to delete the key
	key := name + ":" + m.GetId()
	result := db.Command("del", key)
	if result.Error() != nil {
		return nil, result.Error()
	}

	return m, nil
}

// Find a model by it's id and then delete it
func DeleteById(modelName, id string) (ModelInterface, error) {
	model, err := FindById(modelName, id)
	if err != nil {
		return nil, err
	}

	return Delete(model)
}

// Find a model by modelName and id. modelName must be the
// same name that was used for in the Register() call
func FindById(modelName, id string) (ModelInterface, error) {
	// get the registered type
	typ, err := getRegisteredTypeFromName(modelName)
	if err != nil {
		return nil, err
	}

	// create the key based on the modelName and id
	key := modelName + ":" + id

	// make sure the key exists
	exists, err := keyExists(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		msg := fmt.Sprintf("Couldn't find %s with id = %s", modelName, id)
		return nil, NewKeyNotFoundError(msg)
	}

	// get the stuff from redis
	result := db.Command("hgetall", key)
	if result.Error() != nil {
		return nil, result.Error()
	}

	// convert the redis result to a struct of type typ
	// It's called a ModelInterface here but the underlying
	// type is still typ
	keyValues := result.KeyValues()
	resultMap := convertKeyValuesToMap(keyValues)
	model, err := convertMapToModelInterface(resultMap, typ)
	if err != nil {
		return nil, err
	}

	// Return the ModelInterface with the id set appropriately
	model.SetId(id)
	return model, nil
}

// get the registered name of the model we're trying to save
// based on the interfaces type. If the interface's type has
// not been registered, returns a ModelTypeNotRegisteredError
func getRegisteredNameFromInterface(m ModelInterface) (string, error) {
	typ := reflect.TypeOf(m)
	name, ok := typeToName[typ]
	if !ok {
		return "", NewModelTypeNotRegisteredError(typ)
	}
	return name, nil
}

func getRegisteredTypeFromName(name string) (reflect.Type, error) {
	typ, ok := nameToType[name]
	if !ok {
		return nil, NewModelNameNotRegisteredError(name)
	}
	return typ, nil
}

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

// Returns true iff a given key exists in redis
func keyExists(key string) (bool, error) {
	result := db.Command("exists", key)
	if result.Error() != nil {
		return false, result.Error()
	}

	return result.ValueAsBool()
}

// generates a random string that is more or less
// garunteed to be unique
func generateRandomId() string {
	timeInt := time.Now().Unix()
	timeString := strconv.FormatInt(timeInt, 36)
	randomString := uniuri.NewLen(16)
	return randomString + timeString
}
