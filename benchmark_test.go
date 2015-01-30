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
	models := Models(ms)
	b.ResetTimer()

	// save them all in one go
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

// BenchmarkFilterIntQuery1From1 times a query which selects 1
// model out of 1 total, filtering by the Int field
func BenchmarkFilterIntQuery1From1(b *testing.B) {
	benchmarkFilterIntQuery(b, 1, 1)
}

// BenchmarkFilterIntQuery1From10 times a query which selects 1
// model out of 10 total, filtering by the Int field
func BenchmarkFilterIntQuery1From10(b *testing.B) {
	benchmarkFilterIntQuery(b, 1, 10)
}

// BenchmarkFilterIntQuery10From100 times a query which selects 10
// models out of 100 total, filtering by the Int field
func BenchmarkFilterIntQuery10From100(b *testing.B) {
	benchmarkFilterIntQuery(b, 10, 100)
}

// BenchmarkFilterIntQuery100From1000 times a query which selects 100
// models out of 1000 total, filtering by the Int field
func BenchmarkFilterIntQuery100From1000(b *testing.B) {
	benchmarkFilterIntQuery(b, 100, 1000)
}

//  BenchmarkFilterStringQuery1From1 times a query which selects
// 1 model out of 1 models total, filtering by the String field
func BenchmarkFilterStringQuery1From1(b *testing.B) {
	benchmarkFilterStringQuery(b, 1, 1)
}

//  BenchmarkFilterStringQuery1From10 times a query which selects
// 1 model out of 10 models total, filtering by the String field
func BenchmarkFilterStringQuery1From10(b *testing.B) {
	benchmarkFilterStringQuery(b, 1, 10)
}

//  BenchmarkFilterStringQuery10From100 times a query which selects
// 10 models out of 100 models total, filtering by the String field
func BenchmarkFilterStringQuery10From100(b *testing.B) {
	benchmarkFilterStringQuery(b, 10, 100)
}

//  BenchmarkFilterStringQuery100From1000 times a query which selects
// 100 models out of 1,000 models total, filtering by the String field
func BenchmarkFilterStringQuery100From1000(b *testing.B) {
	benchmarkFilterStringQuery(b, 100, 1000)
}

// BenchmarkFilterBoolQuery1From1 times a query which selects
// 1 model out of 1 models total, filtering by the Bool field
func BenchmarkFilterBoolQuery1From1(b *testing.B) {
	benchmarkFilterBoolQuery(b, 1, 1)
}

// BenchmarkFilterBoolQuery1From10 times a query which selects
// 1 model out of 10 models total, filtering by the Bool field
func BenchmarkFilterBoolQuery1From10(b *testing.B) {
	benchmarkFilterBoolQuery(b, 1, 10)
}

// BenchmarkFilterBoolQuery10From100 times a query which selects
// 10 models out of 100 models total, filtering by the Bool field
func BenchmarkFilterBoolQuery10From100(b *testing.B) {
	benchmarkFilterBoolQuery(b, 10, 100)
}

// BenchmarkFilterBoolQuery100From1000 times a query which selects
// 100 models out of 1000 models total, filtering by the Bool field
func BenchmarkFilterBoolQuery100From1000(b *testing.B) {
	benchmarkFilterBoolQuery(b, 100, 1000)
}

// BenchmarkOrderInt1000 times a query which selects 1,000
// models, ordered by the Int field
func BenchmarkOrderInt1000(b *testing.B) {
	testingSetUp()
	defer testingTearDown()

	// create a sequence of models to be saved
	ms, err := newIndexedPrimativesModels(1000)
	if err != nil {
		b.Error(err)
	}
	if err := MSave(Models(ms)); err != nil {
		b.Error(err)
		b.FailNow()
	}

	q := NewQuery("indexedPrimativesModel").Order("Int").Include("Int")
	benchmarkQuery(b, q)
}

// BenchmarkOrderString1000 times a query which selects 1,000
// models, ordered by the String field
func BenchmarkOrderString1000(b *testing.B) {
	testingSetUp()
	defer testingTearDown()

	// create a sequence of models to be saved
	ms, err := newIndexedPrimativesModels(1000)
	if err != nil {
		b.Error(err)
	}
	if err := MSave(Models(ms)); err != nil {
		b.Error(err)
		b.FailNow()
	}

	q := NewQuery("indexedPrimativesModel").Order("String").Include("String")
	benchmarkQuery(b, q)
}

// BenchmarkOrderBool1000 times a query which selects 1,000
// models, ordered by the Bool field
func BenchmarkOrderBool1000(b *testing.B) {
	testingSetUp()
	defer testingTearDown()

	// create a sequence of models to be saved
	ms, err := newIndexedPrimativesModels(1000)
	if err != nil {
		b.Error(err)
	}
	if err := MSave(Models(ms)); err != nil {
		b.Error(err)
		b.FailNow()
	}

	q := NewQuery("indexedPrimativesModel").Order("Bool").Include("Bool")
	benchmarkQuery(b, q)
}

// BenchmarkComplexQuery times a query which incorporates nearly all options.
// The query has a filter on the Bool and Int Fields, is ordered in reverse
// by the String field, and includes only Bool, Int, and String. Out of 1,000
// models created, 100 should fit the query criteria, but the query limits the
// number of results to 10.
func BenchmarkComplexQuery(b *testing.B) {
	testingSetUp()
	defer testingTearDown()

	// create a sequence of models to be saved
	ms, err := newIndexedPrimativesModels(1000)
	if err != nil {
		b.Error(err)
	}
	// give some models a searchable Int field
	for _, m := range ms[100:200] {
		m.Int = -1
	}
	// give some models a Bool attr of true and
	// all others a Bool attribute of false
	for _, m := range ms {
		m.Bool = false
	}
	for _, m := range ms[150:250] {
		m.Bool = true
	}

	if err := MSave(Models(ms)); err != nil {
		b.Error(err)
		b.FailNow()
	}

	q := NewQuery("indexedPrimativesModel").Filter("Int =", -1).Filter("Bool =", true).Order("-String").Include("Int", "String", "Bool").Limit(10).Offset(10)
	benchmarkQuery(b, q)
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

func benchmarkQuery(b *testing.B, q *Query) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		_, err := q.Run()
		b.StopTimer()
		if err != nil {
			b.Error(err)
		}
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
	benchmarkQuery(b, NewQuery("basicModel"))
}

func benchmarkFilterIntQuery(b *testing.B, selected int, total int) {
	testingSetUp()
	defer testingTearDown()

	ms, err := newIndexedPrimativesModels(total)
	if err != nil {
		b.Error(err)
	}
	for i := 0; i < selected; i++ {
		ms[i].Int = -1
	}
	if err := MSave(Models(ms)); err != nil {
		b.Error(err)
	}
	benchmarkQuery(b, NewQuery("indexedPrimativesModel").Filter("Int =", -1).Include("Int"))
}

func benchmarkFilterStringQuery(b *testing.B, selected int, total int) {
	testingSetUp()
	defer testingTearDown()

	ms, err := newIndexedPrimativesModels(total)
	if err != nil {
		b.Error(err)
	}
	for i := 0; i < selected; i++ {
		ms[i].String = "findMe"
	}
	if err := MSave(Models(ms)); err != nil {
		b.Error(err)
	}
	benchmarkQuery(b, NewQuery("indexedPrimativesModel").Filter("String =", "findMe").Include("String"))
}

func benchmarkFilterBoolQuery(b *testing.B, selected int, total int) {
	testingSetUp()
	defer testingTearDown()

	ms, err := newIndexedPrimativesModels(total)
	if err != nil {
		b.Error(err)
	}
	for i := 0; i < selected; i++ {
		ms[i].Bool = true
	}
	for i := selected; i < total; i++ {
		ms[i].Bool = false
	}
	if err := MSave(Models(ms)); err != nil {
		b.Error(err)
	}
	benchmarkQuery(b, NewQuery("indexedPrimativesModel").Filter("Bool =", true).Include("Bool"))
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
