// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File types.go contains type declarations that are used
// in various places, especially in test package

package test_support

import (
	"github.com/stephenalexbrowne/zoom"
)

// The Person struct
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

// The PrimativeTypes struct
// A struct containing all supported primative types
// and pointers to primative types
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
