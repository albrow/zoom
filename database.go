// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File database.go contains code strictly related to the database, including
// setting up the database with given config, generating unique,
// random ids, and creating and managing a connection pool. There
// are also convenience functions for (e.g.) checking if a key exists
// in redis.

package zoom

import (
	"github.com/dchest/uniuri"
	"github.com/garyburd/redigo/redis"
	"strconv"
	"time"
)

// Configuration contains various options. It should be created once
// and passed in to the Init function during application startup.
type Configuration struct {
	Address  string // Address to connect to. Default: "localhost:6379"
	Network  string // Network to use. Default: "tcp"
	Database int    // Database id to use (using SELECT). Default: 0
}

var pool *redis.Pool

var defaultConfiguration = Configuration{
	Address:  "localhost:6379",
	Network:  "tcp",
	Database: 0,
}

// GetConn gets a connection from the connection pool and returns it.
// It can be used for directly interacting with the database. Check out
// http://godoc.org/github.com/garyburd/redigo/redis for full documentation
// on the redis.Conn type.
func GetConn() redis.Conn {
	return pool.Get()
}

// Init starts the Zoom library and creates a connection pool. It accepts
// a Configuration struct as an argument. Any zero values in the configuration
// will fallback to their default values. Init should be called once during
// application startup.
func Init(passedConfig *Configuration) {
	// compile all the model specs
	if err := compileModelSpecs(); err != nil {
		panic(err)
	}

	config := getConfiguration(passedConfig)
	pool = &redis.Pool{
		MaxIdle:     10,
		MaxActive:   0,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial(config.Network, config.Address)
			if err != nil {
				return nil, err
			}
			if _, err := c.Do("select", strconv.Itoa(config.Database)); err != nil {
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

// Close closes the connection pool and shuts down the Zoom library.
// It should be run when application exits, e.g. using defer.
func Close() {
	pool.Close()
}

// KeyExists returns true iff a given key exists in redis.
// If conn is nil, a new connection will be created and
// closed before the end of the function.
func KeyExists(key string, conn redis.Conn) (bool, error) {
	if conn == nil {
		conn = pool.Get()
		defer conn.Close()
	}
	return redis.Bool(conn.Do("exists", key))
}

// SetContains returns true iff the redis set identified by key contains
// member.  If conn is nil, a new connection will be created and
// closed before the end of the function.
func SetContains(key, member string, conn redis.Conn) (bool, error) {
	if conn == nil {
		conn = pool.Get()
		defer conn.Close()
	}
	return redis.Bool(conn.Do("sismember", key, member))
}

// generateRandomId generates a random string that is more or less
// garunteed to be unique. Used as Ids for records where an Id is
// not otherwise provided.
func generateRandomId() string {
	timeInt := time.Now().Unix()
	timeString := strconv.FormatInt(timeInt, 36)
	randomString := uniuri.NewLen(16)
	return randomString + timeString
}

// getConfiguration returns a well-formed configuration struct.
// If the passedConfig is nil, returns defaultConfiguration.
// Else, for each zero value field in passedConfig,
// use the default value for that field.
func getConfiguration(passedConfig *Configuration) Configuration {
	if passedConfig == nil {
		return defaultConfiguration
	}

	// copy the passedConfig
	newConfig := *passedConfig

	if newConfig.Address == "" {
		newConfig.Address = defaultConfiguration.Address
	}
	if newConfig.Network == "" {
		newConfig.Network = defaultConfiguration.Network
	}
	// since the zero value for int is 0, we can skip config.Database

	return newConfig
}
