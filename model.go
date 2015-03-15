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

func (d DefaultData) GetId() string {
	return d.Id
}

func (d *DefaultData) SetId(id string) {
	d.Id = id
}

var (
	// modelTypeToSpec maps a registered model type to a modelSpec
	modelTypeToSpec map[reflect.Type]*modelSpec = make(map[reflect.Type]*modelSpec)
	// modelNameToSpec maps a registered model name to a modelSpec
	modelNameToSpec map[string]*modelSpec = make(map[string]*modelSpec)
)

type modelSpec struct {
	typ    reflect.Type
	name   string
	fields map[string]*fieldSpec
}

type fieldSpec struct {
	kind      fieldKind
	name      string
	redisName string
	fieldType reflect.Type
	indexKind indexKind
}

type fieldKind int

const (
	primativeField fieldKind = iota
	pointerField
	inconvertibleField
)

type indexKind int

const (
	noIndex indexKind = iota
	numericIndex
	stringIndex
	booleanIndex
)

func compileModelSpec(typ reflect.Type) (*modelSpec, error) {
	ms := &modelSpec{}

	// Iterate through fields
	elem := typ.Elem()
	numFields := elem.NumField()
	for i := 0; i < numFields; i++ {
		field := elem.Field(i)
		fs := &fieldSpec{name: field.Name, fieldType: field.Type}
		ms.fields[fs.name] = fs

		// Parse the "redis" tag
		tag := field.Tag
		redisTag := tag.Get("redis")
		if redisTag == "-" {
			continue // skip field
		}
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

func (ms modelSpec) field(fieldName string) (reflect.StructField, bool) {
	return ms.typ.Elem().FieldByName(fieldName)
}

// indexKey returns a key which is used in redis to store all the ids of every model of a
// given type.
func (ms modelSpec) indexKey() string {
	return ms.name + ":all"
}

func (ms modelSpec) fieldNames() []string {
	names := make([]string, len(ms.fields))
	count := 0
	for _, field := range ms.fields {
		names[count] = field.name
		count++
	}
	return names
}

type modelRef struct {
	model     Model
	modelSpec *modelSpec
}

func newModelRef(m Model) (*modelRef, error) {
	mr := &modelRef{
		model: m,
	}
	typ := reflect.TypeOf(m)
	spec, found := modelTypeToSpec[typ]
	if !found {
		return nil, NewModelTypeNotRegisteredError(typ)
	}
	mr.modelSpec = spec
	return mr, nil
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
	return mr.modelSpec.name + ":" + mr.model.GetId()
}

// mainHashArgs returns the args for the main hash for this model. Typically
// these args should part of an HMSET command.
func (mr *modelRef) mainHashArgs() (redis.Args, error) {
	args := redis.Args{mr.key()}
	ms := mr.modelSpec
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
