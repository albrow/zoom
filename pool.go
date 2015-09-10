// Copyright 2015 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File pool.go contains code strictly related to the database, including
// setting up the database with given config and creating and managing a
// connection pool.

package zoom

import (
	"reflect"
	"time"

	"github.com/garyburd/redigo/redis"
)

// Pool represents a pool of connections. Each pool connects
// to one database and manages its own set of registered models.
type Pool struct {
	// options is the fully parsed conifg, with defaults filling in any
	// blanks from the poolConfig passed into NewPool.
	options *PoolOptions
	// redisPool is a redis.Pool
	redisPool *redis.Pool
	// modelTypeToSpec maps a registered model type to a modelSpec
	modelTypeToSpec map[reflect.Type]*modelSpec
	// modelNameToSpec maps a registered model name to a modelSpec
	modelNameToSpec map[string]*modelSpec
}

// PoolOptions contains various options for a pool.
type PoolOptions struct {
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

// NewPool initializes and returns a pool with the given options. To use all
// the default options, you can pass in nil.
func NewPool(options *PoolOptions) *Pool {
	fullOptions := parsePoolOptions(options)
	pool := &Pool{
		options:         fullOptions,
		modelTypeToSpec: map[reflect.Type]*modelSpec{},
		modelNameToSpec: map[string]*modelSpec{},
	}
	pool.redisPool = &redis.Pool{
		MaxIdle:     10,
		MaxActive:   0,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial(fullOptions.Network, fullOptions.Address)
			if err != nil {
				return nil, err
			}
			// If a options.Password was provided, use the AUTH command to authenticate
			if fullOptions.Password != "" {
				_, err = c.Do("AUTH", fullOptions.Password)
				if err != nil {
					return nil, err
				}
			}
			// Select the database number provided by fullOptions.Database
			if _, err := c.Do("Select", fullOptions.Database); err != nil {
				c.Close()
				return nil, err
			}
			return c, err
		},
	}
	return pool
}

// NewConn gets a connection from the pool and returns it.
// It can be used for directly interacting with the database. See
// http://godoc.org/github.com/garyburd/redigo/redis for full documentation
// on the redis.Conn type.
func (p *Pool) NewConn() redis.Conn {
	return p.redisPool.Get()
}

// Close closes the pool. It should be run whenever the pool is no longer
// needed. It is often used in conjunction with defer.
func (p *Pool) Close() error {
	return p.redisPool.Close()
}

// defaultPoolOptions holds the default values for each config option
// if the zero value is provided in the input configuration, the value
// will fallback to the default value
var defaultPoolOptions = PoolOptions{
	Address:  "localhost:6379",
	Network:  "tcp",
	Database: 0,
	Password: "",
}

// parsePoolOptions returns a well-formed PoolOptions struct.
// If the passedOptions is nil, returns defaultPoolOptions.
// Else, for each zero value field in passedOptions,
// use the default value for that field.
func parsePoolOptions(passedOptions *PoolOptions) *PoolOptions {
	if passedOptions == nil {
		return &defaultPoolOptions
	}
	// copy the passedOptions
	newOptions := *passedOptions
	if newOptions.Address == "" {
		newOptions.Address = defaultPoolOptions.Address
	}
	if newOptions.Network == "" {
		newOptions.Network = defaultPoolOptions.Network
	}
	// since the zero value for int is 0, we can skip config.Database
	// since the zero value for string is "", we can skip config.Address
	return &newOptions
}
