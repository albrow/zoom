package test

import (
	"github.com/stephenalexbrowne/zoom"
	. "launchpad.net/gocheck"
	"testing"
)

// We'll define a person struct as the basis of all our tests
// Throughout these, we will try to save, edit, relate, and delete
// Persons in the database
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

// Gocheck setup...
func Test(t *testing.T) {
	TestingT(t)
}

type MainSuite struct{}

var _ = Suite(&MainSuite{})

func (s *MainSuite) SetUpSuite(c *C) {
	_, err := zoom.InitDb()
	if err != nil {
		c.Error(err)
	}

	err = zoom.Register(&Person{}, "person")
	if err != nil {
		c.Error(err)
	}
}

func (s *MainSuite) TearDownSuite(c *C) {
	zoom.UnregisterName("person")
	_, err := zoom.Db().Do("flushdb")
	if err != nil {
		c.Error(err)
	}
	zoom.CloseDb()
}

func (s *MainSuite) TestSave(c *C) {
	p := NewPerson("Allen", 25)
	err := zoom.Save(p)
	if err != nil {
		c.Error(err)
	}
	c.Assert(p.Name, Equals, "Allen")
	c.Assert(p.Age, Equals, 25)
	c.Assert(p.Id, Not(Equals), "")
}

func (s *MainSuite) TestFindById(c *C) {
	// Create and save a new model
	p1 := NewPerson("Bill", 25)
	zoom.Save(p1)

	// find the model using FindById
	result, err := zoom.FindById("person", p1.Id)
	if err != nil {
		c.Error(err)
	}
	p2 := result.(*Person)

	// Make sure the found model is the same as original
	c.Assert(p2.Name, Equals, p1.Name)
	c.Assert(p2.Age, Equals, p1.Age)
	c.Assert(p2.Id, Equals, p1.Id)
}

func (s *MainSuite) TestDelete(c *C) {
	// Create and save a new model
	p1 := NewPerson("Charles", 25)
	zoom.Save(p1)

	// Make sure it was saved
	key := "person:" + p1.Id
	exists, err := zoom.KeyExists(key)
	if err != nil {
		c.Error(err)
	}
	c.Assert(exists, Equals, true)

	// delete it
	zoom.Delete(p1)

	// Make sure it's gone
	exists, err = zoom.KeyExists(key)
	if err != nil {
		c.Error(err)
	}
	c.Assert(exists, Equals, false)
}

func (s *MainSuite) TestDeleteById(c *C) {
	// Create and save a new model
	p1 := NewPerson("Debbie", 25)
	zoom.Save(p1)

	// Make sure it was saved
	key := "person:" + p1.Id
	exists, err := zoom.KeyExists(key)
	if err != nil {
		c.Error(err)
	}
	c.Assert(exists, Equals, true)

	// delete it
	zoom.DeleteById("person", p1.Id)

	// Make sure it's gone
	exists, err = zoom.KeyExists(key)
	if err != nil {
		c.Error(err)
	}
	c.Assert(exists, Equals, false)
}
