package zoom

// File contains code strictly related to the database, including
// establishing a connection, instantiating a package-wide db var
// and closing the connection. There are also convenience functions
// for (e.g.) checking if a key exists in redis.

import (
	"fmt"
	"github.com/dchest/uniuri"
	"github.com/garyburd/redigo/redis"
	"strconv"
	"time"
)

var db redis.Conn

// initializes and returns the database connection
func InitDb() (redis.Conn, error) {
	if db != nil {
		return db, nil
	} else {
		fmt.Println("zoom: connecting to database...")
		temp, err := redis.Dial("unix", "/tmp/redis.sock")
		if err != nil {
			return nil, err
		}
		db = temp
		// TODO: allow a config variable that sets the databse
		reply, err := db.Do("select", 7)
		if err != nil {
			fmt.Println(redis.String(reply, err))
			return nil, err
		}
		return db, nil
	}
}

func Db() redis.Conn {
	return db
}

// closes the connection to the database
func CloseDb() {
	if db != nil {
		fmt.Println("zoom: closing database connection...")
		db.Close()
	}
}

// Returns true iff a given key exists in redis
func KeyExists(key string) (bool, error) {
	return redis.Bool(db.Do("exists", key))
}

// generates a random string that is more or less
// garunteed to be unique. Used as Ids for records
// where an Id is not otherwise provided.
func generateRandomId() string {
	timeInt := time.Now().Unix()
	timeString := strconv.FormatInt(timeInt, 36)
	randomString := uniuri.NewLen(16)
	return randomString + timeString
}
