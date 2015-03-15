// Copyright 2014 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

package zoom

import (
	"flag"
	"sync"
)

var (
	address  *string = flag.String("address", "localhost:6379", "the address of a redis server to connect to")
	network  *string = flag.String("network", "tcp", "the network to use for the database connection (e.g. 'tcp' or 'unix')")
	database *int    = flag.Int("database", 9, "the redis database number to use for testing")
)

var setUpOnce = sync.Once{}

func testingSetUp() {
	setUpOnce.Do(func() {
		// Init(&Configuration{
		// 	Address:  *address,
		// 	Network:  *network,
		// 	Database: *database,
		// })
		// checkDatabaseEmpty()
		registerTestingTypes()
	})
}

type testModel struct {
	Int    int
	String string
	Bool   bool
	DefaultData
}

type indexedTestModel struct {
	Int    int    `zoom:"index"`
	String string `zoom:"index"`
	Bool   bool   `zoom:"index"`
	DefaultData
}

var (
	testModels        *ModelType
	indexedTestModels *ModelType
)

func registerTestingTypes() {
	testModels := []struct {
		modelType *ModelType
		model     Model
	}{
		{
			modelType: testModels,
			model:     &testModel{},
		},
		{
			modelType: indexedTestModels,
			model:     &indexedTestModel{},
		},
	}
	for _, m := range testModels {
		mType, err := Register(m.model)
		if err != nil {
			panic(err)
		}
		m.modelType = mType
	}
}

func checkDatabaseEmpty() {
	// conn := GetConn()
	// defer conn.Close()
	// n, err := redis.Int(conn.Do("DBSIZE"))
	// if err != nil {
	// 	panic(err.Error())
	// }
	// if n != 0 {
	// 	err := fmt.Errorf("Database #%d is not empty! Testing can not continue.", *database)
	// 	panic(err)
	// }
}

func testingTearDown() {
	// // flush and close the database
	// conn := GetConn()
	// _, err := conn.Do("flushdb")
	// if err != nil {
	// 	panic(err)
	// }
	// conn.Close()
}
