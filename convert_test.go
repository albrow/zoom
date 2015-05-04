// Copyright 2015 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File convert_test.go tests the conversion
// to and from go data structures of a variety of types.

package zoom

import (
	"reflect"
	"testing"
)

func TestConvertPrimatives(t *testing.T) {
	testingSetUp()
	defer testingTearDown()
	testConvertType(t, indexedPrimativesModels, createIndexedPrimativesModel())
}

func TestConvertPointers(t *testing.T) {
	testingSetUp()
	defer testingTearDown()
	testConvertType(t, indexedPointersModels, createIndexedPointersModel())
}

func TestConvertInconvertibles(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	type inconvertiblesModel struct {
		Complex     complex128
		IntSlice    []int
		StringSlice []string
		IntArray    [3]int
		StringArray [3]string
		StringMap   map[string]string
		IntMap      map[int]int
		DefaultData
	}
	inconvertiblesModels, err := Register(&inconvertiblesModel{})
	if err != nil {
		t.Errorf("Unexpected error in Register: %s", err.Error())
	}
	model := &inconvertiblesModel{
		Complex:     randomComplex(),
		IntSlice:    []int{randomInt(), randomInt(), randomInt()},
		StringSlice: []string{randomString(), randomString(), randomString()},
		IntArray:    [3]int{randomInt(), randomInt(), randomInt()},
		StringArray: [3]string{randomString(), randomString(), randomString()},
		StringMap:   map[string]string{randomString(): randomString(), randomString(): randomString()},
		IntMap:      map[int]int{randomInt(): randomInt(), randomInt(): randomInt()},
	}
	testConvertType(t, inconvertiblesModels, model)
}

type embeddable struct {
	Int    int
	String string
	Bool   bool
}

func TestConvertEmbeddedStruct(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	type embeddedStructModel struct {
		embeddable
		DefaultData
	}
	embededStructModels, err := Register(&embeddedStructModel{})
	if err != nil {
		t.Errorf("Unexpected error in Register: %s", err.Error())
	}
	model := &embeddedStructModel{
		embeddable: embeddable{
			Int:    randomInt(),
			String: randomString(),
			Bool:   randomBool(),
		},
	}
	testConvertType(t, embededStructModels, model)
}

func TestEmbeddedPointerToStruct(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	type embeddedPointerToStructModel struct {
		*embeddable
		DefaultData
	}
	embededPointerToStructModels, err := Register(&embeddedPointerToStructModel{})
	if err != nil {
		t.Errorf("Unexpected error in Register: %s", err.Error())
	}
	model := &embeddedPointerToStructModel{
		embeddable: &embeddable{
			Int:    randomInt(),
			String: randomString(),
			Bool:   randomBool(),
		},
	}
	testConvertType(t, embededPointerToStructModels, model)
}

// testConvertType is a general test that uses reflection. It saves model to the databse then finds it. If
// the found copy does not exactly match the original, it reports an error via t.Error or t.Errorf
func testConvertType(t *testing.T, modelType *ModelType, model Model) {
	// Make sure we can save the model without errors
	if err := modelType.Save(model); err != nil {
		t.Errorf("Unexpected error in Save: %s", err.Error())
	}
	// Find the model from the database and scan it into a new copy
	modelCopy, ok := reflect.New(modelType.spec.typ.Elem()).Interface().(Model)
	if !ok {
		t.Fatalf("Unexpected error: Could not convert type %s to Model", modelType.spec.typ.String())
	}
	if err := modelType.Find(model.Id(), modelCopy); err != nil {
		t.Errorf("Unexpected error in Find: %s", err.Error())
	}
	// Make sure the copy equals the original
	if !reflect.DeepEqual(model, modelCopy) {
		t.Errorf("Model of type %T was not saved/retrieved correctly.\nExpected: %+v\nGot:      %+v", model, model, modelCopy)
	}
	// Make sure we can save a model with all nil fields. This should
	// not cause an error.
	emptyModel, ok := reflect.New(modelType.spec.typ.Elem()).Interface().(Model)
	if !ok {
		t.Fatalf("Unexpected error: Could not convert type %s to Model", modelType.spec.typ.String())
	}
	if err := modelType.Save(emptyModel); err != nil {
		t.Errorf("Unexpected error saving an empty model: %s", err.Error())
	}
}
