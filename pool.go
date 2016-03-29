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
	options PoolOptions
	// redisPool is a redis.Pool
	redisPool *redis.Pool
	// modelTypeToSpec maps a registered model type to a modelSpec
	modelTypeToSpec map[reflect.Type]*modelSpec
	// modelNameToSpec maps a registered model name to a modelSpec
	modelNameToSpec map[string]*modelSpec
}

// DefaultPoolOptions holds the default values for each pool option.
var DefaultPoolOptions = PoolOptions{
	Address:     "localhost:6379",
	Database:    0,
	IdleTimeout: 240 * time.Second,
	MaxActive:   1000,
	MaxIdle:     1000,
	Network:     "tcp",
	Password:    "",
	Wait:        true,
}

// PoolOptions contains various options for a pool.
type PoolOptions struct {
	// Address to connect to. Default: "localhost:6379"
	Address string
	// Database id to use (using SELECT). Default: 0
	Database int
	// IdleTimeout is the amount of time to wait before timing out (closing) idle
	// connections. Default: 240 * time.Second
	IdleTimeout time.Duration
	// MaxActive is the maximum number of active connections the pool will keep.
	// A value of 0 means unlimited. Default: 1000
	MaxActive int
	// MaxIdle is the maximum number of idle connections the pool will keep. A
	// value of 0 means unlimited. Default: 1000
	MaxIdle int
	// Network to use. Default: "tcp"
	Network string
	// Password for a password-protected redis database. If not empty,
	// every connection will use the AUTH command during initialization
	// to authenticate with the database. Default: ""
	Password string
	// Wait indicates whether or not the pool should wait for a free connection
	// if the MaxActive limit has been hit. If Wait is false and the MaxActive
	// limit is hit, Zoom will return an error indicating that the pool is
	// exhausted. Default: true
	Wait bool
}

// WithAddress returns a new copy of the options with the Address property set
// to the given value. It does not mutate the original options.
func (options PoolOptions) WithAddress(address string) PoolOptions {
	options.Address = address
	return options
}

// WithDatabase returns a new copy of the options with the Database property set
// to the given value. It does not mutate the original options.
func (options PoolOptions) WithDatabase(database int) PoolOptions {
	options.Database = database
	return options
}

// WithIdleTimeout returns a new copy of the options with the IdleTimeout
// property set to the given value. It does not mutate the original options.
func (options PoolOptions) WithIdleTimeout(timeout time.Duration) PoolOptions {
	options.IdleTimeout = timeout
	return options
}

// WithMaxActive returns a new copy of the options with the MaxActive property
// set to the given value. It does not mutate the original options.
func (options PoolOptions) WithMaxActive(maxActive int) PoolOptions {
	options.MaxActive = maxActive
	return options
}

// WithMaxIdle returns a new copy of the options with the MaxIdle property set
// to the given value. It does not mutate the original options.
func (options PoolOptions) WithMaxIdle(maxIdle int) PoolOptions {
	options.MaxIdle = maxIdle
	return options
}

// WithNetwork returns a new copy of the options with the Network property set
// to the given value. It does not mutate the original options.
func (options PoolOptions) WithNetwork(network string) PoolOptions {
	options.Network = network
	return options
}

// WithPassword returns a new copy of the options with the Password property set
// to the given value. It does not mutate the original options.
func (options PoolOptions) WithPassword(password string) PoolOptions {
	options.Password = password
	return options
}

// WithWait returns a new copy of the options with the Wait property set to the
// given value. It does not mutate the original options.
func (options PoolOptions) WithWait(wait bool) PoolOptions {
	options.Wait = wait
	return options
}

// NewPool creates and returns a new pool using the given address to connect to
// Redis. All the other options will be set to their default values, which can
// be found in DefaultPoolOptions.
func NewPool(address string) *Pool {
	return NewPoolWithOptions(DefaultPoolOptions.WithAddress(address))
}

// NewPoolWithOptions initializes and returns a pool with the given options. You
// can pass in DefaultOptions to use all the default options. Or cal the WithX
// methods of DefaultOptions to change the options you want to change.
func NewPoolWithOptions(options PoolOptions) *Pool {
	pool := &Pool{
		options:         options,
		modelTypeToSpec: map[reflect.Type]*modelSpec{},
		modelNameToSpec: map[string]*modelSpec{},
	}
	pool.redisPool = &redis.Pool{
		MaxIdle:     options.MaxIdle,
		MaxActive:   options.MaxActive,
		IdleTimeout: options.IdleTimeout,
		Wait:        options.Wait,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial(options.Network, options.Address)
			if err != nil {
				return nil, err
			}
			// If a options.Password was provided, use the AUTH command to authenticate
			if options.Password != "" {
				_, err = c.Do("AUTH", options.Password)
				if err != nil {
					return nil, err
				}
			}
			// Select the database number provided by options.Database
			if _, err := c.Do("Select", options.Database); err != nil {
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
// on the redis.Conn type. You must call Close on any connections after you are
// done using them. Failure to call Close can cause a resource leak.
func (p *Pool) NewConn() redis.Conn {
	return p.redisPool.Get()
}

// Close closes the pool. It should be run whenever the pool is no longer
// needed. It is often used in conjunction with defer.
func (p *Pool) Close() error {
	return p.redisPool.Close()
}
