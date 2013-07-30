package test

import (
	"github.com/stephenalexbrowne/zoom"
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
	zoom.Init()

	err := zoom.Register(&Person{}, "person")
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

func (s *MainSuite) TestDelete(c *C) {
	// Create and save a new model
	p1 := NewPerson("Charles", 25)
	zoom.Save(p1)

	// Make sure it was saved
	key := "person:" + p1.Id
	exists, err := zoom.KeyExists(key, nil)
	if err != nil {
		c.Error(err)
	}
	c.Assert(exists, Equals, true)

	// delete it
	zoom.Delete(p1)

	// Make sure it's gone
	exists, err = zoom.KeyExists(key, nil)
	if err != nil {
		c.Error(err)
	}
	c.Assert(exists, Equals, false)

	// Make sure it was removed from index
	ismem, err := zoom.SetContains("person:index", p1.Id, nil)
	if err != nil {
		c.Error(err)
	}
	c.Assert(ismem, Equals, false)

}

func (s *MainSuite) TestDeleteById(c *C) {
	// Create and save a new model
	p1 := NewPerson("Debbie", 25)
	zoom.Save(p1)

	// Make sure it was saved
	key := "person:" + p1.Id
	exists, err := zoom.KeyExists(key, nil)
	if err != nil {
		c.Error(err)
	}
	c.Assert(exists, Equals, true)

	// delete it
	zoom.DeleteById("person", p1.Id)

	// Make sure it's gone
	exists, err = zoom.KeyExists(key, nil)
	if err != nil {
		c.Error(err)
	}
	c.Assert(exists, Equals, false)
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
