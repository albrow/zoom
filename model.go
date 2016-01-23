// Copyright 2015 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File model.go contains code related to the Model interface.
// The Register() method and associated methods are also included here.

package zoom

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/garyburd/redigo/redis"
)

// RandomId can be embedded in any model struct in order to satisfy
// the Model interface. The first time the ModelId method is called
// on an embedded RandomId, it will generate a pseudo-random id which
// is highly likely to be unique.
type RandomId struct {
	Id string
}

// Model is an interface encapsulating anything that can be saved and
// retrieved by Zoom. The only requirement is that a Model must have
// a getter and a setter for a unique id property.
type Model interface {
	ModelId() string
	SetModelId(string)
}

// ModelId returns the id of the model, satisfying the Model interface.
// If r.Id is an empty string, it will generate a pseudo-random id which
// is highly likely to be unique.
func (r *RandomId) ModelId() string {
	if r.Id == "" {
		r.Id = generateRandomId()
	}
	return r.Id
}

// SetModelId sets the id of the model, satisfying the Model interface
func (r *RandomId) SetModelId(id string) {
	r.Id = id
}

// modelSpec contains parsed information about a particular type of model
type modelSpec struct {
	typ          reflect.Type
	name         string
	fieldsByName map[string]*fieldSpec
	fields       []*fieldSpec
	fallback     MarshalerUnmarshaler
}

// fieldSpec contains parsed information about a particular field
type fieldSpec struct {
	kind      fieldKind
	name      string
	redisName string
	typ       reflect.Type
	indexKind indexKind
}

// fieldKind is the kind of a particular field, and is either a primitive,
// a pointer, or an inconvertible.
type fieldKind int

const (
	primativeField     fieldKind = iota // any primitive type
	pointerField                        // pointer to any primitive type
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
	ms := &modelSpec{
		name:         getDefaultModelSpecName(typ),
		fieldsByName: map[string]*fieldSpec{},
		typ:          typ,
	}

	// Iterate through fields
	elem := typ.Elem()
	numFields := elem.NumField()
	for i := 0; i < numFields; i++ {
		field := elem.Field(i)
		// Skip unexported fields. Prior to go 1.6, field.PkgPath won't give us
		// the behavior we want. Unlike packages such as encoding/json and
		// encoding/gob, Zoom does not save unexported embedded structs with
		// exported fields. So instead, we check if the first character of the
		// field name is lowercase.
		if strings.ToLower(field.Name[0:1]) == field.Name[0:1] {
			continue
		}

		// Skip the RandomId field
		if field.Type == reflect.TypeOf(RandomId{}) {
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
			// Primitive
			fs.kind = primativeField
			if shouldIndex {
				if err := setIndexKind(fs, field.Type); err != nil {
					return nil, err
				}
			}
		} else if field.Type.Kind() == reflect.Ptr && typeIsPrimative(field.Type.Elem()) {
			// Pointer to a primitive
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

// getDefaultModelSpecName returns the default name for the given type, which is
// simply the name of the type without the package prefix or dereference
// operators.
func getDefaultModelSpecName(typ reflect.Type) string {
	// Strip any dereference operators
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	nameWithPackage := typ.String()
	// Strip the package name
	return strings.Join(strings.Split(nameWithPackage, ".")[1:], "")
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
func (ms *modelSpec) indexKey() string {
	return ms.name + ":all"
}

// modelKey returns the key that identifies a hash in the database
// which contains all the fields of the model corresponding to the given
// id. It returns an error iff id is empty.
func (ms *modelSpec) modelKey(id string) (string, error) {
	if id == "" {
		return "", fmt.Errorf("zoom: Error in modelKey: id was empty")
	}
	return ms.name + ":" + id, nil
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

// fieldRedisNames returns all the redis names (which might be custom names specified via
// the `redis:"custonName"` struct tag) for each field in the given modelSpec
func (ms modelSpec) fieldRedisNames() []string {
	names := make([]string, len(ms.fields))
	count := 0
	for _, field := range ms.fields {
		names[count] = field.redisName
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

// sortArgs returns arguments that can be used to get all the fields in includeFields
// for all the models which have corresponding ids in setKey. Any fields not in
// includeFields will not be included in the arguments and will not be retrieved from
// redis when the command is eventually run. If limit or offset are not 0, the LIMIT
// option will be added to the arguments with the given limit and offset. setKey must
// be the key of a set or a sorted set which consists of model ids. The arguments
// use they "BY nosort" option, so if a specific order is required, the setKey should be
// a sorted set.
func (ms *modelSpec) sortArgs(setKey string, includeFields []string, limit int, offset uint, orderKind orderKind) redis.Args {
	args := redis.Args{setKey, "BY", "nosort"}
	for _, fieldName := range includeFields {
		args = append(args, "GET", ms.name+":*->"+fieldName)
	}
	// We always want to get the id
	args = append(args, "GET", "#")
	if !(limit == 0 && offset == 0) {
		args = append(args, "LIMIT", offset, limit)
	}
	switch orderKind {
	case ascendingOrder:
		args = append(args, "ASC")
	case descendingOrder:
		args = append(args, "DESC")
	}
	return args
}

// checkModelType returns an error iff model is not of the registered type that
// corresponds to modelSpec.
func (spec *modelSpec) checkModelType(model Model) error {
	if reflect.TypeOf(model) != spec.typ {
		return fmt.Errorf("model was the wrong type. Expected %s but got %T", spec.typ.String(), model)
	}
	return nil
}

// checkModelsType returns an error iff models is not a pointer to a slice of models of the
// registered type that corresponds to modelSpec.
func (spec *modelSpec) checkModelsType(models interface{}) error {
	if reflect.TypeOf(models).Kind() != reflect.Ptr {
		return fmt.Errorf("models should be a pointer to a slice or array of models")
	}
	modelsVal := reflect.ValueOf(models).Elem()
	elemType := modelsVal.Type().Elem()
	switch {
	case !typeIsSliceOrArray(modelsVal.Type()):
		return fmt.Errorf("models should be a pointer to a slice or array of models")
	case !typeIsPointerToStruct(elemType):
		return fmt.Errorf("the elements in models should be pointers to structs")
	case elemType != spec.typ:
		return fmt.Errorf("models were the wrong type. Expected slice or array of %s but got %T", spec.typ.String(), models)
	}
	return nil
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
	return mr.spec.name + ":" + mr.model.ModelId()
}

// mainHashArgs returns the args for the main hash for this model. Typically
// these args should part of an HMSET command.
func (mr *modelRef) mainHashArgs() (redis.Args, error) {
	return mr.mainHashArgsForFields(mr.spec.fieldNames())
}

// mainHashArgsForFields is like mainHashArgs but only returns the hash
// fields which match the given fieldNames.
func (mr *modelRef) mainHashArgsForFields(fieldNames []string) (redis.Args, error) {
	args := redis.Args{mr.key()}
	ms := mr.spec
	for _, fs := range ms.fields {
		// Skip fields whose names do not appear in fieldNames.
		if !stringSliceContains(fieldNames, fs.name) {
			continue
		}
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
			switch fieldVal.Type().Kind() {
			// For nilable types that are nil store NULL
			case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Interface:
				if fieldVal.IsNil() {
					args = args.Add(fs.redisName, "NULL")
					continue
				}
			}
			// For inconvertibles, that are not nil, convert the value to bytes
			// using the gob package.
			valBytes, err := mr.spec.fallback.Marshal(fieldVal.Interface())
			if err != nil {
				return nil, err
			}
			args = args.Add(fs.redisName, valBytes)
		}
	}
	return args, nil
}
