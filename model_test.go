// Copyright 2015 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File model_test.go contains code for testing the model.go file.

package zoom

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestCompileModelSpec(t *testing.T) {
	type Primative struct {
		Int    int
		String string
		Bool   bool
	}
	type Unexported struct {
		privateInt    int
		privateString string
		privateBool   bool
	}
	type Pointer struct {
		Int    *int
		String *string
		Bool   *bool
	}
	type Indexed struct {
		Int    int    `zoom:"index"`
		String string `zoom:"index"`
		Bool   bool   `zoom:"index"`
	}
	type Ignored struct {
		Int    int    `redis:"-"`
		String string `redis:"-"`
		Bool   bool   `redis:"-"`
	}
	type CustomName struct {
		Int    int    `redis:"myInt"`
		String string `redis:"myString"`
		Bool   bool   `redis:"myBool"`
	}
	type Embedded struct {
		Primative
	}
	type private struct {
		Int int
	}
	type EmbeddedPrivate struct {
		private
	}
	testCases := []struct {
		model        interface{}
		expectedSpec *modelSpec
	}{
		{
			model: &Primative{},
			expectedSpec: &modelSpec{
				typ:  reflect.TypeOf(&Primative{}),
				name: "Primative",
				fieldsByName: map[string]*fieldSpec{
					"Int": &fieldSpec{
						kind:      primativeField,
						name:      "Int",
						redisName: "Int",
						typ:       reflect.TypeOf(Primative{}.Int),
						indexKind: noIndex,
					},
					"String": &fieldSpec{
						kind:      primativeField,
						name:      "String",
						redisName: "String",
						typ:       reflect.TypeOf(Primative{}.String),
						indexKind: noIndex,
					},
					"Bool": &fieldSpec{
						kind:      primativeField,
						name:      "Bool",
						redisName: "Bool",
						typ:       reflect.TypeOf(Primative{}.Bool),
						indexKind: noIndex,
					},
				},
				fields: []*fieldSpec{
					{
						kind:      primativeField,
						name:      "Int",
						redisName: "Int",
						typ:       reflect.TypeOf(Primative{}.Int),
						indexKind: noIndex,
					},
					{
						kind:      primativeField,
						name:      "String",
						redisName: "String",
						typ:       reflect.TypeOf(Primative{}.String),
						indexKind: noIndex,
					},
					{
						kind:      primativeField,
						name:      "Bool",
						redisName: "Bool",
						typ:       reflect.TypeOf(Primative{}.Bool),
						indexKind: noIndex,
					},
				},
			},
		},
		{
			model: &Unexported{},
			expectedSpec: &modelSpec{
				typ:          reflect.TypeOf(&Unexported{}),
				name:         "Unexported",
				fieldsByName: map[string]*fieldSpec{},
			},
		},
		{
			model: &Pointer{},
			expectedSpec: &modelSpec{
				typ:  reflect.TypeOf(&Pointer{}),
				name: "Pointer",
				fieldsByName: map[string]*fieldSpec{
					"Int": &fieldSpec{
						kind:      pointerField,
						name:      "Int",
						redisName: "Int",
						typ:       reflect.TypeOf(Pointer{}.Int),
						indexKind: noIndex,
					},
					"String": &fieldSpec{
						kind:      pointerField,
						name:      "String",
						redisName: "String",
						typ:       reflect.TypeOf(Pointer{}.String),
						indexKind: noIndex,
					},
					"Bool": &fieldSpec{
						kind:      pointerField,
						name:      "Bool",
						redisName: "Bool",
						typ:       reflect.TypeOf(Pointer{}.Bool),
						indexKind: noIndex,
					},
				},
				fields: []*fieldSpec{
					{
						kind:      pointerField,
						name:      "Int",
						redisName: "Int",
						typ:       reflect.TypeOf(Pointer{}.Int),
						indexKind: noIndex,
					},
					{
						kind:      pointerField,
						name:      "String",
						redisName: "String",
						typ:       reflect.TypeOf(Pointer{}.String),
						indexKind: noIndex,
					},
					{
						kind:      pointerField,
						name:      "Bool",
						redisName: "Bool",
						typ:       reflect.TypeOf(Pointer{}.Bool),
						indexKind: noIndex,
					},
				},
			},
		},
		{
			model: &Indexed{},
			expectedSpec: &modelSpec{
				typ:  reflect.TypeOf(&Indexed{}),
				name: "Indexed",
				fieldsByName: map[string]*fieldSpec{
					"Int": &fieldSpec{
						kind:      primativeField,
						name:      "Int",
						redisName: "Int",
						typ:       reflect.TypeOf(Indexed{}.Int),
						indexKind: numericIndex,
					},
					"String": &fieldSpec{
						kind:      primativeField,
						name:      "String",
						redisName: "String",
						typ:       reflect.TypeOf(Indexed{}.String),
						indexKind: stringIndex,
					},
					"Bool": &fieldSpec{
						kind:      primativeField,
						name:      "Bool",
						redisName: "Bool",
						typ:       reflect.TypeOf(Indexed{}.Bool),
						indexKind: booleanIndex,
					},
				},
				fields: []*fieldSpec{
					{
						kind:      primativeField,
						name:      "Int",
						redisName: "Int",
						typ:       reflect.TypeOf(Indexed{}.Int),
						indexKind: numericIndex,
					},
					{
						kind:      primativeField,
						name:      "String",
						redisName: "String",
						typ:       reflect.TypeOf(Indexed{}.String),
						indexKind: stringIndex,
					},
					{
						kind:      primativeField,
						name:      "Bool",
						redisName: "Bool",
						typ:       reflect.TypeOf(Indexed{}.Bool),
						indexKind: booleanIndex,
					},
				},
			},
		},
		{
			model: &Ignored{},
			expectedSpec: &modelSpec{
				typ:          reflect.TypeOf(&Ignored{}),
				name:         "Ignored",
				fieldsByName: map[string]*fieldSpec{},
			},
		},
		{
			model: &CustomName{},
			expectedSpec: &modelSpec{
				typ:  reflect.TypeOf(&CustomName{}),
				name: "CustomName",
				fieldsByName: map[string]*fieldSpec{
					"Int": &fieldSpec{
						kind:      primativeField,
						name:      "Int",
						redisName: "myInt",
						typ:       reflect.TypeOf(CustomName{}.Int),
						indexKind: noIndex,
					},
					"String": &fieldSpec{
						kind:      primativeField,
						name:      "String",
						redisName: "myString",
						typ:       reflect.TypeOf(CustomName{}.String),
						indexKind: noIndex,
					},
					"Bool": &fieldSpec{
						kind:      primativeField,
						name:      "Bool",
						redisName: "myBool",
						typ:       reflect.TypeOf(CustomName{}.Bool),
						indexKind: noIndex,
					},
				},
				fields: []*fieldSpec{
					{
						kind:      primativeField,
						name:      "Int",
						redisName: "myInt",
						typ:       reflect.TypeOf(CustomName{}.Int),
						indexKind: noIndex,
					},
					{
						kind:      primativeField,
						name:      "String",
						redisName: "myString",
						typ:       reflect.TypeOf(CustomName{}.String),
						indexKind: noIndex,
					},
					{
						kind:      primativeField,
						name:      "Bool",
						redisName: "myBool",
						typ:       reflect.TypeOf(CustomName{}.Bool),
						indexKind: noIndex,
					},
				},
			},
		},
		{
			model: &Embedded{},
			expectedSpec: &modelSpec{
				typ:  reflect.TypeOf(&Embedded{}),
				name: "Embedded",
				fieldsByName: map[string]*fieldSpec{
					"Primative": {
						kind:      inconvertibleField,
						name:      "Primative",
						redisName: "Primative",
						typ:       reflect.TypeOf(Primative{}),
						indexKind: noIndex,
					},
				},
				fields: []*fieldSpec{
					{
						kind:      inconvertibleField,
						name:      "Primative",
						redisName: "Primative",
						typ:       reflect.TypeOf(Primative{}),
						indexKind: noIndex,
					},
				},
			},
		},
		{
			model: &EmbeddedPrivate{},
			expectedSpec: &modelSpec{
				typ:          reflect.TypeOf(&EmbeddedPrivate{}),
				name:         "EmbeddedPrivate",
				fieldsByName: map[string]*fieldSpec{},
			},
		},
	}
	for _, tc := range testCases {
		gotSpec, err := compileModelSpec(reflect.TypeOf(tc.model))
		if err != nil {
			t.Error("Error compiling model spec: ", err.Error())
			continue
		}
		if !reflect.DeepEqual(tc.expectedSpec, gotSpec) {
			t.Errorf(
				"Incorrect model spec.\nExpected: %s\nBut got:  %s\n",
				spew.Sprint(tc.expectedSpec),
				spew.Sprint(gotSpec),
			)
		}
	}
}
