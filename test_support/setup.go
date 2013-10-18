// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File setup.go contains helper functions SetUp() and TearDown()

// Package test_support contains various types and functions that
// are used to simplify test and benchmark execution. Not intended
// for external use.
package test_support

import (
	"flag"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/stephenalexbrowne/zoom"
	"math/rand"
	"time"
)

var address *string = flag.String("address", "localhost:6379", "the address of a redis server to connect to")
var network *string = flag.String("network", "tcp", "the network to use for the database connection (e.g. 'tcp' or 'unix')")
var database *int = flag.Int("database", 9, "the redis database number to use for testing")

var types map[string]interface{} = map[string]interface{}{
	"person":                &Person{},
	"modelWithList":         &ModelWithList{},
	"modelWithSet":          &ModelWithSet{},
	"artist":                &Artist{},
	"color":                 &Color{},
	"petOwner":              &PetOwner{},
	"pet":                   &Pet{},
	"friend":                &Friend{},
	"primativeTypes":        &PrimativeTypes{},
	"pointerPrimativeTypes": &PointerPrimativeTypes{},
	"incovertibleTypes":     &InconvertibleTypes{},
	"embeddedStruct":        &EmbeddedStruct{},
	"pointerEmbeddedStruct": &PointerEmbeddedStruct{},
}

func SetUp() {
	conn := connect()
	defer conn.Close()

	registerTypes()

	// make sure database is empty
	n, err := redis.Int(conn.Do("DBSIZE"))
	if err != nil {
		panic(err.Error())
	}
	if n != 0 {
		msg := fmt.Sprintf("Database #%d is not empty, test can not continue", *database)
		panic(msg)
	}

	// generate a new seed for rand
	rand.Seed(time.Now().UTC().UnixNano())
}

// initialize zoom and test the connection
func connect() redis.Conn {
	dial()
	conn := zoom.GetConn()
	if err := testConn(conn); err != nil {
		panic(err)
	}
	return conn
}

func dial() {
	zoom.Init(&zoom.Configuration{
		Address:  *address,
		Network:  *network,
		Database: *database,
	})
}

func testConn(conn redis.Conn) error {
	if _, err := conn.Do("PING"); err != nil {
		return err
	}
	return nil
}

// register the types in types.go
func registerTypes() {
	for name, in := range types {
		if err := zoom.Register(in, name); err != nil {
			panic(err)
		}
	}
}

func TearDown() {
	unregisterTypes()

	// flush and close the database
	conn := zoom.GetConn()
	_, err := conn.Do("flushdb")
	if err != nil {
		panic(err)
	}
	conn.Close()
	zoom.Close()
}

// unregister types in types.go
func unregisterTypes() {
	for name, _ := range types {
		zoom.UnregisterName(name)
	}
}
