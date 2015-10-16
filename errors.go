// Copyright 2015 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File errors.go declares all the different errors that might be thrown
// by the package and provides constructors for each one.

package zoom

import "fmt"

// ModelNotFoundError is returned from Find and Query methods if a model
// that fits the given criteria is not found.
type ModelNotFoundError struct {
	Msg string
}

func (e ModelNotFoundError) Error() string {
	return "zoom: ModelNotFoundError: " + e.Msg
}

func newModelNotFoundError(mr *modelRef) error {
	var msg string
	if mr.model.ModelId() != "" {
		msg = fmt.Sprintf("Could not find %s with id = %s", mr.spec.name, mr.model.ModelId())
	} else {
		msg = fmt.Sprintf("Could not find %s with the given criteria", mr.spec.name)
	}
	return ModelNotFoundError{Msg: msg}
}
