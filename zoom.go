package zoom

import (
	"errors"
	"fmt"
	"reflect"
)

// File contains glue code that connects the Model
// abstraction to the database. In other words,
// this is where the magic happens. The most important
// public-facing methods are here.

// writes the interface to the redis database
// throws an error if the type has not yet been
// registered. If in.Id is nil, will mutate in
// by setting the Id.
func Save(in Model) error {

	// make sure we'll dealing with a pointer
	typ := reflect.TypeOf(in)
	if typ.Kind() != reflect.Ptr {
		msg := fmt.Sprintf("zoom: Save() requires a pointer as an argument. The type %T is not a pointer.", in)
		return errors.New(msg)
	}

	return nil

}

// Removes the record from the database
func Delete(m Model) error {

	// get the registered name
	name, err := getRegisteredNameFromInterface(m)
	if err != nil {
		return err
	}

	// TODO: make sure it has an id

	return DeleteById(name, m.GetId())
}

// Delete a record from the interface by its id only
func DeleteById(modelName, id string) error {

	return nil
}

// Find a model by modelName and id. modelName must be the
// same name that was used in the Register() call
func FindById(modelName, id string) (interface{}, error) {

	return nil, nil
}

// ScanById is like FindById, but it will scan the results from the database
// into model, avoiding the need for typecasting after the find.
func ScanById(model Model, id string) error {

	return nil
}

func FindAll(modelName string) ([]interface{}, error) {

	return nil, nil
}
