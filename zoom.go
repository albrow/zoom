// Package zoom provides the top-level API for the zoom library. Zoom is lightweight,
// blazing-fast ORM powered by redis. It allows you to persist any arbitrary struct,
// preserve relationships between structs, retrieve structs by their id, and perform
// limited SQL-like queries.

// File zoom.go contains glue code that connects the Model
// interface to the database. The most basic
// public-facing methods are here.

package zoom

import (
	"errors"
	"fmt"
	"github.com/stephenalexbrowne/zoom/util"
	"reflect"
)

// Save writes an arbitrary struct or structs to the redis database.
// Structs which are savable (i.e. that implement the Model interface)
// will often be referred to as "models". Save throws an error if the type
// of the struct has not yet been registered. If the Id field of the struct
// is nil, Save will mutate the struct by setting the Id. To make a struct
// satisfy the Model interface, you can embed zoom.DefaultData.
func Save(models ...Model) error {
	t := newTransaction()
	for _, m := range models {

		// make sure we'll dealing with a pointer to a struct
		if !util.TypeIsPointerToStruct(reflect.TypeOf(m)) {
			msg := fmt.Sprintf("zoom: Save() requires a pointer to a struct as an argument.\nThe type %T is not a pointer to a struct.", m)
			return errors.New(msg)
		}

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

// Delete removes a struct (or structs) from the database. Will
// throw an error if the type of the struct has not yet been
// registered, or if the Id field of the struct is empty.
func Delete(models ...Model) error {
	t := newTransaction()
	for _, m := range models {
		if m.GetId() == "" {
			return errors.New("zoom: cannot delete because model Id field is empty")
		}
		modelName, err := getRegisteredNameFromInterface(m)
		if err != nil {
			return err
		}
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

// DeleteById removes a struct (or structs) from the database by its
// registered name and id. The modelName argument should be the same
// string name that was used in the Register function. If using variadic
// paramaters, you can only delete models of the same registered name and type.
func DeleteById(modelName string, ids ...string) error {
	t := newTransaction()
	for _, id := range ids {
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
