package zoom

// File contains code strictly related to the database, including
// setting up the database with given config, generating unique,
// random ids, and creating and managing a connection pool. There
// are also convenience functions for (e.g.) checking if a key exists
// in redis.

import (
	"github.com/dchest/uniuri"
	"github.com/stephenalexbrowne/zoom/cache"
	"github.com/stephenalexbrowne/zoom/redis"
	"strconv"
	"time"
)

type Configuration struct {
	Address       string // Address to connect to. Default: "localhost:6379"
	Network       string // Network to use. Default: "tcp"
	Database      int    // Database id to use (using SELECT). Default: 0
	CacheCapacity uint64 // Size of the cache in bytes. Default: 67108864 (64 MB)
	CacheDisabled bool   // If true, cache will be disabled. Default: false
}

var pool *redis.Pool

var defaultConfiguration = Configuration{
	Address:       "localhost:6379",
	Network:       "tcp",
	Database:      0,
	CacheCapacity: 67108864,
	CacheDisabled: false,
}

func GetConn() redis.Conn {
	return pool.Get()
}

// initializes a connection pool to be used to conect to database
// TODO: add some config options
func Init(passedConfig *Configuration) {
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

	zoomCache = cache.NewLRUCache(config.CacheCapacity)
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

// Returns true iff the redis set identified by key contains member.
// If conn is nil, a connection will be created for you.
// said connection will be closed before the end of the function.
func SetContains(key, member string, conn redis.Conn) (bool, error) {
	if conn == nil {
		conn = pool.Get()
		defer conn.Close()
	}
	return redis.Bool(conn.Do("sismember", key, member))
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

// adds value as a member of a redis set identified by {name}:index
// where {name} is the name of the model you want to index.
// If the conn paramater is nil, will get a connection from the
// pool and close it before returning. If conn is a redis.Conn,
// it will use the existing connection.
func addToIndex(name, value string, conn redis.Conn) error {
	if conn == nil {
		conn = pool.Get()
		defer conn.Close()
	}
	key := name + ":index"
	_, err := conn.Do("sadd", key, value)
	return err
}

// return a proper configuration struct.
// if the passed in struct is nil, return defaultConfiguration
// else, for each attribute, if the passed in struct is "", 0, etc,
// use the default value for that attribute.
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
	if newConfig.CacheCapacity == 0 {
		newConfig.CacheCapacity = defaultConfiguration.CacheCapacity
	}
	// if cache is disabled, force cache capacity to 0
	if newConfig.CacheDisabled == true {
		newConfig.CacheCapacity = 0
	}
	// since the zero value for int is 0, we can skip config.Database

	return newConfig
}
