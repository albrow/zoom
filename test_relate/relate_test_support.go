package test_relate

import (
	"fmt"
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
	*zoom.Model
}

// A convenient constructor for the PetOwner struct
func NewPetOwner(name string, age int) *PetOwner {
	p := &PetOwner{
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
	Owner *PetOwner
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

// The Person struct
type Person struct {
	Name    string
	Friends []*Person
	*zoom.Model
}

// A convenient constructor for the Person struct
func NewPerson(name string) *Person {
	p := &Person{
		Name:  name,
		Model: new(zoom.Model),
	}
	return p
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

func compareAsStringSet(expecteds, gots []string) (bool, string) {
	for _, got := range gots {
		index := indexOfStringSlice(got, expecteds)
		if index == -1 {
			msg := fmt.Sprintf("Found unexpected element: %v", got)
			return false, msg
		}
		// remove from expecteds. makes sure we have one of each
		expecteds = removeFromStringSlice(expecteds, index)
	}
	// now expecteds should be empty. If it's not, there's a problem
	if len(expecteds) != 0 {
		msg := fmt.Sprintf("The following expected elements were not found: %v\n", expecteds)
		return false, msg
	}
	return true, "ok"
}
