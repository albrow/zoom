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

// pool is a pool of redis connections
var pool *redis.Pool

// initPool initializes the pool with the given parameters
func initPool(network string, address string, database int, password string) {
	pool = &redis.Pool{
		MaxIdle:     10,
		MaxActive:   0,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			// Connect to config.Address using config.Network
			c, err := redis.Dial(network, address)
			if err != nil {
				return nil, err
			}
			// If a password was provided, use the AUTH command to authenticate
			if password != "" {
				_, err = c.Do("AUTH", password)
				if err != nil {
					return nil, err
				}
			}
			// Select the database number provided by config.Database
			if _, err := c.Do("Select", database); err != nil {
				c.Close()
				return nil, err
			}
			return c, err
		},
	}
}

// NewConn gets a connection from the connection pool and returns it.
// It can be used for directly interacting with the database. See
// http://godoc.org/github.com/garyburd/redigo/redis for full documentation
// on the redis.Conn type.
func NewConn() redis.Conn {
	return pool.Get()
}
