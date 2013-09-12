// Files contains helper functions SetUp() and TearDown()

package support

import (
	"github.com/stephenalexbrowne/zoom"
	"github.com/stephenalexbrowne/zoom/redis"
	"math/rand"
	"time"
)

func SetUp() {
	// TODO: add a tcp fallback if sockets doesn't work

	// initialize zoom
	zoom.Init(&zoom.Configuration{
		Address:  "/tmp/redis.sock",
		Network:  "unix",
		Database: 9,
	})

	// get a connection
	conn := zoom.GetConn()
	defer conn.Close()

	// make sure database #9 is empty
	n, err := redis.Int(conn.Do("DBSIZE"))
	if err != nil {
		panic(err.Error())
	}
	if n != 0 {
		panic("Database #9 is not empty, test can not continue")
	}

	// register the types in types.go
	if err := zoom.Register(&Person{}, "person"); err != nil {
		panic(err.Error())
	}
	if err := zoom.Register(&ModelWithList{}, "modelWithList"); err != nil {
		panic(err.Error())
	}
	if err := zoom.Register(&ModelWithSet{}, "modelWithSet"); err != nil {
		panic(err.Error())
	}
	if err := zoom.Register(&Artist{}, "artist"); err != nil {
		panic(err.Error())
	}
	if err := zoom.Register(&Color{}, "color"); err != nil {
		panic(err.Error())
	}
	if err := zoom.Register(&PetOwner{}, "petOwner"); err != nil {
		panic(err.Error())
	}
	if err := zoom.Register(&Pet{}, "pet"); err != nil {
		panic(err.Error())
	}
	if err := zoom.Register(&Friend{}, "friend"); err != nil {
		panic(err.Error())
	}

	// generate a new seed for rand
	rand.Seed(time.Now().UTC().UnixNano())
}

func TearDown() {

	// unregister types in types.go
	zoom.UnregisterName("person")
	zoom.UnregisterName("modelWithList")
	zoom.UnregisterName("modelWithSet")
	zoom.UnregisterName("artist")
	zoom.UnregisterName("color")
	zoom.UnregisterName("petOwner")
	zoom.UnregisterName("pet")
	zoom.UnregisterName("friend")

	// flush and close the database
	conn := zoom.GetConn()
	_, err := conn.Do("flushdb")
	if err != nil {
		panic(err)
	}
	conn.Close()
	zoom.Close()
}
