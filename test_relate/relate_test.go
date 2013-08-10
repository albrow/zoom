package test_relate

import (
	"github.com/stephenalexbrowne/zoom"
	. "launchpad.net/gocheck"
	"testing"
)

// Gocheck setup...
func Test(t *testing.T) {
	TestingT(t)
}

type RelateSuite struct{}

var _ = Suite(&RelateSuite{})

func (s *RelateSuite) SetUpSuite(c *C) {

	zoom.Init(&zoom.Configuration{Database: 7})

	err := zoom.Register(&Person{}, "person")
	if err != nil {
		c.Error(err)
	}

	err = zoom.Register(&Pet{}, "pet")
	if err != nil {
		c.Error(err)
	}
}

func (s *RelateSuite) TearDownSuite(c *C) {

	zoom.UnregisterName("person")
	zoom.UnregisterName("pet")

	conn := zoom.GetConn()
	_, err := conn.Do("flushdb")
	if err != nil {
		c.Error(err)
	}
	conn.Close()

	zoom.Close()
}

func (s *RelateSuite) TestOneToOne(c *C) {
	person := NewPerson("Alex", 20)
	pet := NewPet("Billy", "barracuda")

	person.Pet = pet
	err := zoom.Save(person)
	if err != nil {
		c.Error(err)
	}

	// result, err := zoom.FindById("person", person.Id)
	// if err != nil {
	// 	c.Error(err)
	// }

	// person2, ok := result.(*Person)
	// if !ok {
	// 	c.Error("Couldn't type assert to *Person: ", person2)
	// }

	// pet2 := person2.Pet
	// c.Assert(pet2, NotNil)
	// c.Assert(pet2.Name, Equals, "Billy")
	// c.Assert(pet2.Kind, Equals, "barracuda")
}
