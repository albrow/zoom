// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

package zoom

import (
	"testing"
)

// just get a connection and close it
func BenchmarkConnection(b *testing.B) {

	testingSetUp()
	defer testingTearDown()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		conn := GetConn()
		conn.Close()
	}
}

// get a connection, call PING, wait for response, and close it
func BenchmarkPing(b *testing.B) {

	checkReply := func(reply interface{}, err error) {
		if err != nil {
			b.Fatal(err)
		}
		str, ok := reply.(string)
		if !ok {
			b.Fatal("couldn't convert reply to string")
		}
		if str != "PONG" {
			b.Fatal("reply was not PONG: ", str)
		}
	}

	benchmarkCommand(b, nil, checkReply, "PING")
}

// get a connection, call SET, wait for response, and close it
func BenchmarkSet(b *testing.B) {

	checkReply := func(reply interface{}, err error) {
		if err != nil {
			b.Fatal(err)
		}
	}

	benchmarkCommand(b, nil, checkReply, "SET", "foo", "bar")
}

// get a connection, call GET, wait for response, and close it
func BenchmarkGet(b *testing.B) {

	setup := func() {
		conn := GetConn()
		_, err := conn.Do("SET", "foo", "bar")
		if err != nil {
			b.Fatal(err)
		}
		conn.Close()
	}

	checkReply := func(reply interface{}, err error) {
		if err != nil {
			b.Fatal(err)
		}
		byt, ok := reply.([]byte)
		if !ok {
			b.Fatal("couldn't convert reply to []byte: ", reply)
		}
		str := string(byt)
		if str != "bar" {
			b.Fatal("reply was not bar: ", str)
		}
	}

	benchmarkCommand(b, setup, checkReply, "GET", "foo")
}

// saves the same record repeatedly
// (after the first save, nothing changes)
func BenchmarkSave(b *testing.B) {
	singleModelSelect := func(i int, models []*basicModel) *basicModel {
		return models[0]
	}
	benchmarkSave(b, 1, singleModelSelect)
}

// finds the same record over and over
func BenchmarkFindById(b *testing.B) {
	testingSetUp()
	defer testingTearDown()

	// create some models
	ms, err := newBasicModels(1000)
	if err != nil {
		b.Error(err)
	}
	MSave(Models(ms))
	ids := make([]string, len(ms))
	for i, p := range ms {
		ids[i] = p.Id
	}

	// reset the timer
	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		id := ids[i%len(ids)]
		b.StartTimer()
		FindById("basicModel", id)
	}
}

// scans the same record over and over
func BenchmarkScanById(b *testing.B) {
	testingSetUp()
	defer testingTearDown()

	// create some models
	ms, err := newBasicModels(1000)
	if err != nil {
		b.Error(err)
	}
	MSave(Models(ms))
	ids := make([]string, len(ms))
	for i, p := range ms {
		ids[i] = p.Id
	}

	// reset the timer
	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		id := ids[i%len(ids)]
		mCopy := new(basicModel)
		b.StartTimer()
		ScanById(id, mCopy)
	}
}

// repeatedly calls delete on a record
// (after the first, the record will have already been deleted)
func BenchmarkRepeatDeleteById(b *testing.B) {
	benchmarkDeleteById(b, 1, singleIdSelect)
}

// randomly calls delete on a list of records
func BenchmarkRandomDeleteById(b *testing.B) {
	benchmarkDeleteById(b, 1000, randomIdSelect)
}

func benchmarkCommand(b *testing.B, setup func(), checkReply func(interface{}, error), cmd string, args ...interface{}) {
	testingSetUp()
	defer testingTearDown()

	if setup != nil {
		setup()
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		conn := GetConn()
		reply, err := conn.Do(cmd, args...)
		b.StopTimer()
		if checkReply != nil {
			checkReply(reply, err)
		}
		b.StartTimer()
		conn.Close()
	}
}

func benchmarkSave(b *testing.B, num int, modelSelect func(int, []*basicModel) *basicModel) {
	testingSetUp()
	defer testingTearDown()

	// create a sequence of models to be saved
	ms, err := newBasicModels(num)
	if err != nil {
		b.Error(err)
	}

	// reset the timer
	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		m := modelSelect(i, ms)
		b.StartTimer()
		err := Save(m)
		b.StopTimer()
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
	}
}

func benchmarkDeleteById(b *testing.B, num int, idSelect func(int, []string) string) {
	testingSetUp()
	defer testingTearDown()

	ms, err := newBasicModels(num)
	if err != nil {
		b.Error(err)
	}
	ids := make([]string, len(ms))
	for i, m := range ms {
		ids[i] = m.Id
	}

	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		id := idSelect(i, ids)
		b.StartTimer()
		DeleteById("basicModel", id)
	}
}

func benchmarkDelete(b *testing.B, num int, modelSelect func(int, []*basicModel) *basicModel) {
	testingSetUp()
	defer testingTearDown()

	ms, err := newBasicModels(num)
	if err != nil {
		b.Error(err)
	}

	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		m := modelSelect(i, ms)
		b.StartTimer()
		Delete(m)
	}
}

func singleIdSelect(i int, ids []string) string {
	return ids[0]
}

func sequentialIdSelect(i int, ids []string) string {
	return ids[i%len(ids)]
}

func randomIdSelect(i int, ids []string) string {
	return ids[randInt(0, len(ids)-1)]
}
