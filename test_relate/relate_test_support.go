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
