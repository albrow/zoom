package test_relate

import (
	"github.com/stephenalexbrowne/zoom"
)

// File contains support code for tests.
// e.g. type declarations, constructors,
// and other methods.

// The Person struct
type Person struct {
	Name string
	Age  int
	Pet  *Pet
	*zoom.Model
}

// A convenient constructor for the Person struct
func NewPerson(name string, age int) *Person {
	p := &Person{
		Name:  name,
		Age:   age,
		Model: new(zoom.Model),
	}
	return p
}

// The Pet struct
type Pet struct {
	Name  string
	Kind  string
	Owner *Person
	*zoom.Model
}

// A convenient constructor for the Pet struct
func NewPet(name, kind string) *Pet {
	p := &Pet{
		Name:  name,
		Kind:  kind,
		Model: new(zoom.Model),
	}
	return p
}

// The Parent struct
type Parent struct {
	Name     string
	Children []*Child
	*zoom.Model
}

// A convenient constructor for the Parent struct
func NewParent(name string) *Parent {
	return &Parent{
		Name:  name,
		Model: new(zoom.Model),
	}
}

// The Child struct
type Child struct {
	Name   string
	Parent *Parent
	*zoom.Model
}

// A convenient constructor for the Child struct
func NewChild(name string) *Child {
	return &Child{
		Name:  name,
		Model: new(zoom.Model),
	}
}

func indexOfStringSlice(a string, list []string) int {
	for i, b := range list {
		if b == a {
			return i
		}
	}
	return -1
}

func removeFromStringSlice(list []string, i int) []string {
	return append(list[:i], list[i+1:]...)
}
