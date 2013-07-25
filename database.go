package zoom

// File contains code strictly related to the database, including
// establishing a connection, instantiating a package-wide db var
// and closing the connection. There are also convenience functions
// for (e.g.) checking if a key exists in redis.

import (
	"fmt"
	"github.com/dchest/uniuri"
	"github.com/stephenalexbrowne/go-redis"
	"strconv"
	"time"
)

type DbConfig redis.Configuration

var db *redis.Database

// initializes and returns the database connection
func InitDb(config DbConfig) *redis.Database {
	if db != nil {
		return db
	} else {
		fmt.Println("zoom: connecting to database...")
		db = redis.Connect(redis.Configuration(config))
		return db
	}
}

// closes the connection to the database
func CloseDb() {
	if db != nil {
		fmt.Println("zoom: closing database connection...")
		db.Close()
	}
}

// Returns true iff a given key exists in redis
func keyExists(key string) (bool, error) {
	result := db.Command("exists", key)
	if result.Error() != nil {
		return false, result.Error()
	}

	return result.ValueAsBool()
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
