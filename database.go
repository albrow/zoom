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

var pool *redis.Pool

func GetConn() redis.Conn {
	return pool.Get()
}

// initializes a connection pool to be used to conect to database
// TODO: add some config options
func Init() {
	fmt.Println("zoom: creating connection pool...")
	pool = &redis.Pool{
		MaxIdle:     3,
		MaxActive:   0,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("unix", "/tmp/redis.sock")
			if err != nil {
				return nil, err
			}
			if _, err := c.Do("select", "7"); err != nil {
				c.Close()
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

// closes the connection pool
// Should be run when application exits
func Close() {
	pool.Close()
}

// Returns true iff a given key exists in redis
// If conn is nil, a connection will be created for you
// said connection will be closed before the end of the function
func KeyExists(key string, conn redis.Conn) (bool, error) {
	if conn == nil {
		conn = pool.Get()
		defer conn.Close()
	}
	return redis.Bool(conn.Do("exists", key))
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
