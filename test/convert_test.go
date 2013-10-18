// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File convert_test.go is intended to test the conversion
// to and from go data structures of a variety of types.

package test

import (
	"github.com/stephenalexbrowne/zoom"
	"github.com/stephenalexbrowne/zoom/test_support"
	"github.com/stephenalexbrowne/zoom/util"
	"reflect"
	"testing"
)

func TestPrimativeTypes(t *testing.T) {
	construct := func() (interface{}, error) {
		pts, err := test_support.NewPrimativeTypes(1)
		return pts[0], err
	}
	testConvertType(reflect.TypeOf(test_support.PrimativeTypes{}), construct, t)
}

func TestPointerToPrimativeTypes(t *testing.T) {
	construct := func() (interface{}, error) {
		pts, err := test_support.NewPointerPrimativeTypes(1)
		return pts[0], err
	}
	testConvertType(reflect.TypeOf(test_support.PointerPrimativeTypes{}), construct, t)
}

func TestInconvertibleTypes(t *testing.T) {
	construct := func() (interface{}, error) {
		pts, err := test_support.NewInconvertibleTypes(1)
		return pts[0], err
	}
	testConvertType(reflect.TypeOf(test_support.InconvertibleTypes{}), construct, t)
}

func TestEmbeddedStruct(t *testing.T) {
	construct := func() (interface{}, error) {
		pts, err := test_support.NewEmbeddedStructs(1)
		return pts[0], err
	}
	testConvertType(reflect.TypeOf(test_support.EmbeddedStruct{}), construct, t)
}

func TestPointerEmbeddedStruct(t *testing.T) {
	construct := func() (interface{}, error) {
		pts, err := test_support.NewPrimativeTypes(1)
		return pts[0], err
	}
	testConvertType(reflect.TypeOf(test_support.PrimativeTypes{}), construct, t)
}

// a general test that uses reflection
func testConvertType(typ reflect.Type, construct func() (in interface{}, err error), t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// construct a model using the construct function
	modelInterface, err := construct()
	if err != nil {
		t.Error(err)
	}
	model, ok := modelInterface.(zoom.Model)
	if !ok {
		t.Errorf("couldn't convert type %T to zoom.Model", modelInterface)
	}
	if err := zoom.Save(model); err != nil {
		t.Error(err)
	}

	// get the id of the model using reflection
	id := reflect.ValueOf(model).Elem().FieldByName("Id").String()

	// create a copy of the same type and use zoom.ScanById
	modelCopyInterface := reflect.New(typ).Interface()
	modelCopy, ok := modelCopyInterface.(zoom.Model)
	if !ok {
		t.Errorf("couldn't convert type %T to zoom.Model", modelCopyInterface)
	}
	if _, err := zoom.ScanById(id, modelCopy).Run(); err != nil {
		t.Error(err)
	}

	// make sure the copy equals the original
	equal, err := util.Equals(model, modelCopy)
	if err != nil {
		t.Error(err)
	}
	if !equal {
		t.Errorf("model was not saved/retrieved correctly.\nExpected: %+v\nGot %+v\n", model, modelCopy)
	}
}
