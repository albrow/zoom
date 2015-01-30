// Copyright 2014 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File errors.go declares all the different errors that might be thrown
// by the package and provides constructors for each one.

package zoom

import (
	"fmt"
	"reflect"
)

// TODO: add more custom error types based on common use cases throughout the package

// NameAlreadyRegisteredError is returned if you try to register a
// name which has already been registered.
type NameAlreadyRegisteredError struct {
	name string
}

func (e *NameAlreadyRegisteredError) Error() string {
	return fmt.Sprintf("zoom: the name %s has already been registered", e.name)
}

func NewNameAlreadyRegisteredError(name string) *NameAlreadyRegisteredError {
	return &NameAlreadyRegisteredError{name}
}

// TypeAlreadyRegisteredError is returned if you try to register a
// type which has already been registered.
type TypeAlreadyRegisteredError struct {
	typ reflect.Type
}

func (e *TypeAlreadyRegisteredError) Error() string {
	return fmt.Sprintf("zoom: the type %s has already been registered", e.typ.String())
}

func NewTypeAlreadyRegisteredError(typ reflect.Type) *TypeAlreadyRegisteredError {
	return &TypeAlreadyRegisteredError{typ}
}

// ModelTypeNotRegisteredError is returned if you attempt to perform
// certain operations for unregistered types.
type ModelTypeNotRegisteredError struct {
	typ reflect.Type
}

func (e *ModelTypeNotRegisteredError) Error() string {
	return fmt.Sprintf("zoom: the type %s has not been registered", e.typ.String())
}

func NewModelTypeNotRegisteredError(typ reflect.Type) *ModelTypeNotRegisteredError {
	return &ModelTypeNotRegisteredError{typ}
}

// ModelNameNotRegisteredError is returned if you attempt to perform
// certain operations for unregistered names.
type ModelNameNotRegisteredError struct {
	name string
}

func (e *ModelNameNotRegisteredError) Error() string {
	return fmt.Sprintf("zoom: the model name %s has not been registered", e.name)
}

func NewModelNameNotRegisteredError(name string) *ModelNameNotRegisteredError {
	return &ModelNameNotRegisteredError{name}
}

// KeyNotFoundError is returned from Find, Scan, and Query functions if the
// model you are trying to find does not exist in the database.
type KeyNotFoundError struct {
	key       string
	modelType reflect.Type
}

func (e *KeyNotFoundError) Error() string {
	return fmt.Sprintf("zoom: could not find model of type %s with key %s", e.modelType.String(), e.key)
}

func NewKeyNotFoundError(key string, modelType reflect.Type) *KeyNotFoundError {
	return &KeyNotFoundError{key, modelType}
}

// ModelNotFoundError is returned from ScanOne and RunOne if a model that matches
// the query criteria was not found.
type ModelNotFoundError struct{}

func (e *ModelNotFoundError) Error() string {
	return "zoom: could not find a model which matches the query criteria"
}

func NewModelNotFoundError() *ModelNotFoundError {
	return &ModelNotFoundError{}
}

// DependencyCycleError is returned from addDependency when the dependency would create a cycle
type DependencyCycleError struct {
	phase *phase
	dep   *phase
}

func (e *DependencyCycleError) Error() string {
	msg := fmt.Sprintf("zoom: could not add dependency (%s depends on %s) because it creates a cycle.", e.phase.id, e.dep.id)
	msg += fmt.Sprintf("\n\t%s depends on: %v", e.phase.id, e.phase.depIds())
	msg += fmt.Sprintf("\n\t%s depends on: %v", e.dep.id, e.dep.depIds())
	return msg
}

func NewDependencyCycleError(phase, dep *phase) *DependencyCycleError {
	return &DependencyCycleError{
		phase: phase,
		dep:   dep,
	}
}
