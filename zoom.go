package zoom

import (
	"errors"
	"fmt"
	"github.com/stephenalexbrowne/zoom/redis"
	"reflect"
)

// File contains glue code that connects the Model
// abstraction to the database. In other words,
// this is where the magic happens.

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

	// invoke redis driver to commit to database
	_, err = conn.Do("hmset", Args{}.Add(key).AddFlat(in)...)
	if err != nil {
		return err
	}

	// add to the index for this model
	err = addToIndex(name, in.GetId(), conn)
	if err != nil {
		return err
	}

	// save the relations
	err = saveRelations(in, val, ss, name, conn)
	if err != nil {
		return err
	}

	return nil
}

// iterate through relations and save each one
func saveRelations(in ModelInterface, val reflect.Value, ss *structSpec, name string, conn redis.Conn) error {
	for _, r := range ss.relations {

		// make sure we're dealing with a valid value
		relVal := val.Elem().FieldByName(r.fieldName)
		if !relVal.IsValid() {
			msg := fmt.Sprintf("zoom: Could not find field %s in %T. Value was %+v\n", r.fieldName, val.Elem().Interface(), val.Elem().Interface())
			return errors.New(msg)
		}
		if relVal.IsNil() {
			// skip empty relations
			continue
		}

		if r.typ == ONE_TO_ONE {

			// commit to database
			relModel, err := commitRelation(r, relVal, conn)
			if err != nil {
				return err
			}

			// add a relation key to the parent interface (in)
			key := name + ":" + in.GetId() + ":" + r.fieldName
			_, err = conn.Do("set", key, relModel.GetId())
			if err != nil {
				return err
			}

		} else if r.typ == ONE_TO_MANY {
			// iterate through the array and save each one
			for i := 0; i < relVal.Len(); i++ {
				relElem := relVal.Index(i)

				// commit to database
				relModel, err := commitRelation(r, relElem, conn)
				if err != nil {
					return err
				}

				// add a relation key to the parent interface (in)
				key := name + ":" + in.GetId() + ":" + r.fieldName
				_, err = conn.Do("sadd", key, relModel.GetId())
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// commits the relational struct to database and returns a ModelInterface that can be used
// to get the id
func commitRelation(r *relation, relVal reflect.Value, conn redis.Conn) (ModelInterface, error) {
	// cast to a ModelInterface
	relModel, ok := relVal.Interface().(ModelInterface)
	if !ok {
		msg := fmt.Sprintf("The type %T does not implement ModelInterface. Does it have a *zoom.Model embedded field?", relVal.Interface())
		return nil, errors.New(msg)
	}

	// assign an id if needed
	if relModel.GetId() == "" {
		relModel.SetId(generateRandomId())
	}

	// format the args
	key := r.redisName + ":" + relModel.GetId()
	args := Args{}.Add(key)
	args = append(args, argsForSpec(relVal.Elem(), r.spec)...)

	// invoke redis driver to commit to database
	_, err := conn.Do("hmset", args...)
	if err != nil {
		return nil, err
	}

	// add to the index for this model
	err = addToIndex(r.redisName, relModel.GetId(), conn)
	if err != nil {
		return nil, err
	}

	return relModel, nil
}

// Removes the record from the database
func Delete(in ModelInterface) error {

	// get the registered name
	name, err := getRegisteredNameFromInterface(in)
	if err != nil {
		return err
	}
	key := name + ":" + in.GetId()

	// TODO: make sure it has an id

	// get a connection
	conn := pool.Get()
	defer conn.Close()

	// invoke redis driver to delete the key
	_, err = conn.Do("del", key)
	if err != nil {
		return err
	}

	// remove from the index
	key = name + ":index"
	_, err = conn.Do("srem", key, in.GetId())
	if err != nil {
		return err
	}

	return nil
}

// Find a model by its id and then delete it
func DeleteById(modelName, id string) error {

	key := modelName + ":" + id

	// get a connection
	conn := pool.Get()
	defer conn.Close()

	_, err := conn.Do("del", key)
	if err != nil {
		return err
	}

	// remove from the index
	key = modelName + ":index"
	_, err = conn.Do("srem", key, id)
	if err != nil {
		return err
	}

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

	// get the stuff from redis
	reply, err := conn.Do("hgetall", key)
	bulk, err := redis.MultiBulk(reply, err)
	if err != nil {
		return nil, err
	}

	// create a new struct and instantiate its Model attribute
	// this gives us the embedded methods and properties on Model
	modelVal := reflect.New(typ.Elem())
	modelVal.Elem().FieldByName("Model").Set(reflect.ValueOf(new(Model)))

	// type assert to ModelInterface so we can use SetId()
	model := modelVal.Interface().(ModelInterface)

	// invoke redis driver to fill in the values of the struct
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

func scanRelations(ss *structSpec, modelName string, modelId string, modelVal reflect.Value, conn redis.Conn) error {
	for _, r := range ss.relations {
		if r.typ == ONE_TO_ONE {
			key := modelName + ":" + modelId + ":" + r.fieldName
			exists, err := KeyExists(key, conn)
			if err != nil {
				return err
			}
			if exists {
				if err := scanOneToOneRelation(r, modelVal, key, conn); err != nil {
					return err
				}
			}
		} else if r.typ == ONE_TO_MANY {
			key := modelName + ":" + modelId + ":" + r.fieldName
			exists, err := KeyExists(key, conn)
			if err != nil {
				return err
			}
			if exists {
				scanOneToManyRelation(r, modelVal, key, conn)
			}
		}
	}
	return nil
}

func scanOneToOneRelation(r *relation, modelVal reflect.Value, key string, conn redis.Conn) error {
	// get the id of the relation from redis
	relId, err := redis.String(conn.Do("get", key))
	if err != nil {
		return err
	}

	// get the settable field
	relField := modelVal.Elem().FieldByName(r.fieldName)

	// make sure we can set relField
	if !relField.CanSet() {
		msg := fmt.Sprintf("zoom: couldn't set field: %+v\n", relField)
		return errors.New(msg)
	}

	// invoke redis to find and scan the relation into the struct represented by modelVal
	if err := scanRelation(r, relId, relField, conn); err != nil {
		return err
	}

	return nil
}

func scanOneToManyRelation(r *relation, modelVal reflect.Value, key string, conn redis.Conn) error {
	// invoke redis driver to get a set of keys
	relIds, err := redis.Strings(conn.Do("smembers", key))
	if err != nil {
		return err
	}

	for _, relId := range relIds {

		// create a settable slice element
		relField := modelVal.Elem().FieldByName(r.fieldName)

		// invoke redis to find and scan the relation into the struct represented by modelVal
		if err := scanRelation(r, relId, relField, conn); err != nil {
			return err
		}
	}
	return nil
}

func scanRelation(r *relation, relId string, settable reflect.Value, conn redis.Conn) error {

	// see if the relation has been saved in redis
	// if not we should just leave it as nil
	key := r.redisName + ":" + relId
	exists, err := KeyExists(key, conn)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	// get the stuff from redis
	bulk, err := redis.MultiBulk(conn.Do("hgetall", key))
	if err != nil {
		return err
	}

	// create a new struct and instantiate its Model attribute
	// this gives us the embedded methods and properties on Model
	var relVal reflect.Value
	if r.typ == ONE_TO_ONE {
		relVal = reflect.New(settable.Type().Elem())
	} else {
		relVal = reflect.New(settable.Type().Elem().Elem())
	}
	relVal.Elem().FieldByName("Model").Set(reflect.ValueOf(new(Model)))

	// type assert to ModelInterface so we can use SetId()
	relModel := relVal.Interface().(ModelInterface)

	// invoke redis driver to fill in the values of the struct
	err = ScanStruct(bulk, relModel)
	if err != nil {
		return err
	}

	relModel.SetId(relId)

	if r.typ == ONE_TO_ONE {
		settable.Set(relVal)
	} else if r.typ == ONE_TO_MANY {
		sliceVal := reflect.Append(settable, relVal)
		settable.Set(sliceVal)
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
