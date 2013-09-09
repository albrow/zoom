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
// registered. If m.Id is nil, will mutate m
// by setting the Id.
func Save(m Model) error {

	// make sure we'll dealing with a pointer
	typ := reflect.TypeOf(m)
	if typ.Kind() != reflect.Ptr {
		msg := fmt.Sprintf("zoom: Save() requires a pointer as an argument. The type %T is not a pointer.", m)
		return errors.New(msg)
	}

	// start a transaction
	t := newTransaction()

	// add a model save operation to the transaction
	if err := t.saveModel(m); err != nil {
		return err
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return err
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

	// make sure it has an id
	if m.GetId() == "" {
		return errors.New("zoom: cannot delete because model Id is blank")
	}

	return DeleteById(name, m.GetId())
}

// Delete a record from the interface by its id only
func DeleteById(modelName, id string) error {

	// start a transaction
	t := newTransaction()

	// add a model delete operation to the transaction
	if err := t.deleteModel(modelName, id); err != nil {
		return err
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return err
	}

	return nil
}
