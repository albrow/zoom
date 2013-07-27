package benchmark

import (
	"github.com/stephenalexbrowne/zoom"
	"log"
)

type Person struct {
	Id   string
	Name string
	Age  int
}

func (p *Person) SetId(id string) {
	p.Id = id
}

func (p *Person) GetId() string {
	return p.Id
}

// A convenient constructor for our Person struct
func NewPerson(name string, age int) *Person {
	p := &Person{
		Name: name,
		Age:  age,
	}
	return p
}

// Database helper functions
// setUp() and tearDown()
func setUp() {
	zoom.Init()
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
