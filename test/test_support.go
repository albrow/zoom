package test

import (
	"github.com/stephenalexbrowne/zoom"
)

// File contains support code for tests.
// e.g. type declarations, constructors,
// and other methods.
type Person struct {
	Name string
	Age  int
	*zoom.Model
}

// A convenient constructor for our Person struct
func NewPerson(name string, age int) *Person {
	p := &Person{
		Name:  name,
		Age:   age,
		Model: new(zoom.Model),
	}
	return p
}

type Pet struct {
	Name string
	Kind string
	*zoom.Model
}

// A convenient constructor for our Pet struct
func NewPet(name, kind string) *Pet {
	p := &Pet{
		Name:  name,
		Kind:  kind,
		Model: new(zoom.Model),
	}
	return p
}

type AllTypes struct {
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
	*zoom.Model
}
