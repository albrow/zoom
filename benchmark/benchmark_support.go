package benchmark

import (
	"github.com/stephenalexbrowne/zoom"
	"log"
)

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

// Database helper functions
// setUp() and tearDown()
func setUp() {
	zoom.Init(&zoom.Configuration{Database: 7})
	zoom.Register(&Person{}, "person")
	conn := zoom.GetConn()
	_, err := conn.Do("flushdb")
	if err != nil {
		log.Fatal(err)
	}
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
