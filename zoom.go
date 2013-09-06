package zoom

import (
	"errors"
	"fmt"
	"github.com/stephenalexbrowne/zoom/redis"
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
	if err := t.addModelSave(m); err != nil {
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
	if err := t.addModelDelete(modelName, id); err != nil {
		return err
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return err
	}

	return nil
}

// Find a model by modelName and id. modelName must be the
// same name that was used in the Register() call
func FindById(modelName, id string) (Model, error) {

	// get the type corresponding to the modelName
	typ, err := getRegisteredTypeFromName(modelName)
	if err != nil {
		return nil, err
	}

	// create a new struct of type typ
	val := reflect.New(typ.Elem())
	m, ok := val.Interface().(Model)
	if !ok {
		msg := fmt.Sprintf("zoom: could not convert val of type %T to Model", val.Interface())
		return nil, errors.New(msg)
	}

	// start a transaction
	t := newTransaction()

	// add a model find operation to the transaction
	if err := t.addModelFind(modelName, id, m); err != nil {
		return nil, err
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return nil, err
	}

	return m, nil
}

// ScanById is like FindById, but it will scan the results from the database
// into model, avoiding the need for typecasting after the find.
func ScanById(m Model, id string) error {

	// get the name corresponding to the type of m
	modelName, err := getRegisteredNameFromInterface(m)
	if err != nil {
		return err
	}

	// start a transaction
	t := newTransaction()

	// add a model find operation to the transaction
	if err := t.addModelFind(modelName, id, m); err != nil {
		return err
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return err
	}

	return nil
}

func FindAll(modelName string) ([]Model, error) {

	// get the type corresponding to the modelName
	typ, err := getRegisteredTypeFromName(modelName)
	if err != nil {
		return nil, err
	}

	// get a connection
	conn := GetConn()
	defer conn.Close()

	// get all the ids for the models
	indexKey := modelName + ":index"
	ids, err := redis.Strings(conn.Do("SMEMBERS", indexKey))
	if err != nil {
		return nil, err
	}

	// start a transaction
	t := newTransaction()

	// allocate a slice of Model
	models := make([]Model, len(ids))

	// iterate through the ids and add a find operation for each model
	for i, id := range ids {

		// instantiate a model using reflection
		modelVal := reflect.New(typ.Elem())
		m, ok := modelVal.Interface().(Model)
		if !ok {
			msg := fmt.Sprintf("zoom: could not convert val of type %T to Model", modelVal.Interface())
			return nil, errors.New(msg)
		}

		// set the ith element of models
		models[i] = m

		// add a find operation for the model m
		if err := t.addModelFind(modelName, id, m); err != nil {
			return nil, err
		}
	}

	// execute the transaction
	if err := t.exec(); err != nil {
		return nil, err
	}

	return models, nil
}
