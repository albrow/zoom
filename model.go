package zoom

// File contains code strictly related to Model and ModelInterface.
// The Register() method and associated methods are also included here.

import (
	"reflect"
)

type Model struct {
	Id string `redis:"-"`
}

type ModelInterface interface {
	GetId() string
	SetId(string)
}

func (m *Model) GetId() string {
	return m.Id
}

func (m *Model) SetId(id string) {
	m.Id = id
}

// maps a type to a string identifier. The string is used
// as a key in the redis database.
var typeToName map[reflect.Type]string = make(map[reflect.Type]string)

// maps a string identifier to a type. This is so you can
// pass in a string for the *ById methods
var nameToType map[string]reflect.Type = make(map[string]reflect.Type)

// adds a model to the map of registered models
// Both name and typeOf(m) must be unique, i.e.
// not already registered
func Register(in interface{}, name string) error {
	typ := reflect.TypeOf(in)
	if alreadyRegisteredType(typ) {
		return NewTypeAlreadyRegisteredError(typ)
	}
	if typ.Kind() != reflect.Ptr {
		return NewInterfaceIsNotPointerError(in)
	}
	if alreadyRegisteredName(name) {
		return NewNameAlreadyRegisteredError(name)
	}
	typeToName[typ] = name
	nameToType[name] = typ
	return nil
}

func UnregisterName(name string) error {
	typ, ok := nameToType[name]
	if !ok {
		return NewModelNameNotRegisteredError(name)
	}
	delete(nameToType, name)
	delete(typeToName, typ)
	return nil
}

func UnregisterType(typ reflect.Type) error {
	name, ok := typeToName[typ]
	if !ok {
		return NewModelTypeNotRegisteredError(typ)
	}
	delete(nameToType, name)
	delete(typeToName, typ)
	return nil
}

// returns true iff the model name has already been registered
func alreadyRegisteredName(n string) bool {
	_, ok := nameToType[n]
	return ok
}

// returns true iff the model type has already been registered
func alreadyRegisteredType(t reflect.Type) bool {
	_, ok := typeToName[t]
	return ok
}

// get the registered name of the model we're trying to save
// based on the interfaces type. If the interface's name/type has
// not been registered, returns a ModelTypeNotRegisteredError
func getRegisteredNameFromInterface(in interface{}) (string, error) {
	typ := reflect.TypeOf(in)
	name, ok := typeToName[typ]
	if !ok {
		return "", NewModelTypeNotRegisteredError(typ)
	}
	return name, nil
}

// get the registered type of the model we're trying to save
// based on the model name. If the interface's name/type has
// not been registered, returns a ModelNameNotRegisteredError
func getRegisteredTypeFromName(name string) (reflect.Type, error) {
	typ, ok := nameToType[name]
	if !ok {
		return nil, NewModelNameNotRegisteredError(name)
	}
	return typ, nil
}
