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
	err = zoom.Register(&PetOwner{}, "petOwner")
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

	err = zoom.Register(&Person{}, "person")
	if err != nil {
		c.Error(err)
	}
}

func (s *RelateSuite) TearDownSuite(c *C) {

	zoom.UnregisterName("petOwner")
	zoom.UnregisterName("pet")
	zoom.UnregisterName("parent")
	zoom.UnregisterName("child")
	zoom.UnregisterName("person")

	conn := zoom.GetConn()
	_, err := conn.Do("flushdb")
	if err != nil {
		c.Error(err)
	}
	conn.Close()

	zoom.Close()
}

func (s *RelateSuite) TestOneToOne(c *C) {

	owner := NewPetOwner("Alex", 20)
	pet := NewPet("Billy", "barracuda")

	owner.Pet = pet
	err := zoom.Save(owner)
	if err != nil {
		c.Error(err)
	}

	result, err := zoom.FindById("petOwner", owner.Id)
	if err != nil {
		c.Error(err)
	}

	ownerCopy, ok := result.(*PetOwner)
	if !ok {
		c.Error("Couldn't type assert to *PetOwner: ", ownerCopy)
	}

	petCopy := ownerCopy.Pet
	c.Assert(petCopy, NotNil)
	c.Assert(petCopy.Name, Equals, "Billy")
	c.Assert(petCopy.Kind, Equals, "barracuda")

	// we'll test the inverse relationship separately for now.
	// Later, zoom might recognize this and set it automatically.
	petCopy.Owner = owner
	err = zoom.Save(petCopy)
	if err != nil {
		c.Error(err)
	}

	result, err = zoom.FindById("pet", petCopy.Id)
	if err != nil {
		c.Error(err)
	}

	petCopy2, ok := result.(*Pet)
	if !ok {
		c.Errorf("Couldn't convert result to *Pet")
	}

	personCopy2 := petCopy2.Owner
	c.Assert(personCopy2, NotNil)
	c.Assert(personCopy2.Name, Equals, "Alex")
	c.Assert(personCopy2.Age, Equals, 20)

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

func (s *RelateSuite) TestManyToMany(c *C) {

	// create 5 people
	fred := NewPerson("Fred")
	george := NewPerson("George")
	hellen := NewPerson("Hellen")
	ilene := NewPerson("Ilene")
	jim := NewPerson("Jim")

	// Fred is friends with George, Hellen, and Jim
	fred.Friends = append(fred.Friends, george, hellen, jim)

	// George is friends with Fred, Hellen, and Ilene
	george.Friends = append(george.Friends, fred, hellen, ilene)

	// save both Fred and George
	if err := zoom.Save(fred); err != nil {
		c.Error(err)
	}
	if err := zoom.Save(george); err != nil {
		c.Error(err)
	}

	// make sure that all 5 people were saved.
	// Hellen, Ilene, and Jim were not saved explicitly, but should
	// be saved because they are referenced by Fred and George.
	// This also checks to make sure Hellen wasn't saved twice.
	results, err := zoom.FindAll("person")
	if err != nil {
		c.Error(err)
	}
	c.Assert(len(results), Equals, 5)

	expectedNames := []string{"Fred", "George", "Hellen", "Ilene", "Jim"}
	gotNames := []string{}
	for _, result := range results {
		friend, ok := result.(*Person)
		if !ok {
			c.Errorf("Couldn't convert %+v to *Person\n", result)
		}
		gotNames = append(gotNames, friend.Name)
	}
	equal, msg := compareAsStringSet(expectedNames, gotNames)
	if !equal {
		c.Errorf(msg)
	}

	// retrieve fred from the database
	result, err := zoom.FindById("person", fred.Id)
	if err != nil {
		c.Error(err)
	}

	// type assert to *Person
	fredCopy, ok := result.(*Person)
	if !ok {
		c.Errorf("Could not convert result to *Person: %+v\n", result)
	}

	// make sure he remembers his name
	c.Assert(fredCopy.Name, Equals, "Fred")

	// make sure he remembers his friends
	expectedNames = []string{"George", "Hellen", "Jim"}
	gotNames = []string{}
	for _, friend := range fred.Friends {
		gotNames = append(gotNames, friend.Name)
	}
	equal, msg = compareAsStringSet(expectedNames, gotNames)
	if !equal {
		c.Errorf(msg)
	}

	// retrieve george from the database
	result, err = zoom.FindById("person", george.Id)
	if err != nil {
		c.Error(err)
	}

	// type assert to *Person
	georgeCopy, ok := result.(*Person)
	if !ok {
		c.Errorf("Could not convert result to *Person: %+v\n", result)
	}

	// make sure he remembers his name
	c.Assert(georgeCopy.Name, Equals, "George")

	// make sure he remembers his friends
	expectedNames = []string{"Fred", "Hellen", "Ilene"}
	gotNames = []string{}
	for _, friend := range george.Friends {
		gotNames = append(gotNames, friend.Name)
	}
	equal, msg = compareAsStringSet(expectedNames, gotNames)
	if !equal {
		c.Errorf(msg)
	}
}
