// Files contains helper functions SetUp() and TearDown()

package support

import (
	"github.com/stephenalexbrowne/zoom"
	"github.com/stephenalexbrowne/zoom/redis"
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
}

func TearDown() {

	// unregister types in types.go
	zoom.UnregisterName("person")

	// flush and close the database
	conn := zoom.GetConn()
	_, err := conn.Do("flushdb")
	if err != nil {
		panic(err)
	}
	conn.Close()
	zoom.Close()
}
