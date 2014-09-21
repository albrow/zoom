// Copyright 2014 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

package zoom

import (
	"testing"
)

// Just gets a connection and closes it
func BenchmarkConnection(b *testing.B) {

	testingSetUp()
	defer testingTearDown()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		conn := GetConn()
		conn.Close()
	}
}

// Gets a connection, calls PING, waits for response, and closes it
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

// Gets a connection, calls SET, waits for response, and closes it
func BenchmarkSet(b *testing.B) {

	checkReply := func(reply interface{}, err error) {
		if err != nil {
			b.Fatal(err)
		}
	}

	benchmarkCommand(b, nil, checkReply, "SET", "foo", "bar")
}

// Gets a connection, calls GET, waits for response, and closes it
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

// Saves the same record repeatedly
// (after the first save, nothing changes)
func BenchmarkSave(b *testing.B) {
	singleModelSelect := func(i int, models []*basicModel) *basicModel {
		return models[0]
	}
	benchmarkSave(b, 1, singleModelSelect)
}

// Saves 100 models in a single transaction using MSAVE
// NOTE: divide the reported time/op by 100 to get the time
// to save per model
func BenchmarkMSave100(b *testing.B) {
	testingSetUp()
	defer testingTearDown()

	ms, err := newBasicModels(100)
	if err != nil {
		b.Error(err)
	}
	b.ResetTimer()
	models := Models(ms)

	for i := 0; i < b.N; i++ {
		MSave(models)
	}
}

// Finds one model at a time randomly from a set of 1000 models
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
		index := randInt(0, len(ms)-1)
		id := ids[index]
		b.StartTimer()
		FindById("basicModel", id)
	}
}

// Finds 100 models at a time selected randomly from a set of 10,000 models
func BenchmarkMFindById100(b *testing.B) {
	testingSetUp()
	defer testingTearDown()

	// create some models
	ms, err := newBasicModels(10000)
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
		selectedIds := selectRandomUniqueIds(100, ids)
		modelNames := fillStringSlice(100, "basicModel")
		b.StartTimer()
		_, err := MFindById(modelNames, selectedIds)
		b.StopTimer()
		if err != nil {
			b.Error(err)
		}
	}
}

// Scans a random model selected from a set of 1000 models
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
		index := randInt(0, len(ms)-1)
		id := ids[index]
		mCopy := new(basicModel)
		b.StartTimer()
		ScanById(id, mCopy)
	}
}

// Scans 100 models at a time selected randomly from a set of 10,000 models
func BenchmarkMScanById100(b *testing.B) {
	testingSetUp()
	defer testingTearDown()

	// create some models
	ms, err := newBasicModels(10000)
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
		selectedIds := selectRandomUniqueIds(100, ids)
		models := make([]*basicModel, 100)
		b.StartTimer()
		err := MScanById(selectedIds, &models)
		b.StopTimer()
		if err != nil {
			b.Error(err)
		}
	}
}

// Repeatedly calls delete on a record
// (after the first, the record will have already been deleted)
func BenchmarkRepeatDeleteById(b *testing.B) {
	benchmarkDeleteById(b, 1, singleIdSelect)
}

// Randomly calls delete on a list of records
// repeated deletes have no effect
func BenchmarkRandomDeleteById(b *testing.B) {
	benchmarkDeleteById(b, 1000, randomIdSelect)
}

// Find all models from a set of 10 models using a query
func BenchmarkFindAllQuery1(b *testing.B) {
	benchmarkFindAllQuery(b, 10)
}

// Find all models from a set of 1,000 models using a query
func BenchmarkFindAllQuery1000(b *testing.B) {
	benchmarkFindAllQuery(b, 1000)
}

// Find all models from a set of 100,000 models using a query
func BenchmarkFindAllQuery100000(b *testing.B) {
	benchmarkFindAllQuery(b, 100000)
}

// Count all models from a set of 10 models using a query
func BenchmarkCountAllQuery1(b *testing.B) {
	benchmarkCountAllQuery(b, 10)
}

// Count all models from a set of 1,000 models using a query
func BenchmarkCountAllQuery1000(b *testing.B) {
	benchmarkCountAllQuery(b, 1000)
}

// Count all models from a set of 100,000 models using a query
func BenchmarkCountAllQuery100000(b *testing.B) {
	benchmarkCountAllQuery(b, 100000)
}

// Deletes 100 models at a time randomly selected from a set of 10,000 models
// repeated delets have no effect
func BenchmarkMDeleteById(b *testing.B) {
	testingSetUp()
	defer testingTearDown()

	// create some models
	ms, err := newBasicModels(10000)
	if err != nil {
		b.Error(err)
	}
	MSave(Models(ms))
	ids := make([]string, len(ms))
	for i, p := range ms {
		ids[i] = p.Id
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		selectedIds := selectRandomUniqueIds(100, ids)
		modelNames := fillStringSlice(100, "basicModel")
		b.StartTimer()
		MDeleteById(modelNames, selectedIds)
	}
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
	if err := MSave(Models(ms)); err != nil {
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

func benchmarkFindAllQuery(b *testing.B, num int) {
	testingSetUp()
	defer testingTearDown()

	ms, err := newBasicModels(num)
	if err != nil {
		b.Error(err)
	}
	if err := MSave(Models(ms)); err != nil {
		b.Error(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StartTimer()
		_, err := NewQuery("basicModel").Run()
		b.StopTimer()
		if err != nil {
			b.Error(err)
		}
	}
}

func benchmarkCountAllQuery(b *testing.B, num int) {
	testingSetUp()
	defer testingTearDown()

	ms, err := newBasicModels(num)
	if err != nil {
		b.Error(err)
	}
	if err := MSave(Models(ms)); err != nil {
		b.Error(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StartTimer()
		_, err := NewQuery("basicModel").Count()
		b.StopTimer()
		if err != nil {
			b.Error(err)
		}
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

// selects num random ids from the set of ids
func selectRandomUniqueIds(num int, ids []string) []string {
	selected := make(map[string]bool)
	for len(selected) < num {
		index := randInt(0, len(ids)-1)
		id := ids[index]
		if _, found := selected[id]; !found {
			selected[id] = true
		}
	}
	results := make([]string, 0)
	for key, _ := range selected {
		results = append(results, key)
	}
	return results
}

// fills a string slice with the num occurences of str and returns it
func fillStringSlice(num int, str string) []string {
	results := make([]string, num)
	for i := 0; i < num; i++ {
		results[i] = str
	}
	return results
}
