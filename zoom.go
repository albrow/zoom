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
func Save(models ...Model) error {

	// start a transaction
	t := newTransaction()

	for _, m := range models {
		// make sure we'll dealing with a pointer
		typ := reflect.TypeOf(m)
		if typ.Kind() != reflect.Ptr {
			msg := fmt.Sprintf("zoom: Save() requires a pointer as an argument. The type %T is not a pointer.", m)
			return errors.New(msg)
		}

		// add a model save operation to the transaction
		if err := t.saveModel(m); err != nil {
			return err
		}
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return err
	}

	return nil

}

// Delete a model from the database
func Delete(models ...Model) error {
	t := newTransaction()
	for _, m := range models {
		// get the registered name
		modelName, err := getRegisteredNameFromInterface(m)
		if err != nil {
			return err
		}

		// make sure it has an id
		if m.GetId() == "" {
			return errors.New("zoom: cannot delete because model Id is blank")
		}

		// add a model delete operation to the transaction
		if err := t.deleteModel(modelName, m.GetId()); err != nil {
			return err
		}
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return err
	}

	return nil
}

// Delete a model from the database by its name and id
func DeleteById(modelName string, ids ...string) error {

	// start a transaction
	t := newTransaction()

	for _, id := range ids {
		// add a model delete operation to the transaction
		if err := t.deleteModel(modelName, id); err != nil {
			return err
		}
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return err
	}

	return nil
}
