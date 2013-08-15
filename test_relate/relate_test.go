package test_relate

import (
	"github.com/stephenalexbrowne/zoom"
	"github.com/stephenalexbrowne/zoom/redis"
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

	// select database 9 and make sure it's empty
	zoom.Init(&zoom.Configuration{Database: 9})

	conn := zoom.GetConn()
	defer conn.Close()

	n, err := redis.Int(conn.Do("DBSIZE"))
	if err != nil {
		c.Error(err)
	}

	if n != 0 {
		c.Errorf("Database #9 is not empty, test can not continue")
	}

	// register the structs we plan to use
	err = zoom.Register(&Person{}, "person")
	if err != nil {
		c.Error(err)
	}

	err = zoom.Register(&Pet{}, "pet")
	if err != nil {
		c.Error(err)
	}

	err = zoom.Register(&Parent{}, "parent")
	if err != nil {
		c.Error(err)
	}

	err = zoom.Register(&Child{}, "child")
	if err != nil {
		c.Error(err)
	}
}

func (s *RelateSuite) TearDownSuite(c *C) {

	zoom.UnregisterName("person")
	zoom.UnregisterName("pet")
	zoom.UnregisterName("parent")
	zoom.UnregisterName("child")

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

	result, err := zoom.FindById("person", person.Id)
	if err != nil {
		c.Error(err)
	}

	person2, ok := result.(*Person)
	if !ok {
		c.Error("Couldn't type assert to *Person: ", person2)
	}

	pet2 := person2.Pet
	c.Assert(pet2, NotNil)
	c.Assert(pet2.Name, Equals, "Billy")
	c.Assert(pet2.Kind, Equals, "barracuda")

	// we'll test the inverse relationship separately for now.
	// Later, zoom might recognize this and set it automatically.
	pet2.Owner = person
	err = zoom.Save(pet2)
	if err != nil {
		c.Error(err)
	}

	result, err = zoom.FindById("pet", pet2.Id)
	if err != nil {
		c.Error(err)
	}

	pet3, ok := result.(*Pet)
	if !ok {
		c.Errorf("Couldn't convert result to *Pet")
	}

	person3 := pet3.Owner
	c.Assert(person3, NotNil)
	c.Assert(person3.Name, Equals, "Alex")
	c.Assert(person3.Age, Equals, 20)

}

func (s *RelateSuite) TestOneToMany(c *C) {
	// Create a Parent and two children
	parent := NewParent("Christine")
	child1 := NewChild("Derick")
	child2 := NewChild("Elise")

	// assign the children to the parent
	parent.Children = append(parent.Children, child1, child2)

	// save the parent
	err := zoom.Save(parent)
	if err != nil {
		c.Error(err)
	}

	// retrieve the parent from db
	reply, err := zoom.FindById("parent", parent.Id)
	if err != nil {
		c.Error(err)
	}

	// type assert it to *Parent
	parent2, ok := reply.(*Parent)
	if !ok {
		c.Errorf("Couldn't convert result to *Parent: ", reply)
	}

	// make sure that the children match the original
	// length should be 2
	c.Assert(len(parent2.Children), Equals, 2)
	// the names of the two children should be "Derick" and "Elise"
	expectedNames := []string{"Derick", "Elise"}
	for _, child := range parent2.Children {
		index := indexOfStringSlice(child.Name, expectedNames)
		if index == -1 {
			c.Error("Unexpected child.name: ", child.Name)
		}
		// remove from expected. makes sure we have one of each
		expectedNames = removeFromStringSlice(expectedNames, index)
	}
	// now expectedNames should be empty. If it's not, there's a problem
	if len(expectedNames) != 0 {
		c.Errorf("At least one expected child.name was not found: %v\n", expectedNames)
	}
}
