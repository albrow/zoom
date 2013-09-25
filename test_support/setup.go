// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// Package test_support contains various types and functions that
// are used to simplify test and benchmark execution. Not intended
// for external use.

// File setup.go contains helper functions SetUp() and TearDown()

package test_support

import (
	"fmt"
	"github.com/stephenalexbrowne/zoom"
	"github.com/stephenalexbrowne/zoom/redis"
	"math/rand"
	"time"
)

func SetUp() {

	conn := connect()
	defer conn.Close()

	// make sure database #9 is empty
	n, err := redis.Int(conn.Do("DBSIZE"))
	if err != nil {
		panic(err.Error())
	}
	if n != 0 {
		panic("Database #9 is not empty, test can not continue")
	}

	// register the types in types.go
	if err := zoom.Register(&Person{}, "person"); err != nil {
		panic(err.Error())
	}
	if err := zoom.Register(&ModelWithList{}, "modelWithList"); err != nil {
		panic(err.Error())
	}
	if err := zoom.Register(&ModelWithSet{}, "modelWithSet"); err != nil {
		panic(err.Error())
	}
	if err := zoom.Register(&Artist{}, "artist"); err != nil {
		panic(err.Error())
	}
	if err := zoom.Register(&Color{}, "color"); err != nil {
		panic(err.Error())
	}
	if err := zoom.Register(&PetOwner{}, "petOwner"); err != nil {
		panic(err.Error())
	}
	if err := zoom.Register(&Pet{}, "pet"); err != nil {
		panic(err.Error())
	}
	if err := zoom.Register(&Friend{}, "friend"); err != nil {
		panic(err.Error())
	}

	// generate a new seed for rand
	rand.Seed(time.Now().UTC().UnixNano())
}

func connect() redis.Conn {

	// initialize zoom
	dialUNIX()

	// if dialUNIX failed, try using a tcp connection
	conn := zoom.GetConn()
	if err := testConn(conn); err != nil {
		fmt.Println("WARNING: falling back to tcp connection. For maximum performance use a socket connection on /tmp/redis.sock")
		dialTCP()
		conn = zoom.GetConn()
		if err := testConn(conn); err != nil {
			// if dialTCP failed, panic.
			panic(err)
		}
	}

	// get a connection
	return conn
}

func dialUNIX() {
	zoom.Init(&zoom.Configuration{
		Address:  "/tmp/redis.sock",
		Network:  "unix",
		Database: 9,
	})
}

func dialTCP() {
	zoom.Init(&zoom.Configuration{
		Address:  "localhost:6379",
		Network:  "tcp",
		Database: 9,
	})
}

func testConn(conn redis.Conn) error {
	if _, err := conn.Do("PING"); err != nil {
		return err
	}
	return nil
}

func TearDown() {

	// unregister types in types.go
	zoom.UnregisterName("person")
	zoom.UnregisterName("modelWithList")
	zoom.UnregisterName("modelWithSet")
	zoom.UnregisterName("artist")
	zoom.UnregisterName("color")
	zoom.UnregisterName("petOwner")
	zoom.UnregisterName("pet")
	zoom.UnregisterName("friend")

	// flush and close the database
	conn := zoom.GetConn()
	_, err := conn.Do("flushdb")
	if err != nil {
		panic(err)
	}
	conn.Close()
	zoom.Close()
}
