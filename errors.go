// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File errors.go declares all the different errors that might be thrown
// by the package and provides constructors for each one.

package zoom

import (
	"reflect"
)

// TODO: add more custom error types based on common use cases throughout the package

// NameAlreadyRegistered
type NameAlreadyRegisteredError struct {
	name string
}

func (e *NameAlreadyRegisteredError) Error() string {
	return "zoom: the name '" + e.name + "' has already been registered"
}

func NewNameAlreadyRegisteredError(name string) *NameAlreadyRegisteredError {
	return &NameAlreadyRegisteredError{name}
}

// TypeAlreadyRegistered
type TypeAlreadyRegisteredError struct {
	typ reflect.Type
}

func (e *TypeAlreadyRegisteredError) Error() string {
	return "zoom: the type '" + e.typ.String() + "' has already been registered"
}

func NewTypeAlreadyRegisteredError(typ reflect.Type) *TypeAlreadyRegisteredError {
	return &TypeAlreadyRegisteredError{typ}
}

// ModelTypeNotRegistered
type ModelTypeNotRegisteredError struct {
	typ reflect.Type
}

func (e *ModelTypeNotRegisteredError) Error() string {
	return "zoom: the type '" + e.typ.String() + "' has not been registered"
}

func NewModelTypeNotRegisteredError(typ reflect.Type) *ModelTypeNotRegisteredError {
	return &ModelTypeNotRegisteredError{typ}
}

// ModelNameNotRegistered
type ModelNameNotRegisteredError struct {
	name string
}

func (e *ModelNameNotRegisteredError) Error() string {
	return "zoom: the model name '" + e.name + "' has not been registered"
}

func NewModelNameNotRegisteredError(name string) *ModelNameNotRegisteredError {
	return &ModelNameNotRegisteredError{name}
}
