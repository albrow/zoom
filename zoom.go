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
// registered. If in.Id is nil, will mutate in
// by setting the Id.
func Save(in ModelInterface) error {

	// make sure we'll dealing with a pointer
	typ := reflect.TypeOf(in)
	if typ.Kind() != reflect.Ptr {
		msg := fmt.Sprintf("zoom: Save() requires a pointer as an argument. The type %T is not a pointer.", in)
		return errors.New(msg)
	}

	// get the value
	val := reflect.ValueOf(in)
	if val.IsNil() {
		return errors.New("zoom: attempted to call save on a nil pointer!")
	}

	// get a connection from the pool
	conn := pool.Get()
	defer conn.Close()

	// get the struct spec
	ss := structSpecForType(typ.Elem())

	// get the registered name
	name, err := getRegisteredNameFromInterface(in)
	if err != nil {
		return err
	}

	// prepare the arguments for redis driver
	// if no id was provided, we should generate one
	if in.GetId() == "" {
		in.SetId(generateRandomId())
	}
	key := name + ":" + in.GetId()

	// start a multi/exec command. i.e. create a command queue
	if err := conn.Send("multi"); err != nil {
		// cancel transaction and return err
		conn.Do("discard")
		return err
	}

	// add command to queue
	if err := conn.Send("hmset", Args{}.Add(key).AddFlat(in)...); err != nil {
		// cancel transaction and return err
		conn.Do("discard")
		return err
	}

	// add to the index for this model
	err = queueAddToIndex(name, in.GetId(), conn)
	if err != nil {
		return err
	}

	// save the relations
	err = saveRelations(in, val, ss, name, conn)
	if err != nil {
		return err
	}

	// finally, commit the transaction
	// they were all writes, so the return value isn't needed
	if _, err := conn.Do("exec"); err != nil {
		return err
	}

	// add to the cache
	modelCache.Set(key, newCacheValue(in))

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

	return DeleteById(name, in.GetId())
}

// Delete a record from the interface by its id only
func DeleteById(modelName, id string) error {

	key := modelName + ":" + id

	// get a connection
	conn := pool.Get()
	defer conn.Close()

	// start a transaction
	conn.Send("multi")

	// add a command to the queue which will
	// delete the main key
	if err := conn.Send("del", key); err != nil {
		return err
	}

	// add a command to the queue which will
	// remove it from the index
	key = modelName + ":index"
	if err := conn.Send("srem", key, id); err != nil {
		return err
	}

	// execute the commands
	_, err := conn.Do("exec")
	if err != nil {
		return err
	}

	// remove from the cache
	modelCache.Delete(key)

	return nil
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

	// check if the model is in the cache
	val, found := modelCache.Get(key)
	if found {
		cv, ok := val.(*cacheValue)
		if !ok {
			return nil, errors.New("zoom: Got from cache but couldn't convert to cacheValue")
		}
		return cv.value, nil
	}

	// open a connection
	conn := pool.Get()
	defer conn.Close()

	// make sure the key exists
	exists, err := KeyExists(key, conn)
	if err != nil {
		return nil, err
	}
	if !exists {
		msg := fmt.Sprintf("Couldn't find %s with id = %s", modelName, id)
		return nil, NewKeyNotFoundError(msg)
	}

	// create a new struct and instantiate its Model attribute
	// this gives us the embedded methods and properties on Model
	modelVal := reflect.New(typ.Elem())
	modelVal.Elem().FieldByName("Model").Set(reflect.ValueOf(new(Model)))

	// type assert to ModelInterface so we can use SetId()
	model := modelVal.Interface().(ModelInterface)

	// get the field values from redis
	reply, err := conn.Do("hgetall", key)
	bulk, err := redis.MultiBulk(reply, err)
	if err != nil {
		return nil, err
	}

	// fill in the values of the struct
	err = ScanStruct(bulk, model)
	if err != nil {
		return nil, err
	}

	// set the id
	model.SetId(id)

	// scan relations and add them as attributes to model
	ss := structSpecForType(typ.Elem())
	if err := scanRelations(ss, modelName, id, modelVal, conn); err != nil {
		return nil, err
	}

	// return it
	return model, nil
}

// ScanById is like FindById, but it will scan the results from the database
// into model, avoiding the need for typecasting after the find.
func ScanById(model ModelInterface, id string) error {

	// get the type and name
	typ := reflect.TypeOf(model)
	modelName, found := typeToName[typ]
	if !found {
		return NewModelTypeNotRegisteredError(typ)
	}

	// create the key based on the modelName and id
	key := modelName + ":" + id

	// check if the model is in the cache
	val, found := modelCache.Get(key)
	if found {
		cv, ok := val.(*cacheValue)
		if !ok {
			return errors.New("zoom: Got from cache but couldn't convert to cacheValue")
		}
		modelVal := reflect.ValueOf(model).Elem()
		modelVal.Set(reflect.ValueOf(cv.value).Elem())
		return nil
	}

	// open a connection
	conn := pool.Get()
	defer conn.Close()

	// make sure the key exists
	exists, err := KeyExists(key, conn)
	if err != nil {
		return err
	}
	if !exists {
		msg := fmt.Sprintf("Couldn't find %s with id = %s", modelName, id)
		return NewKeyNotFoundError(msg)
	}

	// get the stuff from redis
	reply, err := conn.Do("hgetall", key)
	bulk, err := redis.MultiBulk(reply, err)
	if err != nil {
		return err
	}

	// create a new struct and instantiate its Model attribute
	// this gives us the embedded methods and properties on Model
	modelVal := reflect.ValueOf(model)
	modelVal.Elem().FieldByName("Model").Set(reflect.ValueOf(new(Model)))

	// invoke redis driver to fill in the values of the struct
	err = ScanStruct(bulk, model)
	if err != nil {
		return err
	}

	// set the id
	model.SetId(id)

	// scan relations and add them as attributes to model
	ss := structSpecForType(typ.Elem())
	if err := scanRelations(ss, modelName, id, modelVal, conn); err != nil {
		return err
	}

	return nil
}

func FindAll(modelName string) ([]interface{}, error) {

	// get a connection
	conn := pool.Get()
	defer conn.Close()

	// invoke redis driver to get indexed keys and convert to []interface{}
	key := modelName + ":index"
	ids, err := redis.Strings(conn.Do("smembers", key))
	if err != nil {
		return nil, err
	}

	// iterate through each id. find the corresponding model. append to results.
	results := make([]interface{}, len(ids), len(ids))
	for i, id := range ids {
		m, err := FindById(modelName, id)
		if err != nil {
			return nil, err
		}
		results[i] = m
	}

	return results, nil
}
