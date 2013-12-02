// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File query_test.go tests the query abstraction (query.go)

package zoom

import (
	"reflect"
	"testing"
)

func TestQueryAllRun(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms, err := newBasicModels(5)
	if err != nil {
		t.Error(err)
	}
	if err := MSave(Models(ms)); err != nil {
		t.Error(err)
	}
	modelsMap := make(map[string]*basicModel)
	for _, m := range ms {
		modelsMap[m.Id] = m
	}

	results, err := NewQuery("basicModel").Run()
	if err != nil {
		t.Error(err)
	}
	gots := results.([]*basicModel)
	if len(gots) != len(ms) {
		t.Errorf("gots was not the right length.\nExpected: %d\nGot: %d\n", len(ms), len(gots))
	}

	for i, got := range gots {
		if got.getId() == "" {
			t.Errorf("Got model has nil id on iteration %d", i)
		}
		expected, found := modelsMap[got.Id]
		if !found {
			t.Errorf("Got unexpected id: %s", got.Id)
		}
		if equal, err := looseEquals(expected, got); err != nil {
			t.Error(err)
		} else if !equal {
			t.Errorf("Got model was not valid.\nExpected: %+v\nGot: %+v\n", expected, got)
		}
	}
}

func TestQueryAllScan(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms, err := newBasicModels(5)
	if err != nil {
		t.Error(err)
	}
	if err := MSave(Models(ms)); err != nil {
		t.Error(err)
	}
	modelsMap := make(map[string]*basicModel)
	for _, m := range ms {
		modelsMap[m.Id] = m
	}

	var gots []*basicModel
	if err := NewQuery("basicModel").Scan(&gots); err != nil {
		t.Error(err)
	}
	if len(gots) != len(ms) {
		t.Errorf("gots was not the right length.\nExpected: %d\nGot: %d\n", len(ms), len(gots))
	}

	for i, got := range gots {
		if got.getId() == "" {
			t.Errorf("Got model has nil id on iteration %d", i)
		}
		expected, found := modelsMap[got.Id]
		if !found {
			t.Errorf("Got unexpected id: %s", got.Id)
		}
		if equal, err := looseEquals(expected, got); err != nil {
			t.Error(err)
		} else if !equal {
			t.Errorf("Got model was not valid.\nExpected: %+v\nGot: %+v\n", expected, got)
		}
	}
}

func TestQueryAllCount(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms, err := newBasicModels(5)
	if err != nil {
		t.Error(err)
	}
	if err := MSave(Models(ms)); err != nil {
		t.Error(err)
	}

	got, err := NewQuery("basicModel").Count()
	if err != nil {
		t.Error(err)
	}

	if got != 5 {
		t.Errorf("Model count incorrect. Expected 5 but got %d\n", got)
	}
}

func TestQueryAllIdsOnly(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms, err := newBasicModels(5)
	if err != nil {
		t.Error(err)
	}
	if err := MSave(Models(ms)); err != nil {
		t.Error(err)
	}

	gotIds, err := NewQuery("basicModel").IdsOnly()
	if err != nil {
		t.Error(err)
	}
	expectedIds := make([]string, 0)
	for _, m := range ms {
		expectedIds = append(expectedIds, m.Id)
	}

	if equal, msg := compareAsSet(expectedIds, gotIds); !equal {
		t.Errorf("Ids were incorrect.\nExpected: %v\nGot: %v\n%s\n", expectedIds, gotIds, msg)
	}
}

func TestOrderNumericAsc(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms := []*indexedPrimativesModel{}
	for i := 0; i < 5; i++ {
		m := &indexedPrimativesModel{
			Int: i,
		}
		ms = append(ms, m)
	}

	if err := MSave(Models(ms)); err != nil {
		t.Error(err)
	}

	q := NewQuery("indexedPrimativesModel").Order("Int")
	testQueryWithExpectedIds(t, q, modelIds(Models(ms)), true)
}

func TestOrderNumericDesc(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms := []*indexedPrimativesModel{}
	for i := 0; i < 5; i++ {
		m := &indexedPrimativesModel{
			Int: i,
		}
		ms = append(ms, m)
	}

	if err := MSave(Models(ms)); err != nil {
		t.Error(err)
	}

	// expected ids is reversed
	expectedIds := make([]string, len(ms))
	for i, j := 0, len(ms)-1; i <= j; i, j = i+1, j-1 {
		expectedIds[i], expectedIds[j] = ms[j].getId(), ms[i].getId()
	}

	q := NewQuery("indexedPrimativesModel").Order("-Int")
	testQueryWithExpectedIds(t, q, expectedIds, true)
}

func TestOrderBooleanAsc(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms := make([]*indexedPrimativesModel, 2)
	ms[0] = &indexedPrimativesModel{
		Bool: false,
	}
	ms[1] = &indexedPrimativesModel{
		Bool: true,
	}

	if err := MSave(Models(ms)); err != nil {
		t.Error(err)
	}

	q := NewQuery("indexedPrimativesModel").Order("Bool")
	testQueryWithExpectedIds(t, q, modelIds(Models(ms)), true)
}

func TestOrderBooleanDesc(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms := make([]*indexedPrimativesModel, 2)
	ms[0] = &indexedPrimativesModel{
		Bool: false,
	}
	ms[1] = &indexedPrimativesModel{
		Bool: true,
	}

	if err := MSave(Models(ms)); err != nil {
		t.Error(err)
	}

	// expected ids is reversed
	expectedIds := make([]string, len(ms))
	for i, j := 0, len(ms)-1; i <= j; i, j = i+1, j-1 {
		expectedIds[i], expectedIds[j] = ms[j].getId(), ms[i].getId()
	}

	q := NewQuery("indexedPrimativesModel").Order("-Bool")
	testQueryWithExpectedIds(t, q, expectedIds, true)
}

func testQueryWithExpectedModels(t *testing.T, query RunScanner, expected []Model, orderMatters bool) {
	queryTester(t, query, func(t *testing.T, results interface{}) {
		// make sure results is the right length
		if reflect.ValueOf(results).Len() != len(expected) {
			t.Errorf("results was not the right length. Expected: %d. Got: %d.\n", reflect.ValueOf(results).Len(), len(expected))
		}

		// compare expected to results
		if orderMatters {
			if !reflect.DeepEqual(expected, results) {
				t.Errorf("Results were incorrect.\nExpected: %v\nGot: %v\n", modelIds(expected), modelIds(Models(results)))
			}
		} else {
			equal, msg := compareAsSet(expected, results)
			if !equal {
				t.Errorf("Results were incorrect\n%s\nExpected: %v\nGot: %v\n", msg, modelIds(expected), modelIds(Models(results)))
			}
		}
	})
}

func testQueryWithExpectedIds(t *testing.T, query RunScanner, expected []string, orderMatters bool) {
	queryTester(t, query, func(t *testing.T, results interface{}) {
		gotIds := modelIds(Models(results))

		// make sure results is the right length
		if len(gotIds) != len(expected) {
			t.Errorf("results was not the right length. Expected: %d. Got: %d.\n", len(gotIds), len(expected))
		}

		// compare expected to results
		if orderMatters {
			if !reflect.DeepEqual(expected, gotIds) {
				t.Errorf("Results were incorrect.\nExpected: %v\nGot: %v\n", expected, gotIds)
			}
		} else {
			equal, msg := compareAsSet(expected, gotIds)
			if !equal {
				t.Errorf("Results were incorrect\n%s\nExpected: %v\nGot: %v\n", msg, expected, gotIds)
			}
		}
	})
}

func queryTester(t *testing.T, query RunScanner, checker func(*testing.T, interface{})) {
	// execute the query
	results, err := query.Run()
	if err != nil {
		t.Error(err)
	}

	// run the checker function
	checker(t, results)
}
