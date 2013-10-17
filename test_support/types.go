// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File types.go contains type declarations that are used
// in various places, especially in test package

package test_support

import (
	"github.com/stephenalexbrowne/zoom"
)

type Person struct {
	Name string
	Age  int
	zoom.DefaultData
}

type ModelWithList struct {
	List []string `redisType:"list"`
	zoom.DefaultData
}

type ModelWithSet struct {
	Set []string `redisType:"set"`
	zoom.DefaultData
}

type Artist struct {
	Name          string
	FavoriteColor *Color
	zoom.DefaultData
}

type Color struct {
	R int
	G int
	B int
	zoom.DefaultData
}

type PetOwner struct {
	Name string
	Pets []*Pet
	zoom.DefaultData
}

type Pet struct {
	Name string
	zoom.DefaultData
}

type Friend struct {
	Name    string
	Friends []*Friend
	zoom.DefaultData
}

// PrimativeTypes is a struct containing all supported primative types
type PrimativeTypes struct {
	Uint    uint
	Uint8   uint8
	Uint16  uint16
	Uint32  uint32
	Uint64  uint64
	Int     int
	Int8    int8
	Int16   int16
	Int32   int32
	Int64   int64
	Float32 float32
	Float64 float64
	Byte    byte
	Rune    rune
	String  string
	zoom.DefaultData
}

// PointerPrimativeTypes is a struct containing pointers to all
// supported primative types.
type PointerPrimativeTypes struct {
	Uint    *uint
	Uint8   *uint8
	Uint16  *uint16
	Uint32  *uint32
	Uint64  *uint64
	Int     *int
	Int8    *int8
	Int16   *int16
	Int32   *int32
	Int64   *int64
	Float32 *float32
	Float64 *float64
	Byte    *byte
	Rune    *rune
	String  *string
	zoom.DefaultData
}

// InconvertibleTypes is a struct containing fields which are
// not directly convertible. A fallback (e.g. gob/json) encoding
// should be used.
type InconvertibleTypes struct {
	Complex     complex128
	IntSlice    []int
	StringSlice []string
	IntArray    [3]int
	StringArray [3]string
	StringMap   map[string]string
	IntMap      map[int]int
	zoom.DefaultData
}

// EmbeddedStruct is a struct containing an embedded struct
// of an unregistered type
type EmbeddedStruct struct {
	Embed
	zoom.DefaultData
}

// PointerEmbeddedStruct is a struct containing an embedded
// struct pointer of an unregistered type
type PointerEmbeddedStruct struct {
	*Embed
	zoom.DefaultData
}

type Embed struct {
	Int    int
	String string
}
