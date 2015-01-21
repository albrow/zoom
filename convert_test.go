// Copyright 2014 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File convert_test.go tests the conversion
// to and from go data structures of a variety of types.

package zoom

import (
	"reflect"
	"testing"
)

func TestPrimativeTypes(t *testing.T) {
	construct := func() (interface{}, error) {
		ms, err := newPrimativeTypesModels(1)
		return ms[0], err
	}
	testConvertType(reflect.TypeOf(primativeTypesModel{}), construct, t)
}

func TestPointerToPrimativeTypes(t *testing.T) {
	construct := func() (interface{}, error) {
		ms, err := newPointersToPrimativeTypesModels(1)
		return ms[0], err
	}
	testConvertType(reflect.TypeOf(pointersToPrimativeTypesModel{}), construct, t)
}

func TestInconvertibleTypes(t *testing.T) {
	// we'll have to do this test manually, because looseEquals doesn't
	// support some of the types here.
	// make a model with all nil fields

	testingSetUp()
	defer testingTearDown()

	ms, err := newInconvertibleTypesModels(1)
	if err != nil {
		t.Error(err)
	}
	m := ms[0]
	if err := Save(m); err != nil {
		t.Error(err)
	}
	mCopy := new(inconvertibleTypesModel)
	if err := ScanById(m.Id, mCopy); err != nil {
		t.Error(err)
	}

	// make sure the copy equals the original
	equal := true
	equal = equal && (m.Complex == mCopy.Complex)
	if len(m.IntSlice) != len(mCopy.IntSlice) {
		equal = false
	} else {
		for i, mInt := range m.IntSlice {
			mCopyInt := mCopy.IntSlice[i]
			equal = equal && (mInt == mCopyInt)
		}
	}
	if len(m.StringSlice) != len(mCopy.StringSlice) {
		equal = false
	} else {
		for i, mString := range m.StringSlice {
			mCopyString := mCopy.StringSlice[i]
			equal = equal && (mString == mCopyString)
		}
	}
	if len(m.IntArray) != len(mCopy.IntArray) {
		equal = false
	} else {
		for i, mInt := range m.IntArray {
			mCopyInt := mCopy.IntArray[i]
			equal = equal && (mInt == mCopyInt)
		}
	}
	if len(m.StringArray) != len(mCopy.StringArray) {
		equal = false
	} else {
		for i, mString := range m.StringArray {
			mCopyString := mCopy.StringArray[i]
			equal = equal && (mString == mCopyString)
		}
	}
	if len(m.StringMap) != len(mCopy.StringMap) {
		equal = false
	} else {
		for mKey, mValue := range m.StringMap {
			if mCopyValue, found := mCopy.StringMap[mKey]; !found {
				equal = false
			} else {
				equal = equal && (mValue == mCopyValue)
			}
		}
	}
	if len(m.IntMap) != len(mCopy.IntMap) {
		equal = false
	} else {
		for mKey, mValue := range m.IntMap {
			if mCopyValue, found := mCopy.IntMap[mKey]; !found {
				equal = false
			} else {
				equal = equal && (mValue == mCopyValue)
			}
		}
	}
	if !equal {
		t.Errorf("model was not saved/retrieved correctly.\nExpected: %+v\nGot %+v\n", m, mCopy)
	}
}

func TestEmbeddedStruct(t *testing.T) {
	construct := func() (interface{}, error) {
		return &embeddedStructModel{
			embed: embed{
				Int: 42,
			},
		}, nil
	}
	testConvertType(reflect.TypeOf(embeddedStructModel{}), construct, t)
}

func TestEmbeddedPointerToStruct(t *testing.T) {
	construct := func() (interface{}, error) {
		return &embeddedPointerToStructModel{
			embed: &embed{
				Int: 42,
			},
		}, nil
	}
	testConvertType(reflect.TypeOf(embeddedPointerToStructModel{}), construct, t)
}

// a general test that uses reflection
func testConvertType(typ reflect.Type, construct func() (in interface{}, err error), t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// make a model with all nil fields
	m1Interface := reflect.New(typ).Interface()
	m1, ok := m1Interface.(Model)
	if !ok {
		t.Errorf("couldn't convert type %T to Model", m1Interface)
	}

	// construct a model using the construct function
	m2Interface, err := construct()
	if err != nil {
		t.Error(err)
	}
	m2, ok := m2Interface.(Model)
	if !ok {
		t.Errorf("couldn't convert type %T to Model", m2Interface)
	}
	if err := MSave([]Model{m1, m2}); err != nil {
		t.Error(err)
	}

	// create a copy of the same type and use ScanById
	m2CopyInterface := reflect.New(typ).Interface()
	m2Copy, ok := m2CopyInterface.(Model)
	id := m2.GetId()
	if !ok {
		t.Errorf("couldn't convert type %T to Model", m2CopyInterface)
	}
	if err := ScanById(id, m2Copy); err != nil {
		t.Error(err)
	}

	// make sure the copy equals the original
	equal, err := looseEquals(m2, m2Copy)
	if err != nil {
		t.Error(err)
	}
	if !equal {
		t.Errorf("model was not saved/retrieved correctly.\nExpected: %+v\nGot %+v\n", m2, m2Copy)
	}
}
