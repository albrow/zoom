package zoom

import (
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"reflect"
)

// File contains glue code that connects the Model
// abstraction to the database. In other words,
// this is where the magic happens.

// writes the interface to the redis database
// throws an error if the type has not yet been
// registered
func Save(in ModelInterface) error {
	// get the registered name
	name, err := getRegisteredNameFromInterface(in)
	if err != nil {
		return err
	}

	// prepare the arguments for redis driver
	// if no id was provided, we should generate one
	id := in.GetId()
	if id == "" {
		id = generateRandomId()
	}
	key := name + ":" + id

	// invoke redis driver to commit to database
	_, err = db.Do("hmset", redis.Args{}.Add(key).AddFlat(in)...)
	if err != nil {
		return err
	}
	in.SetId(id)

	return nil
}

// Removes the record from the database
func Delete(in ModelInterface) error {

	// get the registered name
	name, err := getRegisteredNameFromInterface(in)
	if err != nil {
		return err
	}

	// TODO: make sure it has an id

	// invoke redis driver to delete the key
	key := name + ":" + in.GetId()
	_, err = db.Do("del", key)
	if err != nil {
		return err
	}

	return nil
}

// Find a model by its id and then delete it
func DeleteById(modelName, id string) error {
	key := modelName + ":" + id
	_, err := db.Do("del", key)
	return err
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
	exists, err := KeyExists(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		msg := fmt.Sprintf("Couldn't find %s with id = %s", modelName, id)
		return nil, NewKeyNotFoundError(msg)
	}

	// get the stuff from redis
	reply, err := db.Do("hgetall", key)
	bulk, err := redis.MultiBulk(reply, err)
	if err != nil {
		return nil, err
	}

	modelVal := reflect.New(typ.Elem())
	model := modelVal.Interface()
	modelI, ok := model.(ModelInterface)
	if !ok {
		return nil, errors.New("Couldn't convert to ModelInterface. Does interface implement SetId and GetId?")
	}
	modelI.SetId(id)

	err = redis.ScanStruct(bulk, model)
	if err != nil {
		return nil, err
	}
	return model, nil
}
