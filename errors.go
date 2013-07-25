package zoom

// File declares all the different errors that might be thrown
// by the package and provides constructors for each one.

import (
	"fmt"
	"reflect"
)

// ---
// NameAlreadyRegistered
type NameAlreadyRegisteredError struct {
	name string
}

func (e *NameAlreadyRegisteredError) Error() string {
	return "The name '" + e.name + "' has already been registered."
}

func NewNameAlreadyRegisteredError(name string) *NameAlreadyRegisteredError {
	return &NameAlreadyRegisteredError{name}
}

// ---
// TypeAlreadyRegistered
type TypeAlreadyRegisteredError struct {
	typ reflect.Type
}

func (e *TypeAlreadyRegisteredError) Error() string {
	return "The type '" + e.typ.String() + "' has already been registered."
}

func NewTypeAlreadyRegisteredError(typ reflect.Type) *TypeAlreadyRegisteredError {
	return &TypeAlreadyRegisteredError{typ}
}

// ---
// ModelTypeNotRegistered
type ModelTypeNotRegisteredError struct {
	typ reflect.Type
}

func (e *ModelTypeNotRegisteredError) Error() string {
	return "The type '" + e.typ.String() + "' has not been registered."
}

func NewModelTypeNotRegisteredError(typ reflect.Type) *ModelTypeNotRegisteredError {
	return &ModelTypeNotRegisteredError{typ}
}

// ---
// ModelNameNotRegistered
type ModelNameNotRegisteredError struct {
	name string
}

func (e *ModelNameNotRegisteredError) Error() string {
	return "The model name '" + e.name + "' has not been registered."
}

func NewModelNameNotRegisteredError(name string) *ModelNameNotRegisteredError {
	return &ModelNameNotRegisteredError{name}
}

// ---
// KeyNotFound
type KeyNotFoundError struct {
	msg string
}

func (e *KeyNotFoundError) Error() string {
	return e.msg
}

func NewKeyNotFoundError(msg string) *KeyNotFoundError {
	e := KeyNotFoundError{msg}
	return &e
}

// ---
// MissingParamater
type MissingParamaterError struct {
	msg string
}

func (e *MissingParamaterError) Error() string {
	return e.msg
}

func NewMissingParamaterError(msg string) *MissingParamaterError {
	e := MissingParamaterError{msg}
	return &e
}

// ---
// InterfaceIsNotPointer
type InterfaceIsNotPointerError struct {
	in interface{}
}

func (e *InterfaceIsNotPointerError) Error() string {
	return fmt.Sprintf("Interface of type %T is not a pointer. Try again with the 'address of' operator?", e.in)
}

func NewInterfaceIsNotPointerError(in interface{}) *InterfaceIsNotPointerError {
	e := InterfaceIsNotPointerError{in}
	return &e
}

// ---
// RelationNotFound
type RelationNotFoundError struct {
	name string
}

func (e *RelationNotFoundError) Error() string {
	return "The relation by the name '" + e.name + "' does not exist."
}

func NewRelationNotFoundError(name string) *RelationNotFoundError {
	return &RelationNotFoundError{name}
}
