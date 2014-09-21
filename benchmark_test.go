// Copyright 2014 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

package zoom

import (
	"testing"
)

// BenchmarkConnection times getting a connection and closing it
func BenchmarkConnection(b *testing.B) {
	testingSetUp()
	defer testingTearDown()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn := GetConn()
		conn.Close()
	}
}

// BenchmarkPing times the PING command
func BenchmarkPing(b *testing.B) {
	benchmarkCommand(b, nil, "PING")
}

// BenchmarkSet times the SET command
func BenchmarkSet(b *testing.B) {
	benchmarkCommand(b, nil, "SET", "foo", "bar")
}

// BenchmarkGet times the GET command
func BenchmarkGet(b *testing.B) {
	setup := func() {
		conn := GetConn()
		_, err := conn.Do("SET", "foo", "bar")
		if err != nil {
			b.Fatal(err)
		}
		conn.Close()
	}
	benchmarkCommand(b, setup, "GET", "foo")
}

// benchmarkCommand times a specific redis command. It calls setup
// then resets the timer before issuing cmd on each iteration.
func benchmarkCommand(b *testing.B, setup func(), cmd string, args ...interface{}) {
	testingSetUp()
	defer testingTearDown()
	if setup != nil {
		setup()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn := GetConn()
		if _, err := conn.Do(cmd, args...); err != nil {
			b.Fatal(err)
		}
		conn.Close()
	}
}

// BenchmarkSave times saving a single record
func BenchmarkSave(b *testing.B) {
	singleModelSelect := func(i int, models []*basicModel) *basicModel {
		return models[0]
	}
	benchmarkSave(b, 1, singleModelSelect)
}

// BenchmarkMSave100 times saving 100 models in a single transaction.
// NOTE: divide the reported time/op by 100 to get the time
// to save per model
func BenchmarkMSave100(b *testing.B) {
	testingSetUp()
	defer testingTearDown()
	// create 100 models
	ms, err := newBasicModels(100)
	if err != nil {
		b.Error(err)
	}
	b.ResetTimer()

	// save them all in one go
	models := Models(ms)
	for i := 0; i < b.N; i++ {
		MSave(models)
	}
}

// BenchmarkFindById times finding one model at a time randomly from
// a set of 1000 models
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

// BenchmarkMFindById times finding 100 models at a time selected randomly
// from a set of 10,000 models
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

// BenchmarkDeleteById times deleting a random model from
// a list of 10,000 models
func BenchmarkDeleteById(b *testing.B) {
	benchmarkDeleteById(b, 10000, randomIdSelect)
}

// BenchmarkMDeleteById times deleting 100 models at a time
// randomly selected from a set of 10,000 models
func BenchmarkMDeleteById100(b *testing.B) {
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

// BenchmarkFindAllQuery10 times finding all models from a set
// of 10 models using a query
func BenchmarkFindAllQuery10(b *testing.B) {
	benchmarkFindAllQuery(b, 10)
}

// BenchmarkFindAllQuery1000 times finding all models from a set
// of 1,000 models using a query
func BenchmarkFindAllQuery1000(b *testing.B) {
	benchmarkFindAllQuery(b, 1000)
}

// BenchmarkFindAllQuery100000 times finding all models from a set
// of 1,000 models using a query
func BenchmarkFindAllQuery100000(b *testing.B) {
	benchmarkFindAllQuery(b, 100000)
}

// BenchmarkCountAllQuery10 times counting 10 models
func BenchmarkCountAllQuery10(b *testing.B) {
	benchmarkCountAllQuery(b, 10)
}

// BenchmarkCountAllQuery1000 times counting 1,000 models
func BenchmarkCountAllQuery1000(b *testing.B) {
	benchmarkCountAllQuery(b, 1000)
}

// BenchmarkCountAllQuery100000 times counting 100,000 models
func BenchmarkCountAllQuery100000(b *testing.B) {
	benchmarkCountAllQuery(b, 100000)
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

func randomIdSelect(i int, ids []string) string {
	return ids[randInt(0, len(ids)-1)]
}

// selectRandomUniqueIds selects num random ids from the set of ids
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

// fillStringSlice fills a string slice with the num occurences of str and returns it
func fillStringSlice(num int, str string) []string {
	results := make([]string, num)
	for i := 0; i < num; i++ {
		results[i] = str
	}
	return results
}
