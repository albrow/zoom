package zoom

// File contains code strictly related to the database

import (
	"code.google.com/p/tcgl/redis"
	"fmt"
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
