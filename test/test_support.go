package test

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
	Name string
	Kind string
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

// The AllTypes struct
// A struct containing all supported types
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

// Database helper functions
// setUp() and tearDown()
func setUp() error {
	zoom.Init(&zoom.Configuration{Database: 9})

	conn := zoom.GetConn()
	defer conn.Close()

	n, err := redis.Int(conn.Do("DBSIZE"))
	if err != nil {
		return err
	}

	if n != 0 {
		return errors.New("Database #9 is not empty, test can not continue")
	}

	zoom.Register(&Person{}, "person")
	return nil
}

func tearDown() {
	conn := zoom.GetConn()
	_, err := conn.Do("flushdb")
	if err != nil {
		log.Fatal(err)
	}
	conn.Close()
	zoom.Close()
}
