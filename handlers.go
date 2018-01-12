package zoom

import (
	"fmt"
	"reflect"

	"github.com/garyburd/redigo/redis"
)

// ReplyHandler is a function which does something with the reply from a Redis
// command or script. See https://godoc.org/github.com/garyburd/redigo/redis
// for a description of the concrete types for reply.
type ReplyHandler func(reply interface{}) error

// newAlwaysErrorHandler returns a ReplyHandler that ignores the reply and
// always returns the given error.
func newAlwaysErrorHandler(err error) ReplyHandler {
	return func(interface{}) error {
		return err
	}
}

// newModelExistsHandler returns a reply handler which will return a
// ModelNotFound error if the value of reply is false. It is expected to be
// used as the reply handler for an EXISTS command.
func newModelExistsHandler(collection *Collection, modelID string) ReplyHandler {
	return func(reply interface{}) error {
		exists, err := redis.Bool(reply, nil)
		if err != nil {
			return err
		}
		if !exists {
			return ModelNotFoundError{
				Collection: collection,
				Msg:        fmt.Sprintf("Could not find %s with id = %s", collection.spec.name, modelID),
			}
		}
		return nil
	}
}

// NewScanIntHandler returns a ReplyHandler which will convert the reply to an
// integer and set the value of i to the converted integer. The ReplyHandler
// will return an error if there was a problem converting the reply.
func NewScanIntHandler(i *int) ReplyHandler {
	return func(reply interface{}) error {
		var err error
		(*i), err = redis.Int(reply, nil)
		if err != nil {
			return err
		}
		return nil
	}
}

// NewScanBoolHandler returns a ReplyHandler which will convert the reply to a
// bool and set the value of i to the converted bool. The ReplyHandler
// will return an error if there was a problem converting the reply.
func NewScanBoolHandler(b *bool) ReplyHandler {
	return func(reply interface{}) error {
		var err error
		(*b), err = redis.Bool(reply, nil)
		if err != nil {
			return err
		}
		return nil
	}
}

// NewScanStringHandler returns a ReplyHandler which will convert the reply to
// a string and set the value of i to the converted string. The ReplyHandler
// will return an error if there was a problem converting the reply.
func NewScanStringHandler(s *string) ReplyHandler {
	return func(reply interface{}) error {
		var err error
		(*s), err = redis.String(reply, nil)
		if err != nil {
			return err
		}
		return nil
	}
}

// NewScanFloat64Handler returns a ReplyHandler which will convert the reply to a
// float64 and set the value of f to the converted value. The ReplyHandler
// will return an error if there was a problem converting the reply.
func NewScanFloat64Handler(f *float64) ReplyHandler {
	return func(reply interface{}) error {
		var err error
		(*f), err = redis.Float64(reply, nil)
		if err != nil {
			return err
		}
		return nil
	}
}

// NewScanStringsHandler returns a ReplyHandler which will convert the reply to
// a slice of strings and set the value of strings to the converted value. The
// returned ReplyHandler will grow or shrink strings as needed. The ReplyHandler
// will return an error if there was a problem converting the reply.
func NewScanStringsHandler(strings *[]string) ReplyHandler {
	return func(reply interface{}) error {
		var err error
		(*strings), err = redis.Strings(reply, nil)
		if err != nil {
			return err
		}
		return nil
	}
}

// newScanModelRefHandler works exactly like the exported NewScanModelHandler,
// but it expects a *modelRef as the final argument instead of a Model. See
// the documentation for NewScanModelHandler for more information.
func newScanModelRefHandler(fieldNames []string, mr *modelRef) ReplyHandler {
	return func(reply interface{}) error {
		fieldValues, err := redis.Values(reply, nil)
		if err != nil {
			if err == redis.ErrNil {
				return newModelNotFoundError(mr)
			}
			return err
		}
		if err := scanModel(fieldNames, fieldValues, mr); err != nil {
			return err
		}
		return nil
	}
}

// NewScanModelHandler returns a ReplyHandler which will scan all the values in
// the reply into the fields of model. It expects a reply that looks like the
// output of an HMGET command, without the field names included. The order of
// fieldNames must correspond to the order of the values in the reply.
//
// fieldNames should be the actual field names as they appear in the struct
// definition, not the Redis names which may be custom. The special field name
// "-" is used to represent the id for the model and will be set by the
// ReplyHandler using the SetModelID method.
//
// For example, if fieldNames is ["Age", "Name", "-"] and the reply from Redis
// looks like this:
//
//   1) "25"
//   2) "Bob"
//   3) "b1C7B0yETtXFYuKinndqoa"
//
// The ReplyHandler will set the Age and the Name of the model to 25 and "Bob",
// respectively, using reflection. Then it will set the id of the model to
// "b1C7B0yETtXFYuKinndqoa" using the model's SetModelID method.
func NewScanModelHandler(fieldNames []string, model Model) ReplyHandler {
	// Create a modelRef that wraps the given model.
	collection, err := getCollectionForModel(model)
	if err != nil {
		return newAlwaysErrorHandler(err)
	}
	mr := &modelRef{
		collection: collection,
		model:      model,
		spec:       collection.spec,
	}
	// Create and return a reply handler using newScanModelRefHandler
	return newScanModelRefHandler(fieldNames, mr)
}

// newScanModelsHandler operates exactly like the exported NewScanModelsHandler,
// but expects a *modelSpec as the first argument instead of a *Collection. See
// the documentation for NewScanModelsHandler for more information.
func newScanModelsHandler(spec *modelSpec, fieldNames []string, models interface{}) ReplyHandler {
	return func(reply interface{}) error {
		allFields, err := redis.Values(reply, nil)
		modelsVal := reflect.ValueOf(models).Elem()
		if err != nil {
			if err == redis.ErrNil {
				// This means no models matched the criteria. Set the length of
				// models to 0 to indicate this and then return.
				modelsVal.SetLen(0)
				return nil
			}
			return err
		}
		numFields := len(fieldNames)
		numModels := len(allFields) / numFields
		for i := 0; i < numModels; i++ {
			start := i * numFields
			stop := i*numFields + numFields
			fieldValues := allFields[start:stop]
			var modelVal reflect.Value
			if modelsVal.Len() > i {
				// Use the pre-existing value at index i
				modelVal = modelsVal.Index(i)
				if modelVal.IsNil() {
					// If the value is nil, allocate space for it
					modelsVal.Index(i).Set(reflect.New(spec.typ.Elem()))
				}
			} else {
				// Index i is out of range of the existing slice. Create a
				// new modelVal and append it to modelsVal
				modelVal = reflect.New(spec.typ.Elem())
				modelsVal.Set(reflect.Append(modelsVal, modelVal))
			}
			mr := &modelRef{
				spec:  spec,
				model: modelVal.Interface().(Model),
			}
			if err := scanModel(fieldNames, fieldValues, mr); err != nil {
				return err
			}
		}
		// Trim the slice if it is longer than the number of models we scanned
		// in.
		if numModels < modelsVal.Len() {
			modelsVal.SetLen(numModels)
			modelsVal.SetCap(numModels)
		}
		return nil
	}
}

// NewScanModelsHandler returns a ReplyHandler which will scan the values of the
// reply into each corresponding Model in models. models should be a pointer to
// a slice of some concrete Model type. The type of the Models in models should
// match the type of the given Collection.
//
// fieldNames should be the actual field names as they appear in the struct
// definition, not the Redis names which may be custom. The special value of "-"
// in fieldNames represents the id of the model and will be set via the
// SetModelID method. The ReplyHandler will use the length of fieldNames to
// determine which fields belong to which models.
//
// The returned ReplyHandler will grow or shrink models as needed. It expects a
// reply which is a flat array of field values, with no separation between the
// fields for each model. The order of the values in the reply must correspond
// to the order of fieldNames.
//
// For example, if fieldNames is ["Age", "Name", "-"] and the reply from Redis
// looks like this:
//
//   1) "25"
//   2) "Bob"
//   3) "b1C7B0yETtXFYuKinndqoa"
//   4) "27"
//   5) "Alice"
//   6) "NmjzzzDyNJpsCpPKnndqoa"
//
// NewScanModelsHandler will use the first value in the reply to set the Age
// field of the first Model to 25 using reflection. It will use the second value
// of the reply to set Name value of the first Model to "Bob" using reflection.
// And finally, it will use the third value in reply to set the id of the first
// Model by calling its SetModelID method. Because the length of fieldNames is 3
// in this case, the ReplyHandler will assign the first three to the first
// model, the next three to the second model, etc.
func NewScanModelsHandler(collection *Collection, fieldNames []string, models interface{}) ReplyHandler {
	return newScanModelsHandler(collection.spec, fieldNames, models)
}

// newScanOneModelHandler returns a ReplyHandler which will scan reply into the
// given model. It differs from NewScanModelHandler in that it expects reply to
// have an underlying type of [][]byte{}. Specifically, if fieldNames is
// ["Age", "Name", "-"], reply should look like:
//
//   1) "25"
//   2) "Bob"
//   3) "b1C7B0yETtXFYuKinndqoa"
//
// Note that this is similar to the kind of reply expected by
// NewScanModelHandler except that there should only ever be len(fieldNames)
// fields in the reply (i.e. enough fields for exactly one model). If the reply
// is nil or an empty array, the ReplyHandler will return an error. This makes
// newScanOneModelHandler useful in contexts where you expect exactly one model
// to match certain query criteria (e.g. for Query.RunOne).
func newScanOneModelHandler(q *query, spec *modelSpec, fieldNames []string, model Model) ReplyHandler {
	return func(reply interface{}) error {
		// Use reflection to create a slice which contains only one element, the
		// given model. We'll then pass this in to newScanModelsHandler to set the
		// value of model.
		modelsVal := reflect.New(reflect.SliceOf(reflect.TypeOf(model)))
		modelsVal.Elem().Set(reflect.Append(modelsVal.Elem(), reflect.ValueOf(model)))
		if err := newScanModelsHandler(spec, fieldNames, modelsVal.Interface())(reply); err != nil {
			return err
		}
		// Return an error if we didn't find any models matching the criteria.
		// When you use newScanOneModelHandler, you are explicitly saying that you
		// expect exactly one model.
		if modelsVal.Elem().Len() == 0 {
			msg := fmt.Sprintf("Could not find a model with the given query criteria: %s", q)
			return ModelNotFoundError{Msg: msg}
		}
		return nil
	}
}
