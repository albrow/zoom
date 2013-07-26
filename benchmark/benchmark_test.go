package benchmark

import (
	"github.com/stephenalexbrowne/zoom"
	"testing"
)

// We'll define a person struct as the basis of all our tests
// Throughout these, we will try to save, edit, relate, and delete
// Persons in the database
type Person struct {
	Name      string
	Age       int
	SiblingId string `refersTo:"person" as:"sibling"`
	*zoom.Model
}

// A convenient constructor for our Person struct
func NewPerson(name string, age int) *Person {
	p := &Person{
		Name: name,
		Age:  age,
	}
	p.Model = zoom.NewModelFor(p)
	return p
}

func BenchmarkSave(b *testing.B) {

	config := zoom.DbConfig{
		Database:   15,
		PoolSize:   99999,
		UseSockets: true,
		Address:    "/tmp/redis.sock",
	}
	zoom.InitDb(config)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := NewPerson("Bob", 25)
		p.Save()
	}
}

func BenchmarkFindById(b *testing.B) {
	config := zoom.DbConfig{
		Database:   15,
		PoolSize:   99999,
		UseSockets: true,
		Address:    "/tmp/redis.sock",
	}
	zoom.InitDb(config)
	p := NewPerson("Bob", 25)
	p.Save()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		zoom.FindById("person", p.Id)
	}
}

func BenchmarkOneToOneRelation(b *testing.B) {
	config := zoom.DbConfig{
		Database:   15,
		PoolSize:   99999,
		UseSockets: true,
		Address:    "/tmp/redis.sock",
	}
	zoom.InitDb(config)
	p1 := NewPerson("Alice", 27)
	p1.Save()
	p2 := NewPerson("Bob", 25)
	p2.SiblingId = p1.Id
	p2.Save()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p2.Fetch("sibling")
	}

}
