// Copyright 2015 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File benchmark_test.go contains all benchmarks.

package zoom

import (
	"math/rand"
	"testing"
)

// BenchmarkConnection just gets a connection and then closes it
func BenchmarkConnection(b *testing.B) {
	testingSetUp()
	defer testingTearDown()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn := testPool.NewConn()
		conn.Close()
	}
}

// BenchmarkPing sends the PING command
func BenchmarkPing(b *testing.B) {
	benchmarkCommand(b, "PING")
}

// BenchmarkSet sends the SET command
func BenchmarkSet(b *testing.B) {
	benchmarkCommand(b, "SET", "foo", "bar")
}

// BenchmarkGet sends the GET command after first sending SET
func BenchmarkGet(b *testing.B) {
	conn := testPool.NewConn()
	defer conn.Close()
	_, err := conn.Do("SET", "foo", "bar")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	benchmarkCommand(b, "GET", "foo")
}

// benchmarkCommand benchmarks a specific redis command.
func benchmarkCommand(b *testing.B, cmd string, args ...interface{}) {
	testingSetUp()
	defer testingTearDown()
	for i := 0; i < b.N; i++ {
		conn := testPool.NewConn()
		if _, err := conn.Do(cmd, args...); err != nil {
			conn.Close()
			b.Fatal(err)
		}
		conn.Close()
	}
}

// BenchmarkSave saves a single model
func BenchmarkSave(b *testing.B) {
	testingSetUp()
	defer testingTearDown()

	models := createTestModels(1)
	model := models[0]
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		testModels.Save(model)
	}
}

// BenchmarkSave100 saves 100 models in a single transaction.
func BenchmarkSave100(b *testing.B) {
	testingSetUp()
	defer testingTearDown()

	models := createTestModels(100)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		t := testPool.NewTransaction()
		for _, model := range models {
			t.Save(testModels, model)
		}
		b.StartTimer()
		if err := t.Exec(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFind finds one model at a time randomly from
// a set of 1,000 models
func BenchmarkFind(b *testing.B) {
	testingSetUp()
	defer testingTearDown()

	models, err := createAndSaveTestModels(1000)
	if err != nil {
		b.Fatal(err)
	}
	ids := modelIds(Models(models))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		id := selectUnique(1, ids)[0]
		b.StartTimer()
		testModels.Find(id, &testModel{})
	}
}

// BenchmarkFind100 finds 100 models at a time (in a single transaction)
// selected randomly from a set of 1,000 models
func BenchmarkFind100(b *testing.B) {
	testingSetUp()
	defer testingTearDown()

	models, err := createAndSaveTestModels(1000)
	if err != nil {
		b.Fatal(err)
	}
	ids := modelIds(Models(models))
	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		selectedIds := selectUnique(100, ids)
		t := testPool.NewTransaction()
		for _, id := range selectedIds {
			t.Find(testModels, id, &testModel{})
		}
		b.StartTimer()
		if err := t.Exec(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFindAll100 finds 100 models with the FindAll command
func BenchmarkFindAll100(b *testing.B) {
	testingSetUp()
	defer testingTearDown()

	_, err := createAndSaveTestModels(100)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := testModels.FindAll(&[]*testModel{}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFindAll10000 finds 10,000 models with the FindAll command
func BenchmarkFindAll10000(b *testing.B) {
	testingSetUp()
	defer testingTearDown()

	_, err := createAndSaveTestModels(10000)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := testModels.FindAll(&[]*testModel{}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDelete deletes a single model. It recreates the model on
// each iteration, but stops the timer during the Save method, so only
// Delete is timed.
func BenchmarkDelete(b *testing.B) {
	testingSetUp()
	defer testingTearDown()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Stop the timer, save a model, then start the timer again
		models, err := createAndSaveTestModels(1)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		if _, err := testModels.Delete(models[0].ModelId()); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDelete deletes 100 models in a single transaction. It
// recreates the models on each iteration, but stops the timer during
// the Save methods, so only Delete is timed.
func BenchmarkDelete100(b *testing.B) {
	testingSetUp()
	defer testingTearDown()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Stop the timer, save the models, then start the timer again
		models, err := createAndSaveTestModels(100)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		t := testPool.NewTransaction()
		for _, model := range models {
			deleted := false
			t.Delete(testModels, model.ModelId(), &deleted)
		}
		if err := t.Exec(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDeleteAll100 deletes 100 models with the DeleteAll command.
// It recreates the models on each iteration, but stops the timer during
// the Save methods, so only DeleteAll is timed.
func BenchmarkDeleteAll100(b *testing.B) {
	testingSetUp()
	defer testingTearDown()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		_, err := createAndSaveTestModels(100)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		if _, err := testModels.DeleteAll(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDeleteAll1000 deletes 1,000 models with the DeleteAll command
func BenchmarkDeleteAll1000(b *testing.B) {
	testingSetUp()
	defer testingTearDown()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		_, err := createAndSaveTestModels(1000)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		if _, err := testModels.DeleteAll(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCount100 counts 100 models with the Count command
func BenchmarkCount100(b *testing.B) {
	testingSetUp()
	defer testingTearDown()

	_, err := createAndSaveTestModels(100)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := testModels.Count(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCount10000 counts 10,000 models with the Count command
func BenchmarkCount10000(b *testing.B) {
	testingSetUp()
	defer testingTearDown()

	_, err := createAndSaveTestModels(10000)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := testModels.Count(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkQueryFilterInt1From1 runs a query which selects 1
// model out of 1 total, filtering by the Int field
func BenchmarkQueryFilterInt1From1(b *testing.B) {
	benchmarkQueryFilterInt(b, 1, 1)
}

// BenchmarkQueryFilterInt1From10 runs a query which selects 1
// model out of 10 total, filtering by the Int field
func BenchmarkQueryFilterInt1From10(b *testing.B) {
	benchmarkQueryFilterInt(b, 1, 10)
}

// BenchmarkQueryFilterInt10From100 runs a query which selects 10
// models out of 100 total, filtering by the Int field
func BenchmarkQueryFilterInt10From100(b *testing.B) {
	benchmarkQueryFilterInt(b, 10, 100)
}

// BenchmarkQueryFilterInt100From1000 runs a query which selects 100
// models out of 1000 total, filtering by the Int field
func BenchmarkQueryFilterInt100From1000(b *testing.B) {
	benchmarkQueryFilterInt(b, 100, 1000)
}

//  BenchmarkQueryFilterString1From1 runs a query which selects
// 1 model out of 1 models total, filtering by the String field
func BenchmarkQueryFilterString1From1(b *testing.B) {
	benchmarkQueryFilterString(b, 1, 1)
}

//  BenchmarkQueryFilterString1From10 runs a query which selects
// 1 model out of 10 models total, filtering by the String field
func BenchmarkQueryFilterString1From10(b *testing.B) {
	benchmarkQueryFilterString(b, 1, 10)
}

//  BenchmarkQueryFilterString10From100 runs a query which selects
// 10 models out of 100 models total, filtering by the String field
func BenchmarkQueryFilterString10From100(b *testing.B) {
	benchmarkQueryFilterString(b, 10, 100)
}

//  BenchmarkQueryFilterString100From1000 runs a query which selects
// 100 models out of 1,000 models total, filtering by the String field
func BenchmarkQueryFilterString100From1000(b *testing.B) {
	benchmarkQueryFilterString(b, 100, 1000)
}

// BenchmarkQueryFilterBool1From1 runs a query which selects
// 1 model out of 1 models total, filtering by the Bool field
func BenchmarkQueryFilterBool1From1(b *testing.B) {
	benchmarkQueryFilterBool(b, 1, 1)
}

// BenchmarkQueryFilterBool1From10 runs a query which selects
// 1 model out of 10 models total, filtering by the Bool field
func BenchmarkQueryFilterBool1From10(b *testing.B) {
	benchmarkQueryFilterBool(b, 1, 10)
}

// BenchmarkQueryFilterBool10From100 runs a query which selects
// 10 models out of 100 models total, filtering by the Bool field
func BenchmarkQueryFilterBool10From100(b *testing.B) {
	benchmarkQueryFilterBool(b, 10, 100)
}

// BenchmarkQueryFilterBool100From1000 runs a query which selects
// 100 models out of 1000 models total, filtering by the Bool field
func BenchmarkQueryFilterBool100From1000(b *testing.B) {
	benchmarkQueryFilterBool(b, 100, 1000)
}

// BenchmarkQueryOrderInt100 runs a query which finds all 100 models ordered
// by the Int field
func BenchmarkQueryOrderInt100(b *testing.B) {
	benchmarkQueryOrder(b, 100, "Int")
}

// BenchmarkQueryOrderInt10000 runs a query which finds all 10,000 models
// ordered by the Int field
func BenchmarkQueryOrderInt10000(b *testing.B) {
	benchmarkQueryOrder(b, 10000, "Int")
}

// BenchmarkQueryOrderString100 runs a query which finds all 100 models ordered
// by the String field
func BenchmarkQueryOrderString100(b *testing.B) {
	benchmarkQueryOrder(b, 100, "String")
}

// BenchmarkQueryOrderString10000 runs a query which finds all 10,000 models
// ordered by the String field
func BenchmarkQueryOrderString10000(b *testing.B) {
	benchmarkQueryOrder(b, 10000, "String")
}

// BenchmarkQueryOrderBool100 runs a query which finds all 100 models ordered
// by the Bool field
func BenchmarkQueryOrderBool100(b *testing.B) {
	benchmarkQueryOrder(b, 100, "Bool")
}

// BenchmarkQueryOrderBool10000 runs a query which finds all 10,000 models
// ordered by the Bool field
func BenchmarkQueryOrderBool10000(b *testing.B) {
	benchmarkQueryOrder(b, 10000, "Bool")
}

// BenchmarkComplexQuery runs a query which incorporates nearly all options.
// The query has a filter on the String and Int Fields, is ordered in reverse
// by the Bool field, and includes only Bool and Int. Out of 1,000 models created,
// 100 should fit the query criteria, but the query limits the number of results
// to 10.
func BenchmarkComplexQuery(b *testing.B) {
	testingSetUp()
	defer testingTearDown()

	// create 1000 models to be saved
	models := createIndexedTestModels(1000)

	// give 100 models an Int value of 1 and all others
	// an Int value of 2
	for _, m := range models[100:200] {
		m.Int = 1
	}
	for _, m := range append(models[0:100], models[200:]...) {
		m.Int = 2
	}
	// give 100 models a String value of "find me" and all others
	// a String value of "not me"
	for _, m := range models[150:250] {
		m.String = "find me"
	}
	for _, m := range append(models[0:150], models[250:]...) {
		m.String = "not me"
	}
	// Save all the models in a single transaction
	t := testPool.NewTransaction()
	for _, model := range models {
		t.Save(indexedTestModels, model)
	}
	if err := t.Exec(); err != nil {
		b.Fatal(err)
	}

	// Construct the query and benchmark it
	q := indexedTestModels.NewQuery().Filter("Int =", 1).Filter("String =", "find me").Order("Bool").Include("Int", "Bool").Limit(10).Offset(10)
	benchmarkQuery(b, q)
}

func benchmarkQuery(b *testing.B, q *Query) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := q.Run(&[]*indexedTestModel{}); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkQueryFilterInt(b *testing.B, selected int, total int) {
	testingSetUp()
	defer testingTearDown()

	models := createIndexedTestModels(total)
	t := testPool.NewTransaction()
	for i := 0; i < selected; i++ {
		models[i].Int = 1
		t.Save(indexedTestModels, models[i])
	}
	for i := selected; i < len(models); i++ {
		models[i].Int = 2
		t.Save(indexedTestModels, models[i])
	}
	if err := t.Exec(); err != nil {
		b.Fatal(err)
	}
	benchmarkQuery(b, indexedTestModels.NewQuery().Filter("Int =", 1))
}

func benchmarkQueryFilterString(b *testing.B, selected int, total int) {
	testingSetUp()
	defer testingTearDown()

	models := createIndexedTestModels(total)
	t := testPool.NewTransaction()
	for i := 0; i < selected; i++ {
		models[i].String = "find me"
		t.Save(indexedTestModels, models[i])
	}
	for i := selected; i < len(models); i++ {
		models[i].String = "not me"
		t.Save(indexedTestModels, models[i])
	}
	if err := t.Exec(); err != nil {
		b.Fatal(err)
	}
	benchmarkQuery(b, indexedTestModels.NewQuery().Filter("String =", "find me"))
}

func benchmarkQueryFilterBool(b *testing.B, selected int, total int) {
	testingSetUp()
	defer testingTearDown()

	models := createIndexedTestModels(total)
	t := testPool.NewTransaction()
	for i := 0; i < selected; i++ {
		models[i].Bool = true
		t.Save(indexedTestModels, models[i])
	}
	for i := selected; i < len(models); i++ {
		models[i].Bool = false
		t.Save(indexedTestModels, models[i])
	}
	if err := t.Exec(); err != nil {
		b.Fatal(err)
	}
	benchmarkQuery(b, indexedTestModels.NewQuery().Filter("Bool =", true))
}

func benchmarkQueryOrder(b *testing.B, n int, field string) {
	testingSetUp()
	defer testingTearDown()

	if _, err := createAndSaveIndexedTestModels(n); err != nil {
		b.Fatal(err)
	}
	q := indexedTestModels.NewQuery().Order(field)
	benchmarkQuery(b, q)
}

// selectUnique selects num random, unique strings from a slice of strings
func selectUnique(num int, ids []string) []string {
	selected := make(map[string]bool)
	for len(selected) < num {
		index := rand.Intn(len(ids) - 1)
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
