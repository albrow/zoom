package zoom

// File contains glue code that connects the Model
// abstraction to the database. In other words,
// this is where the magic happens.

import (
	"errors"
	"fmt"
)

// writes the interface to the redis database
// throws an error if the type has not yet been
// registered
func (m *Model) Save() error {
	//fmt.Println("models.Save() was called")

	parent := m.Parent

	// get the registered name
	name, err := getRegisteredNameFromInterface(parent)
	if err != nil {
		return err
	}

	// prepare the arguments for redis driver
	// if no id was provided, we should generate one
	var id = m.Id
	if id == "" {
		id = generateRandomId()
	}

	key := name + ":" + id
	args, err := convertInterfaceToArgSlice(key, parent)
	if err != nil {
		return err
	}

	// invoke redis driver to commit to database
	result := db.Command("hmset", args...)
	if result.Error() != nil {
		return result.Error()
	}

	// return as a ModelInterface with the id set
	parentModel, ok := parent.(ModelInterface)
	if !ok {
		return errors.New("Couldn't convert to ModelInterface")
	}
	parentModel.SetId(id)
	return nil
}

// Removes the record from the database
func (m *Model) Delete() error {
	//fmt.Println("models.Delete() was called")

	parent := m.Parent

	// get the registered name
	name, err := getRegisteredNameFromInterface(parent)
	if err != nil {
		return err
	}

	// invoke redis driver to delete the key
	key := name + ":" + m.Id
	result := db.Command("del", key)
	if result.Error() != nil {
		return result.Error()
	}

	return nil
}

// Find a model by its id and then delete it
func DeleteById(modelName, id string) error {
	result, err := FindById(modelName, id)
	if err != nil {
		return err
	}
	model, ok := result.(ModelInterface)
	if !ok {
		return errors.New("Couldn't convert to ModelInterface")
	}

	return model.Delete()
}

// Find a model by modelName and id. modelName must be the
// same name that was used in the Register() call
func FindById(modelName, id string) (interface{}, error) {
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
	m, err := convertKeyValuesToModelInterface(keyValues, typ)
	if err != nil {
		return nil, err
	}

	// type assert the result to a ModelInterface
	model, ok := m.(ModelInterface)
	if !ok {
		return nil, errors.New("Couldn't convert to ModelInterface")
	}

	// Return the ModelInterface with the id set appropriately
	model.SetId(id)
	// whoa. I know what you're thinking. This next line of code
	// makes no sense. What's actually happening is that the SetParent
	// method has *Model as a reciever. *Model is embedded in something else
	// (a "parent") which has more attributes that we need to access (particularly
	// for the Save() method). So what we're really doing here is calling
	// something.Model.SetParent(something).
	model.SetParent(model)
	return model, nil
}
