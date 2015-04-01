// Copyright 2014 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File model.go contains code related to the Model interface.
// The Register() method and associated methods are also included here.

package zoom

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"reflect"
	"strings"
)

// DefaultData should be embedded in any struct you wish to save.
// It includes important fields and required methods to implement Model.
type DefaultData struct {
	Id string `redis:"-"`
}

// Model is an interface encapsulating anything that can be saved.
// Any struct which includes an embedded DefaultData field satisfies
// the Model interface.
type Model interface {
	GetId() string
	SetId(string)
	// TODO: add getters and setters for other default fields?
}

// GetId returns the id of the model, satisfying the Model interface
func (d DefaultData) GetId() string {
	return d.Id
}

// SetId sets the id of the model, satisfying the Model interface
func (d *DefaultData) SetId(id string) {
	d.Id = id
}

// modelSpec contains parsed information about a particular type of model
type modelSpec struct {
	typ          reflect.Type
	name         string
	fieldsByName map[string]*fieldSpec
	fields       []*fieldSpec
}

// fieldSpec contains parsed information about a particular field
type fieldSpec struct {
	kind      fieldKind
	name      string
	redisName string
	typ       reflect.Type
	indexKind indexKind
}

// fieldKind is the kind of a particular field, and is either a primative,
// a pointer, or an inconvertible.
type fieldKind int

const (
	primativeField     fieldKind = iota // any primative type
	pointerField                        // pointer to any primative type
	inconvertibleField                  // all other types
)

// indexKind is the kind of an index, and is either noIndex, numericIndex,
// stringIndex, or booleanIndex.
type indexKind int

const (
	noIndex indexKind = iota
	numericIndex
	stringIndex
	booleanIndex
)

// compilesModelSpec examines typ using reflection, parses its fields,
// and returns a modelSpec.
func compileModelSpec(typ reflect.Type) (*modelSpec, error) {
	ms := &modelSpec{fieldsByName: map[string]*fieldSpec{}, typ: typ}

	// Iterate through fields
	elem := typ.Elem()
	numFields := elem.NumField()
	for i := 0; i < numFields; i++ {
		field := elem.Field(i)
		// Skip the DefaultData field
		if field.Type == reflect.TypeOf(DefaultData{}) {
			continue
		}

		// Parse the "redis" tag
		tag := field.Tag
		redisTag := tag.Get("redis")
		if redisTag == "-" {
			continue // skip field
		}
		fs := &fieldSpec{name: field.Name, typ: field.Type}
		ms.fieldsByName[fs.name] = fs
		ms.fields = append(ms.fields, fs)
		if redisTag != "" {
			fs.redisName = redisTag
		} else {
			fs.redisName = fs.name
		}

		// Parse the "zoom" tag (currently only "index" is supported)
		zoomTag := tag.Get("zoom")
		shouldIndex := false
		if zoomTag != "" {
			options := strings.Split(zoomTag, ",")
			for _, op := range options {
				switch op {
				case "index":
					shouldIndex = true
				default:
					return nil, fmt.Errorf("zoom: unrecognized option specified in struct tag: %s", op)
				}
			}
		}

		// Detect the kind of the field and (if applicable) the kind of the index
		if typeIsPrimative(field.Type) {
			// Primative
			fs.kind = primativeField
			if shouldIndex {
				if err := setIndexKind(fs, field.Type); err != nil {
					return nil, err
				}
			}
		} else if field.Type.Kind() == reflect.Ptr && typeIsPrimative(field.Type.Elem()) {
			// Pointer to a primative
			fs.kind = pointerField
			if shouldIndex {
				if err := setIndexKind(fs, field.Type.Elem()); err != nil {
					return nil, err
				}
			}
		} else {
			// All other types are considered inconvertible
			fs.kind = inconvertibleField
		}
	}
	return ms, nil
}

// setIndexKind sets the indexKind field of fs based on fieldType
func setIndexKind(fs *fieldSpec, fieldType reflect.Type) error {
	switch {
	case typeIsNumeric(fieldType):
		fs.indexKind = numericIndex
	case typeIsString(fieldType):
		fs.indexKind = stringIndex
	case typeIsBool(fieldType):
		fs.indexKind = booleanIndex
	default:
		return fmt.Errorf("zoom: Requested index on unsupported type %s", fieldType.String())
	}
	return nil
}

// allIndexKey returns a key which is used in redis to store all the ids of every model of a
// given type
func (ms *modelSpec) allIndexKey() string {
	return ms.name + ":all"
}

// modelKey returns the key that identifies a hash in the database
// which contains all the fields of the given model. It returns an error
// iff the model does not have an id.
func (ms *modelSpec) modelKey(model Model) (string, error) {
	if model.GetId() == "" {
		return "", fmt.Errorf("zoom: Error in modelKey: model does not have an id and therefore cannot have a valid key")
	}
	return ms.name + ":" + model.GetId(), nil
}

// fieldNames returns all the field names for the given modelSpec
func (ms modelSpec) fieldNames() []string {
	names := make([]string, len(ms.fields))
	count := 0
	for _, field := range ms.fields {
		names[count] = field.name
		count++
	}
	return names
}

// fieldIndexKey returns the key for the sorted set used to index the field identified
// by fieldName. It returns an error if fieldName does not identify a field in the spec
// or if the field it identifies is not an indexed field.
func (ms *modelSpec) fieldIndexKey(fieldName string) (string, error) {
	fs, found := ms.fieldsByName[fieldName]
	if !found {
		return "", fmt.Errorf("Type %s has no field named %s", ms.typ.Name(), fieldName)
	} else if fs.indexKind == noIndex {
		return "", fmt.Errorf("%s.%s is not an indexed field", ms.typ.Name(), fieldName)
	}
	return ms.name + ":" + fs.redisName, nil
}

// modelRef represents a reference to a particular model. It consists of the model object
// itself and a pointer to the corresponding spec. This allows us to avoid constant lookups
// in the modelTypeToSpec map.
type modelRef struct {
	model Model
	spec  *modelSpec
}

// value is an alias for reflect.ValueOf(mr.model)
func (mr *modelRef) value() reflect.Value {
	return reflect.ValueOf(mr.model)
}

// elemValue dereferences the model and returns the
// underlying struct. If the model is a nil pointer,
// it will panic if the model is a nil pointer
func (mr *modelRef) elemValue() reflect.Value {
	if mr.value().IsNil() {
		msg := fmt.Sprintf("zoom: panic in elemValue(). Model of type %T was nil", mr.model)
		panic(msg)
	}
	return mr.value().Elem()
}

// fieldValue is an alias for mr.elemValue().FieldByName(name). It panics if
// the model behind mr does not have a field with the given name or if
// the model is nil.
func (mr *modelRef) fieldValue(name string) reflect.Value {
	return mr.elemValue().FieldByName(name)
}

// key returns a key which is used in redis to store the model
func (mr *modelRef) key() string {
	return mr.spec.name + ":" + mr.model.GetId()
}

// mainHashArgs returns the args for the main hash for this model. Typically
// these args should part of an HMSET command.
func (mr *modelRef) mainHashArgs() (redis.Args, error) {
	args := redis.Args{mr.key()}
	ms := mr.spec
	for _, fs := range ms.fields {
		fieldVal := mr.fieldValue(fs.name)
		switch fs.kind {
		case primativeField:
			args = args.Add(fs.redisName, fieldVal.Interface())
		case pointerField:
			if !fieldVal.IsNil() {
				args = args.Add(fs.redisName, fieldVal.Elem().Interface())
			} else {
				args = args.Add(fs.redisName, "NULL")
			}
		case inconvertibleField:
			if fieldVal.Type().Kind() == reflect.Ptr && fieldVal.IsNil() {
				args = args.Add(fs.redisName, "NULL")
			} else {
				// For inconvertibles, we convert the value to bytes using the gob package.
				valBytes, err := defaultMarshalerUnmarshaler.Marshal(fieldVal.Interface())
				if err != nil {
					return nil, err
				}
				args = args.Add(fs.redisName, valBytes)
			}
		}
	}
	return args, nil
}
