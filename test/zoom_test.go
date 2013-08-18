package test

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

type MainSuite struct{}

var _ = Suite(&MainSuite{})

func (s *MainSuite) SetUpSuite(c *C) {

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

	err = zoom.Register(&Person{}, "person")
	if err != nil {
		c.Error(err)
	}
}

func (s *MainSuite) TearDownSuite(c *C) {
	zoom.UnregisterName("person")
	conn := zoom.GetConn()
	_, err := conn.Do("flushdb")
	if err != nil {
		c.Error(err)
	}
	conn.Close()
	zoom.Close()
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

	// make sure it was added to the index
	ismem, err := zoom.SetContains("person:index", p.Id, nil)
	if err != nil {
		c.Error(err)
	}
	c.Assert(ismem, Equals, true)
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

func (s *MainSuite) TestScanById(c *C) {
	// Create and save a new model
	p1 := NewPerson("Joe", 23)
	zoom.Save(p1)

	// find and scan the model using ScanById
	p2 := &Person{}
	err := zoom.ScanById(p2, p1.Id)
	if err != nil {
		c.Error(err)
	}

	// Make sure the found model is the same as original
	c.Assert(p2.Name, Equals, p1.Name)
	c.Assert(p2.Age, Equals, p1.Age)
	c.Assert(p2.Id, Equals, p1.Id)
}

func (s *MainSuite) TestDelete(c *C) {
	// Create and save a new model
	p := NewPerson("Charles", 25)
	zoom.Save(p)

	// Make sure it was saved
	key := "person:" + p.Id
	exists, err := zoom.KeyExists(key, nil)
	if err != nil {
		c.Error(err)
	}
	c.Assert(exists, Equals, true)

	// delete it
	zoom.Delete(p)

	// Make sure it's gone
	exists, err = zoom.KeyExists(key, nil)
	if err != nil {
		c.Error(err)
	}
	c.Assert(exists, Equals, false)

	// Make sure it was removed from index
	ismem, err := zoom.SetContains("person:index", p.Id, nil)
	if err != nil {
		c.Error(err)
	}
	c.Assert(ismem, Equals, false)

}

func (s *MainSuite) TestDeleteById(c *C) {
	// Create and save a new model
	p := NewPerson("Debbie", 25)
	zoom.Save(p)

	// Make sure it was saved
	key := "person:" + p.Id
	exists, err := zoom.KeyExists(key, nil)
	if err != nil {
		c.Error(err)
	}
	c.Assert(exists, Equals, true)

	// delete it
	zoom.DeleteById("person", p.Id)

	// Make sure it's gone
	exists, err = zoom.KeyExists(key, nil)
	if err != nil {
		c.Error(err)
	}
	c.Assert(exists, Equals, false)

	// Make sure it was removed from index
	ismem, err := zoom.SetContains("person:index", p.Id, nil)
	if err != nil {
		c.Error(err)
	}
	c.Assert(ismem, Equals, false)
}

func (s *MainSuite) TestFindAll(c *C) {

	// register the Pet model
	err := zoom.Register(new(Pet), "pet")
	if err != nil {
		c.Error(err)
	}

	// Create and save some Pets
	// we can assume this works since
	// Save() was tested previously
	p1 := NewPet("Elroy", "emu")
	p2 := NewPet("Fred", "ferret")
	p3 := NewPet("Gus", "gecko")
	zoom.Save(p1)
	zoom.Save(p2)
	zoom.Save(p3)

	// query to get a list of all the pets
	results, err := zoom.FindAll("pet")
	if err != nil {
		c.Error(err)
	}

	// make sure the results is the right length
	c.Assert(len(results), Equals, 3)

	// make sure each item in results is correct
	// NOTE: this is tricky because the order can
	// change in redis and we haven't asked for any sorting
	expecteds := []*Pet{p1, p2, p3}
	for i, result := range results {
		// first, each item in results should be able to be casted to *Pet
		pResult, ok := result.(*Pet)
		if !ok {
			c.Errorf("Couldn't cast results[%d] to *Pet", i)
		}
		// second, each item in results should have a valid Id
		if pResult.Id == "" {
			c.Error("Id was not set for Pet: ", pResult)
		}
		// third, each item in results should correspond and be equal to
		// one of the items in expecteds
		foundIdMatch := false
		for _, expected := range expecteds {
			if pResult.Id == expected.Id {
				c.Assert(pResult.Name, Equals, expected.Name)
				c.Assert(pResult.Kind, Equals, expected.Kind)
				foundIdMatch = true
				break
			}
		}
		// if we've reached here, the result.Id did not match any of the ids
		// in expecteds.
		if foundIdMatch == false {
			c.Errorf("Couldn't find matching id in result set for: %s", pResult.Id)
		}
	}
}

// NOTE:
// All primative types are supported except complex64 and complex128
// A full list of supported types:
//
// uint
// uint8
// uint16
// uint32
// uint64
// int
// int8
// int16
// int32
// int64
// float32
// float64
// complex64
// complex128
// byte
// rune
// string
//

func (s *MainSuite) TestSaveSupportedTypes(c *C) {

	// Register our types model
	zoom.Register(&AllTypes{}, "types")

	// create and save a struct which contains all the numeric types (except complex)
	myTypes := &AllTypes{
		Uint:    uint(0),
		Uint8:   uint8(1),
		Uint16:  uint16(2),
		Uint32:  uint32(3),
		Uint64:  uint64(4),
		Int:     5,
		Int8:    int8(6),
		Int16:   int16(7),
		Int32:   int32(8),
		Int64:   int64(9),
		Float32: float32(10.0),
		Float64: float64(11.0),
		Byte:    byte(12),
		Rune:    rune(13),
		String:  "14",
		Model:   new(zoom.Model),
	}
	err := zoom.Save(myTypes)
	if err != nil {
		c.Error(err)
	}
	c.Assert(myTypes.Id, NotNil)

	// retrieve it from the database and typecast
	result, err := zoom.FindById("types", myTypes.Id)
	if err != nil {
		c.Error(err)
	}
	resultTypes, ok := result.(*AllTypes)
	if !ok {
		c.Error("Couldn't type assert to *AllTypes!")
	}

	// make sure all the struct members are the same as before
	c.Assert(resultTypes.Uint, Equals, myTypes.Uint)
	c.Assert(resultTypes.Uint8, Equals, myTypes.Uint8)
	c.Assert(resultTypes.Uint16, Equals, myTypes.Uint16)
	c.Assert(resultTypes.Uint32, Equals, myTypes.Uint32)
	c.Assert(resultTypes.Uint64, Equals, myTypes.Uint64)
	c.Assert(resultTypes.Int, Equals, myTypes.Int)
	c.Assert(resultTypes.Int8, Equals, myTypes.Int8)
	c.Assert(resultTypes.Int16, Equals, myTypes.Int16)
	c.Assert(resultTypes.Int32, Equals, myTypes.Int32)
	c.Assert(resultTypes.Int64, Equals, myTypes.Int64)
	c.Assert(resultTypes.Float32, Equals, myTypes.Float32)
	c.Assert(resultTypes.Float64, Equals, myTypes.Float64)
	c.Assert(resultTypes.Byte, Equals, myTypes.Byte)
	c.Assert(resultTypes.Rune, Equals, myTypes.Rune)

}
