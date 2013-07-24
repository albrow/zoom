package zoom_test

import (
	"github.com/stephenalexbrowne/zoom"
	. "launchpad.net/gocheck"
	"testing"
	"time"
)

type Person struct {
	Name      string
	Age       int
	SiblingId string `refersTo:"person"`
	*zoom.Model
}

// Hook up gocheck into the "go test" runner.
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

func (s *MainSuite) TestSaveFindAndDelete(c *C) {
	p := &Person{
		Name:  "Bob",
		Age:   25,
		Model: new(zoom.Model),
	}
	result, err := zoom.Save(p)
	if err != nil {
		c.Error(err)
	}
	person, ok := result.(*Person)
	if !ok {
		c.Error("Could not convert result to person")
	}
	c.Assert(person, NotNil)
	c.Assert(person.Name, Equals, "Bob")
	c.Assert(person.Age, Equals, 25)
	c.Assert(person.Id, NotNil)

	// Since the database sets the id and we don't know what
	// it is until after we check the return of Save(), we
	// should also test the FindById and DeleteById methods here
	p2, err := zoom.FindById("person", person.Id)
	if err != nil {
		c.Error(err)
	}
	c.Assert(p2, DeepEquals, person)

	p3, err := zoom.DeleteById("person", person.Id)
	if err != nil {
		c.Error(err)
	}
	c.Assert(p3, DeepEquals, person)

	// Now that the thing has been deleted, we should make sure that
	// FindById returns a KeyNotFound error
	fooey, err := zoom.FindById("person", person.Id)
	c.Assert(fooey, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err, FitsTypeOf, zoom.NewKeyNotFoundError(""))
}

func (s *MainSuite) TestInvalidRefersToCausesError(c *C) {
	type InvalidPerson struct {
		SiblingId string `refersTo:"foo"` // this name hasn't been registered
		*zoom.Model
	}
	err := zoom.Register(&InvalidPerson{}, "invalid")
	if err != nil {
		c.Error(err)
	}
	p := &InvalidPerson{
		Model: new(zoom.Model),
	}
	_, err = zoom.Save(p)

	// We expect a ModelNameNotRegisteredError
	c.Assert(err, NotNil)
	c.Assert(err, FitsTypeOf, zoom.NewModelNameNotRegisteredError(""))
	zoom.UnregisterName("invalid")
}

func (s *MainSuite) TestInvalidRelationalIdCausesError(c *C) {
	p := &Person{
		SiblingId: "invalidId", // this id doesn't exist
		Model:     new(zoom.Model),
	}
	_, err := zoom.Save(p)

	// We expect a KeyNotFoundError
	c.Assert(err, NotNil)
	c.Assert(err, FitsTypeOf, zoom.NewKeyNotFoundError(""))
}

func (s *MainSuite) TestSaveSibling(c *C) {
	p1 := &Person{
		Name:  "Alice",
		Age:   27,
		Model: new(zoom.Model),
	}
	result, _ := zoom.Save(p1)
	person1 := result.(*Person)
	p2 := &Person{
		Name:      "Bob",
		Age:       25,
		SiblingId: person1.Id,
		Model:     new(zoom.Model),
	}
	result, _ = zoom.Save(p2)
	person2 := result.(*Person)
	person1.SiblingId = person2.Id
	result, _ = zoom.Save(person1)
	person1 = result.(*Person)

	c.Assert(person1.SiblingId, Equals, person2.Id)
	c.Assert(person2.SiblingId, Equals, person1.Id)
}
