package test_relate

import (
	"github.com/stephenalexbrowne/zoom"
)

// File contains support code for tests.
// e.g. type declarations, constructors,
// and other methods.

// The PetOwner struct
type PetOwner struct {
	Name string
	Age  int
	Pet  *Pet
	zoom.DefaultData
}

// The Pet struct
type Pet struct {
	Name  string
	Kind  string
	Owner *PetOwner
	zoom.DefaultData
}

// The Parent struct
type Parent struct {
	Name     string
	Children []*Child
	zoom.DefaultData
}

// The Child struct
type Child struct {
	Name   string
	Parent *Parent
	zoom.DefaultData
}

// The Person struct
type Person struct {
	Name    string
	Friends []*Person
	zoom.DefaultData
}
