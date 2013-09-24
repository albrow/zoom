// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

package test

import (
	"github.com/stephenalexbrowne/zoom"
	"testing"
)

func TestRegisteredModelMustBePointer(t *testing.T) {
	err := zoom.Register("person", "invalid")
	if err == nil {
		t.Error("Registering with an invalid schema should throw an error")
	}
}

func TestRegisteredModelMustBePointerToStruct(t *testing.T) {
	str := "invalid"
	err := zoom.Register(&str, "invalid")
	if err == nil {
		t.Error("Registering with an invalid schema should throw an error")
	}
}
