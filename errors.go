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

// NameAlreadyRegisteredError is returned if you try to register a
// name which has already been registered.
type NameAlreadyRegisteredError struct {
	Name string
}

func (e NameAlreadyRegisteredError) Error() string {
	return fmt.Sprintf("zoom: NameAlreadyRegisteredError: the name %s has already been registered", e.Name)
}

// TypeAlreadyRegisteredError is returned if you try to register a
// type which has already been registered.
type TypeAlreadyRegisteredError struct {
	Typ reflect.Type
}

func (e TypeAlreadyRegisteredError) Error() string {
	return fmt.Sprintf("zoom: TypeAlreadyRegisteredError: the type %s has already been registered", e.Typ.String())
}

// ModelNotFoundError is returned from Find and Query methods if a model
// that fits the given criteria is not found.
type ModelNotFoundError struct {
	Msg string
}

func (e ModelNotFoundError) Error() string {
	return "zoom: ModelNotFoundError: %s" + e.Msg
}
