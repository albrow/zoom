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

	// set the id
	relModel.SetId(relId)

	// set settable to the value of the new struct we just created
	if r.typ == ONE_TO_ONE {
		settable.Set(relVal)
	} else if r.typ == ONE_TO_MANY {
		// for one-to-many, this means appending to the slice of related structs
		sliceVal := reflect.Append(settable, relVal)
		settable.Set(sliceVal)
	}

	return nil
}
