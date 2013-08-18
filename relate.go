package zoom

import (
	"errors"
	"fmt"
	"github.com/stephenalexbrowne/zoom/redis"
	"reflect"
)

// File contains code strictly having to do with relations. This includes
// saving relations in the database and converting the database representation
// back into the standard go struct form.

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
			relModel, err := queueRelation(r, relVal, conn)
			if err != nil {
				return err
			}

			// add a command to the queue which will
			// add a relation key to the parent interface (in)
			key := name + ":" + in.GetId() + ":" + r.fieldName
			if err := conn.Send("set", key, relModel.GetId()); err != nil {
				// cancel transaction and return err
				conn.Do("discard")
				return err
			}

		} else if r.typ == ONE_TO_MANY {
			// iterate through the array and save each one
			for i := 0; i < relVal.Len(); i++ {
				relElem := relVal.Index(i)

				// commit to database
				relModel, err := queueRelation(r, relElem, conn)
				if err != nil {
					return err
				}

				// add a command to the queue which will
				// add a relation key to the parent interface (in)
				key := name + ":" + in.GetId() + ":" + r.fieldName
				if err := conn.Send("sadd", key, relModel.GetId()); err != nil {
					// cancel transaction and return err
					conn.Do("discard")
					return err
				}
			}
		}
	}
	return nil
}

// adds a command to the queue to add the relational struct to database and
// returns a ModelInterface that can be used
func queueRelation(r *relation, relVal reflect.Value, conn redis.Conn) (ModelInterface, error) {
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

	// add a command to the queue which will
	// add the struct to the database as a redis hash
	if err := conn.Send("hmset", args...); err != nil {
		// cancel transaction and return err
		conn.Do("discard")
		return nil, err
	}

	// add to the index for this model
	if err := queueAddToIndex(r.redisName, relModel.GetId(), conn); err != nil {
		return nil, err
	}

	return relModel, nil
}

func scanRelations(ss *structSpec, modelName string, modelId string, modelVal reflect.Value, conn redis.Conn) error {
	// iterate through each relation in the spec
	for _, r := range ss.relations {

		// construct the key for the related model
		key := modelName + ":" + modelId + ":" + r.fieldName

		// make sure the key exists
		exists, err := KeyExists(key, conn)
		if err != nil {
			return err
		}
		if exists {
			if r.typ == ONE_TO_ONE {
				if err := scanOneToOneRelation(r, modelVal, key, conn); err != nil {
					return err
				}
			} else if r.typ == ONE_TO_MANY {
				if err := scanOneToManyRelation(r, modelVal, key, conn); err != nil {
					return err
				}
			}
		} // if it doesn't exist, we do nothing
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

	// see if the relation has been saved in redis
	// if not we should just leave it as nil
	key = r.redisName + ":" + relId
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
	relVal = reflect.New(relField.Type().Elem())
	relVal.Elem().FieldByName("Model").Set(reflect.ValueOf(new(Model)))

	// type assert to ModelInterface so we can use SetId()
	relModel := relVal.Interface().(ModelInterface)

	// invoke redis driver to fill in the values of the struct
	err = ScanStruct(bulk, relModel)
	if err != nil {
		return err
	}

	// set the id
	relModel.SetId(relId)

	// set relField to the value of the new struct we just created
	relField.Set(relVal)

	return nil
}

func scanOneToManyRelation(r *relation, modelVal reflect.Value, key string, conn redis.Conn) error {
	// invoke redis driver to get a set of keys
	relIds, err := redis.Strings(conn.Do("smembers", key))
	if err != nil {
		return err
	}

	// setup a transaction
	if err := conn.Send("multi"); err != nil {
		return err
	}

	// We'll pipeline this by first executing all the redis queries,
	// then dealing with each response.
	for _, relId := range relIds {
		if err := queueOneToManyRelation(r, relId, conn); err != nil {
			return err
		}
	}

	// call EXEC to execute all the commands in the transaction queue
	replies, err := redis.MultiBulk(conn.Do("exec"))
	if err != nil {
		return err
	}

	// now iterate through each reply and create structs, then append them to the slice
	for _, reply := range replies {

		// convert each reply to a slice
		bulk, err := redis.MultiBulk(reply, nil)
		if err != nil {
			return err
		}

		// create a settable slice
		settable := modelVal.Elem().FieldByName(r.fieldName)

		// create a new struct and instantiate its Model attribute
		// this gives us the embedded methods and properties on Model
		relVal := reflect.New(settable.Type().Elem().Elem())
		relVal.Elem().FieldByName("Model").Set(reflect.ValueOf(new(Model)))

		// type assert to ModelInterface so we can use SetId()
		relModel := relVal.Interface().(ModelInterface)

		// fill in the values of the struct
		err = ScanStruct(bulk, relModel)
		if err != nil {
			return err
		}

		// set the id
		// TODO: fix this!
		// relModel.SetId(relId)

		// append to the slice of related structs
		sliceVal := reflect.Append(settable, relVal)
		settable.Set(sliceVal)
	}

	return nil
}

func queueOneToManyRelation(r *relation, relId string, conn redis.Conn) error {

	// see if the relation has been saved in redis
	// if not we should just leave it as nil
	key := r.redisName + ":" + relId
	// NOTE: here we're passing in nil, so the function will create
	// and close a new connection for us. This prevents ruining the transaction
	// going on in our current conn.
	exists, err := KeyExists(key, nil)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	// add the command to the queu
	if err := conn.Send("hgetall", key); err != nil {
		return err
	}

	return nil
}
