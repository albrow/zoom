package zoom_test

import (
	"github.com/stephenalexbrowne/zoom"
	. "launchpad.net/gocheck"
	"testing"
	"time"
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

// Gocheck setup...
func Test(t *testing.T) {
	TestingT(t)
}

type MainSuite struct{}

var _ = Suite(&MainSuite{})

func (s *MainSuite) SetUpSuite(c *C) {
	config := zoom.DbConfig{
		Timeout:  10 * time.Second,
		Database: 15,
		PoolSize: 99999,
	}
	zoom.InitDb(config)

	err := zoom.Register(&Person{}, "person")
	if err != nil {
		c.Error(err)
	}
}

func (s *MainSuite) TearDownSuite(c *C) {
	zoom.CloseDb()
}

func (s *MainSuite) TestSave(c *C) {
	p := NewPerson("Bob", 25)
	err := p.Save()
	if err != nil {
		c.Error(err)
	}
	c.Assert(p.Name, Equals, "Bob")
	c.Assert(p.Age, Equals, 25)
	c.Assert(p.Id, NotNil)
}

func (s *MainSuite) TestFindById(c *C) {
	// Create and save a new model
	p1 := NewPerson("Jane", 26)
	p1.Save()

	// find the model using FindById
	result, err := zoom.FindById("person", p1.Id)
	if err != nil {
		c.Error(err)
	}
	p2 := result.(*Person)

	// Make sure the found model is the same as original
	c.Assert(p2.Name, Equals, p1.Name)
	c.Assert(p2.Age, Equals, p1.Age)
}

func (s *MainSuite) TestDeleteById(c *C) {
	// Create and save a new model
	p := NewPerson("Bill", 22)
	p.Save()

	// Delete it
	err := zoom.DeleteById("person", p.Id)
	if err != nil {
		c.Error(err)
	}

	// Now that the thing has been deleted, we should make sure that
	// FindById returns a KeyNotFound error
	fooey, err := zoom.FindById("person", p.Id)
	c.Assert(fooey, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err, FitsTypeOf, zoom.NewKeyNotFoundError(""))
}

func (s *MainSuite) TestInvalidRefersToCausesError(c *C) {
	// Here we'll create a new struct InvalidPerson
	// We don't really need a constructor since we're only
	// doing this once
	type InvalidPerson struct {
		SiblingId string `refersTo:"foo"` // this name hasn't been registered
		*zoom.Model
	}
	err := zoom.Register(&InvalidPerson{}, "invalid")
	if err != nil {
		c.Error(err)
	}

	// Create a new InvalidPerson
	p := &InvalidPerson{}
	p.Model = zoom.NewModelFor(p)

	// Try to save it
	err = p.Save()

	// We expect a ModelNameNotRegisteredError because refersTo:"foo" is not a valid
	// registered model name
	c.Assert(err, NotNil)
	c.Assert(err, FitsTypeOf, zoom.NewModelNameNotRegisteredError(""))
	zoom.UnregisterName("invalid")
}

// TODO: change this so that the error occurs on Fetch() not Save()
func (s *MainSuite) TestInvalidRelationalIdCausesError(c *C) {
	// Create and save a new Person
	p := NewPerson("foo", 99)
	p.SiblingId = "invalid" // not a valid person id
	err := p.Save()

	// We expect a KeyNotFoundError because "invalid" is not a valid id
	c.Assert(err, NotNil)
	c.Assert(err, FitsTypeOf, zoom.NewKeyNotFoundError(""))
}

func (s *MainSuite) TestSaveSibling(c *C) {
	// Create and save two new persons: p1 and p2
	p1 := NewPerson("Alice", 27)
	p1.Save()
	p2 := NewPerson("Bob", 25)
	p2.SiblingId = p1.Id
	p2.Save()
	p1.SiblingId = p2.Id
	p1.Save()

	// Set p1 as a sibling of p2 and p2 as a sibling of p1
	c.Assert(p1.SiblingId, Equals, p2.Id)
	c.Assert(p2.SiblingId, Equals, p1.Id)

	// fetch the sibling of p1 (should equal p2)
	result, err := p1.Fetch("sibling")
	if err != nil {
		c.Error(err)
	}
	c.Assert(result, DeepEquals, p2)

	// fetch the sibling of p2 (should equal p1)
	result, err = p2.Fetch("sibling")
	if err != nil {
		c.Error(err)
	}
	c.Assert(result, DeepEquals, p1)

}
