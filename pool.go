// Copyright 2015 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File pool.go contains code strictly related to the database, including
// setting up the database with given config and creating and managing a
// connection pool.

package zoom

import (
	"github.com/garyburd/redigo/redis"
	"reflect"
	"time"
)

// Pool represents a pool of connections. Each pool connects
// to one database. Each pool also manages its own set of
// registered models.
type Pool struct {
	// config is the fully parsed conifg, with defaults filling in any
	// blanks from the poolConfig passed into NewPool.
	config *PoolConfig
	// redisPool is a redis.Pool
	redisPool *redis.Pool
	// modelTypeToSpec maps a registered model type to a modelSpec
	modelTypeToSpec map[reflect.Type]*modelSpec
	// modelNameToSpec maps a registered model name to a modelSpec
	modelNameToSpec map[string]*modelSpec
}

// NewPool initializes and returns a pool with the given parameters
func NewPool(config *PoolConfig) *Pool {
	fullConfig := parseConfig(config)
	pool := &Pool{
		config:          fullConfig,
		modelTypeToSpec: map[reflect.Type]*modelSpec{},
		modelNameToSpec: map[string]*modelSpec{},
	}
	pool.redisPool = &redis.Pool{
		MaxIdle:     10,
		MaxActive:   0,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial(fullConfig.Network, fullConfig.Address)
			if err != nil {
				return nil, err
			}
			// If a config.Password was provided, use the AUTH command to authenticate
			if fullConfig.Password != "" {
				_, err = c.Do("AUTH", fullConfig.Password)
				if err != nil {
					return nil, err
				}
			}
			// Select the database number provided by fullConfig.Database
			if _, err := c.Do("Select", fullConfig.Database); err != nil {
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

// Close closes the connection pool. It should be run when application
// exits, e.g. using defer.
func (p *Pool) Close() error {
	return p.redisPool.Close()
}

// defaultPoolConfig holds the default values for each config option
// if the zero value is provided in the input configuration, the value
// will fallback to the default value
var defaultPoolConfig = PoolConfig{
	Address:  "localhost:6379",
	Network:  "tcp",
	Database: 0,
	Password: "",
}

// parseConfig returns a well-formed PoolConfig struct.
// If the passedConfig is nil, returns defaultPoolConfig.
// Else, for each zero value field in passedConfig,
// use the default value for that field.
func parseConfig(passedConfig *PoolConfig) *PoolConfig {
	if passedConfig == nil {
		return &defaultPoolConfig
	}
	// copy the passedConfig
	newConfig := *passedConfig
	if newConfig.Address == "" {
		newConfig.Address = defaultPoolConfig.Address
	}
	if newConfig.Network == "" {
		newConfig.Network = defaultPoolConfig.Network
	}
	// since the zero value for int is 0, we can skip config.Database
	// since the zero value for string is "", we can skip config.Address
	return &newConfig
}

// PoolConfig contains various options for a pool.
type PoolConfig struct {
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
