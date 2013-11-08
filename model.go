// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File model.go contains code related to DefaultData and Model.
// The Register() method and associated methods are also included here.

package zoom

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// DefaultData should be embedded in any struct you wish to save.
// It includes important fields and required methods to implement Model.
type DefaultData struct {
	Id string `redis:"-"`
	// TODO: add other default fields?
}

// Model is an interface encapsulating anything that can be saved.
// Any struct which includes an embedded DefaultData field satisfies
// the Model interface.
type Model interface {
	getId() string
	setId(string)
	// TODO: add getters and setters for other default fields?
}

type modelSpec struct {
	modelType      reflect.Type
	modelName      string
	primatives     map[string]primative     // primative types: int, float, string, etc.
	pointers       map[string]pointer       // pointers to primative tyeps: *int, *float, *string, etc.
	inconvertibles map[string]inconvertible // types which cannot be directly converted. fallback to json/msgpack
	sets           map[string]externalSet   // separate set entities
	lists          map[string]externalList  // separate list entities
	relationships  map[string]relationship  // pointers to structs of registered types
	// TODO add external hashes
}

type primative struct {
	redisName string
	fieldName string
	fieldType reflect.Type
}

type pointer struct {
	redisName string
	fieldName string
	fieldType reflect.Type
	elemType  reflect.Type
}

type inconvertible struct {
	redisName string
	fieldName string
	fieldType reflect.Type
}

type externalSet struct {
	redisName string
	fieldName string
	fieldType reflect.Type
	elemType  reflect.Type
}

type externalList struct {
	redisName string
	fieldName string
	fieldType reflect.Type
	elemType  reflect.Type
}

type relationship struct {
	redisName string
	fieldName string
	fieldType reflect.Type
	elemType  reflect.Type
	rType     relationType
}

type relationType int

const (
	oneToOne = iota
	oneToMany
)

type modelRef struct {
	model     Model
	modelSpec modelSpec
}

// maps a type to a string identifier. The string is used
// as a key in the redis database.
var modelTypeToName map[reflect.Type]string = make(map[reflect.Type]string)

// maps a string identifier to a type. This is so you can
// pass in a string for the *ById methods
var modelNameToType map[string]reflect.Type = make(map[string]reflect.Type)

// maps a string identifier to a modelSpec
var modelSpecs map[string]modelSpec = make(map[string]modelSpec)

func newModelSpec(name string, typ reflect.Type) modelSpec {
	return modelSpec{
		modelType:      typ,
		modelName:      name,
		primatives:     make(map[string]primative),
		pointers:       make(map[string]pointer),
		inconvertibles: make(map[string]inconvertible),
		sets:           make(map[string]externalSet),
		lists:          make(map[string]externalList),
		relationships:  make(map[string]relationship),
	}
}

func newModelRefFromInterface(m Model) (modelRef, error) {
	mr := modelRef{
		model: m,
	}
	modelName, err := getRegisteredNameFromInterface(m)
	if err != nil {
		return mr, err
	}
	mr.modelSpec = modelSpecs[modelName]
	return mr, nil
}

func newModelRefFromName(modelName string) (modelRef, error) {
	mr := modelRef{
		modelSpec: modelSpecs[modelName],
	}
	// create a new struct of the proper type
	val := reflect.New(mr.modelSpec.modelType.Elem())
	m, ok := val.Interface().(Model)
	if !ok {
		msg := fmt.Sprintf("zoom: could not convert val of type %T to Model", val.Interface())
		return mr, errors.New(msg)
	}
	mr.model = m
	return mr, nil
}

func (d DefaultData) getId() string {
	return d.Id
}

func (d *DefaultData) setId(id string) {
	d.Id = id
}

// Register adds a model type to the list of registered types. Any struct
// you wish to save must be registered first. The type of model must be
// unique, i.e. not already registered. Each registered model gets a name,
// a unique string identifier, which by default is just the string version
// of the type (the asterisk and any package prefixes are stripped). See
// RegisterName if you would prefer to specify a custom name.
func Register(model Model) error {
	typeName := reflect.TypeOf(model).Elem().String()
	// strip package name
	modelName := reverseString(strings.Split(reverseString(typeName), ".")[0])
	return RegisterName(modelName, model)
}

// RegisterName is like Register but allows you to specify a custom
// name to use for the model type. The custom name will be used as
// a prefix for all models of this type stored in redis. The custom
// name will also be used in functions which require a model name,
// such as queries.
func RegisterName(name string, model Model) error {
	// make sure the name and type have not been previously registered
	typ := reflect.TypeOf(model)
	if modelTypeIsRegistered(typ) {
		return NewTypeAlreadyRegisteredError(typ)
	} else if modelNameIsRegistered(name) {
		return NewNameAlreadyRegisteredError(name)
	} else if !typeIsPointerToStruct(typ) {
		msg := fmt.Sprintf("zoom: Register and RegisterName require a pointer to a struct as an argument.\nThe type %T is not a pointer to a struct.", model)
		return errors.New(msg)
	}

	modelTypeToName[typ] = name
	modelNameToType[name] = typ
	if err := compileModelSpecs(); err != nil {
		return err
	}

	return nil
}

func compileModelSpecs() error {
	for name, typ := range modelNameToType {
		ms := newModelSpec(name, typ)
		if err := compileModelSpec(typ, &ms); err != nil {
			return err
		}
		modelSpecs[name] = ms
	}
	return nil
}

// TODO: take into account embedded structs
func compileModelSpec(typ reflect.Type, ms *modelSpec) error {

	// iterate through fields
	elem := typ.Elem()
	numFields := elem.NumField()
	for i := 0; i < numFields; i++ {
		field := elem.Field(i)
		if field.Name == "DefaultData" {
			continue // skip default data
		}
		// get the redisName
		tag := field.Tag
		redisName := tag.Get("redis")
		if redisName == "-" {
			continue // skip field
		} else if redisName == "" {
			redisName = field.Name
		}
		if typeIsPrimative(field.Type) {
			// primative
			p := primative{
				redisName: redisName,
				fieldName: field.Name,
				fieldType: field.Type,
			}
			ms.primatives[field.Name] = p
		} else if field.Type.Kind() == reflect.Ptr {
			if typeIsPrimative(field.Type.Elem()) {
				// pointer to a primative
				p := pointer{
					redisName: redisName,
					fieldName: field.Name,
					fieldType: field.Type,
					elemType:  field.Type.Elem(),
				}
				ms.pointers[field.Name] = p
			} else if typeIsPointerToStruct(field.Type) {
				if modelTypeIsRegistered(field.Type) {
					// one-to-one relationship
					ms.relationships[field.Name] = relationship{
						redisName: redisName,
						fieldName: field.Name,
						fieldType: field.Type,
						elemType:  field.Type.Elem(),
						rType:     oneToOne,
					}
				} else {
					// a pointer to a struct of unregistered type is incovertable
					ms.addInconvertible(field, redisName)
				}
			}
		} else if typeIsSliceOrArray(field.Type) {
			if typeIsPointerToStruct(field.Type.Elem()) {
				if modelTypeIsRegistered(field.Type.Elem()) {
					// one-to-many relationship
					ms.relationships[field.Name] = relationship{
						redisName: redisName,
						fieldName: field.Name,
						fieldType: field.Type,
						elemType:  field.Type.Elem().Elem(),
						rType:     oneToMany,
					}
				} else {
					// a slice or array of pointers to structs of unregistered type is incovertable
					ms.addInconvertible(field, redisName)
				}
			} else {
				redisType := tag.Get("redisType")
				if redisType == "list" {
					l := externalList{
						fieldName: field.Name,
						redisName: redisName,
						fieldType: field.Type,
						elemType:  field.Type.Elem(),
					}
					ms.lists[field.Name] = l
				} else if redisType == "set" {
					s := externalSet{
						fieldName: field.Name,
						redisName: redisName,
						fieldType: field.Type,
						elemType:  field.Type.Elem(),
					}
					ms.sets[field.Name] = s
				} else if redisType == "" {
					// if application did not specify it wanted an external list or external set, treat
					// the array or slice as inconvertible. Later it will be encoded to a string format
					// and written directly into the redis hash.
					ms.addInconvertible(field, redisName)
				} else {
					msg := fmt.Sprintf("zoom: redisType tag for type %s was invalid.\nShould be either 'list' or 'set'.\nGot: %s", typ.String(), redisType)
					return errors.New(msg)
				}
			}
		} else {
			// if we've reached here, the field type is inconvertible
			ms.addInconvertible(field, redisName)
		}
	}

	return nil
}

func (ms *modelSpec) addInconvertible(field reflect.StructField, redisName string) {
	ms.inconvertibles[field.Name] = inconvertible{
		fieldName: field.Name,
		redisName: redisName,
		fieldType: field.Type,
	}
}

// Unregister removes a model type from the list of registered types.
// You only need to call UnregisterName or UnregisterType, not both.
func Unregister(model Model) error {
	modelType := reflect.TypeOf(model)
	name, ok := modelTypeToName[modelType]
	if !ok {
		return NewModelTypeNotRegisteredError(modelType)
	}
	delete(modelNameToType, name)
	delete(modelTypeToName, modelType)
	return nil
}

// UnregisterName removes a model type (identified by modelName) from the list of
// registered model types. You only need to call UnregisterName or UnregisterType,
// not both.
func UnregisterName(name string) error {
	typ, ok := modelNameToType[name]
	if !ok {
		return NewModelNameNotRegisteredError(name)
	}
	delete(modelNameToType, name)
	delete(modelTypeToName, typ)
	return nil
}

// modelNameIsRegistered returns true iff the model name has already been registered
func modelNameIsRegistered(n string) bool {
	_, ok := modelNameToType[n]
	return ok
}

// modelTypeIsRegistered returns true iff the model type has already been registered
func modelTypeIsRegistered(t reflect.Type) bool {
	_, ok := modelTypeToName[t]
	return ok
}

// getRegisteredNameFromType gets the registered name of the model we're
// trying to save based on the type. If the interface's name/type
// has not been registered, returns a ModelTypeNotRegisteredError
func getRegisteredNameFromType(typ reflect.Type) (string, error) {
	name, ok := modelTypeToName[typ]
	if !ok {
		return "", NewModelTypeNotRegisteredError(typ)
	}
	return name, nil
}

// getRegisteredNameFromInterface gets the registered name of the model we're
// trying to save based on the interfaces type. If the interface's name/type
// has not been registered, returns a ModelTypeNotRegisteredError
func getRegisteredNameFromInterface(in interface{}) (string, error) {
	return getRegisteredNameFromType(reflect.TypeOf(in))
}

// getRegisteredTypeFromName gets the registered type of the model we're trying
// to save based on the model name. If the interface's name/type has not been registered,
// returns a ModelNameNotRegisteredError
func getRegisteredTypeFromName(name string) (reflect.Type, error) {
	typ, ok := modelNameToType[name]
	if !ok {
		return nil, NewModelNameNotRegisteredError(name)
	}
	return typ, nil
}

func (mr modelRef) elemVal() reflect.Value {
	return reflect.ValueOf(mr.model).Elem()
}

func (mr modelRef) modelVal() reflect.Value {
	return reflect.ValueOf(mr.model)
}

func (mr modelRef) value(fieldName string) reflect.Value {
	return mr.elemVal().FieldByName(fieldName)
}

func (mr modelRef) key() string {
	return mr.modelSpec.modelName + ":" + mr.model.getId()
}

func (mr modelRef) indexKey() string {
	return mr.modelSpec.modelName + ":all"
}

func (ms modelSpec) field(fieldName string) (reflect.StructField, bool) {
	return ms.modelType.Elem().FieldByName(fieldName)
}

func (ms modelSpec) indexKey() string {
	return ms.modelName + ":all"
}

// returns the args that should be sent to the redis driver
// and used in a HMSET command
func (mr modelRef) mainHashArgs() ([]interface{}, error) {
	args := []interface{}{mr.key()}
	ms := mr.modelSpec
	for _, p := range ms.primatives {
		args = append(args, p.redisName, mr.value(p.fieldName).Interface())
	}
	for _, p := range ms.pointers {
		args = append(args, p.redisName, mr.value(p.fieldName).Elem().Interface())
	}
	for _, inc := range ms.inconvertibles {
		// TODO: account for the possibility of json, msgpack or custom fallbacks
		valBytes, err := defaultMarshalerUnmarshaler.Marshal(mr.value(inc.fieldName).Interface())
		if err != nil {
			return args, err
		}
		args = append(args, inc.redisName, valBytes)
	}
	return args, nil
}
