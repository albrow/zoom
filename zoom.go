package zoom

// File contains glue code that connects the Model
// abstraction to the database. In other words,
// this is where the magic happens.

import (
	"fmt"
	"reflect"
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

// Removes the record from the database
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
// same name that was used in the Register() call
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
	model, err := convertKeyValuesToModelInterface(keyValues, typ)
	if err != nil {
		return nil, err
	}

	// Return the ModelInterface with the id set appropriately
	model.SetId(id)
	return model, nil
}
