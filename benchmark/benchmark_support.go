package benchmark

import (
	"github.com/stephenalexbrowne/zoom"
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
	zoom.InitDb()
	zoom.Register(&Person{}, "person")
}

func tearDown() {
	zoom.CloseDb()
}
