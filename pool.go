// Copyright 2015 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File pool.go contains code strictly related to the database, including
// setting up the database with given config and creating and managing a
// connection pool.

package zoom

import (
	"github.com/garyburd/redigo/redis"
	"time"
)

// Configuration contains various options. It should be created once
// and passed in to the Init function during application startup.
type Configuration struct {
	// Address to connect to. Default: "localhost:6379"
	Address string
	// Network to use. Default: "tcp"
	Network string
	// Database id to use (using SELECT). Default: 0
	Database int
	// Password for a password-protected redis database. If not empty,
	// every connection will use the AUTH command during initialization
	// to authenticate with the database. Default: ""
	Password string
}

// pool is a pool of redis connections
var pool *redis.Pool

// defaultConfiguration holds the default values for each config option
// if the zero value is provided in the input configuration, the value
// will fallback to the default value
var defaultConfiguration = Configuration{
	Address:  "localhost:6379",
	Network:  "tcp",
	Database: 0,
	Password: "",
}

// NewConn gets a connection from the connection pool and returns it.
// It can be used for directly interacting with the database. See
// http://godoc.org/github.com/garyburd/redigo/redis for full documentation
// on the redis.Conn type.
func NewConn() redis.Conn {
	return pool.Get()
}

// Init starts the Zoom library and creates a connection pool. It accepts
// a Configuration struct as an argument. Any zero values in the configuration
// will fallback to their default values. Init should be called once during
// application startup.
func Init(passedConfig *Configuration) {
	config := getConfiguration(passedConfig)
	pool = &redis.Pool{
		MaxIdle:     10,
		MaxActive:   0,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			// Connect to config.Address using config.Network
			c, err := redis.Dial(config.Network, config.Address)
			if err != nil {
				return nil, err
			}
			// If a password was provided, use the AUTH command to authenticate
			if config.Password != "" {
				_, err = c.Do("AUTH", config.Password)
				if err != nil {
					return nil, err
				}
			}
			// Select the database number provided by config.Database
			if _, err := c.Do("Select", config.Database); err != nil {
				c.Close()
				return nil, err
			}
			return c, err
		},
	}
}

// Close closes the connection pool and shuts down the Zoom library.
// It should be run when application exits, e.g. using defer.
func Close() {
	pool.Close()
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
	// since the zero value for string is "", we can skip config.Address
	return newConfig
}
