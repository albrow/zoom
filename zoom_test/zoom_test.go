package zoom_test

import (
	"github.com/stephenalexbrowne/zoom"
	. "launchpad.net/gocheck"
	"testing"
	"time"
)

type Person struct {
	Name string
	Age  int
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
