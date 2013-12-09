// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File query_test.go tests the query abstraction (query.go)

// TODO: test all edge cases for limit and offset where applicable
// TODO: fix possible bug with limits and offsets for unordered queries

package zoom

import (
	"reflect"
	"strconv"
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

// Test all the corner cases for limits on unordered queries
func TestQueryAllLimit(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	bms, err := newBasicModels(4)
	if err != nil {
		t.Error(err)
	}
	if err := MSave(Models(bms)); err != nil {
		t.Error(err)
	}
	models := Models(bms)

	limits := []uint{
		0, 1, 2, 3, 4, 5,
	}
	expectedLengths := []int{
		4, 1, 2, 3, 4, 4,
	}
	kCombinations := make(map[int][][]Model)
	// 4 choose 1
	kCombinations[1] = [][]Model{
		[]Model{models[0]},
		[]Model{models[1]},
		[]Model{models[2]},
		[]Model{models[3]},
	}
	// 4 choose 2
	kCombinations[2] = [][]Model{
		[]Model{models[0], models[1]},
		[]Model{models[0], models[2]},
		[]Model{models[0], models[3]},
		[]Model{models[1], models[2]},
		[]Model{models[1], models[3]},
		[]Model{models[2], models[3]},
	}
	// 4 choose 3
	kCombinations[3] = [][]Model{
		[]Model{models[0], models[1], models[2]},
		[]Model{models[0], models[1], models[3]},
		[]Model{models[0], models[2], models[3]},
		[]Model{models[1], models[2], models[3]},
	}
	// 4 choose 4
	kCombinations[4] = [][]Model{
		[]Model{models[0], models[1], models[2], models[3]},
	}

	for i, limit := range limits {
		expectedLength := expectedLengths[i]
		if expectedLength == len(models) {
			q := NewQuery("basicModel").Limit(limit)
			testQueryWithExpectedModels(t, q, models, false)
		} else {
			results, err := NewQuery("basicModel").Limit(limit).Run()
			if err != nil {
				t.Error(err)
			}
			// We expect that at most N unique models will be returned, where
			// N is the limit. Since the query is unordered, there is no definition
			// of which N models zoom will return. So we have to check all possibilities
			// of choosing N models from the slice of models. If one matches, then the
			// test should pass. If none of the possible combinations match, it should fail
			for _, expected := range kCombinations[int(limit)] {
				if equal, _ := compareAsSet(expected, results); equal {
					return
				}
			}
			// if we reached here, none of the possible combinations passed
			t.Errorf("results were invalid on iteration %d\nExpected some combination of %d unique models\nGot: ", i, expectedLength, modelIds(Models(results)))
		}
	}

}

// Tests all the corner cases for counting with limits
func TestQueryAllLimitCount(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	limits := []uint{
		0, 2, 4,
	}
	expecteds := []int{
		3, 2, 3,
	}

	ms, err := newBasicModels(3)
	if err != nil {
		t.Error(err)
	}
	if err := MSave(Models(ms)); err != nil {
		t.Error(err)
	}

	for i, limit := range limits {
		expected := expecteds[i]
		got, err := NewQuery("basicModel").Limit(limit).Count()
		if err != nil {
			t.Error(err)
		}

		if got != expected {
			t.Errorf("Model count incorrect on iteration %d. Expected %d but got %d\n", i, expected, got)
		}
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

func TestQueryAllIncludes(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms, err := newPrimativeTypesModels(1)
	if err != nil {
		t.Error(err)
	}
	m := ms[0]
	if err := Save(m); err != nil {
		t.Error(err)
	}

	mCopies := make([]*primativeTypesModel, 0)
	if err := NewQuery("primativeTypesModel").Include("Int", "String").Scan(&mCopies); err != nil {
		t.Error(err)
	}
	mCopy := mCopies[0]

	// The Int and String fields, which were included, should be the same
	if m.Int != mCopy.Int {
		t.Errorf("Int field was incorrect.\nExpected: %d\nGot: %d\n", m.Int, mCopy.Int)
	}
	if m.String != mCopy.String {
		t.Errorf("String field was incorrect.\nExpected: %d\nGot: %d\n", m.String, mCopy.String)
	}

	// The remaining fields in mCopy should be blank
	if mCopy.Uint != 0 {
		t.Errorf("Expected mCopy.Uint to be 0 but was %d", mCopy.Uint)
	}
	if mCopy.Uint8 != 0 {
		t.Errorf("Expected mCopy.Uint8 to be 0 but was %d", mCopy.Uint8)
	}
	if mCopy.Uint16 != 0 {
		t.Errorf("Expected mCopy.Uint16 to be 0 but was %d", mCopy.Uint16)
	}
	if mCopy.Uint32 != 0 {
		t.Errorf("Expected mCopy.Uint32 to be 0 but was %d", mCopy.Uint32)
	}
	if mCopy.Uint64 != 0 {
		t.Errorf("Expected mCopy.Uint64 to be 0 but was %d", mCopy.Uint64)
	}
	if mCopy.Int8 != 0 {
		t.Errorf("Expected mCopy.Int8 to be 0 but was %d", mCopy.Int8)
	}
	if mCopy.Int16 != 0 {
		t.Errorf("Expected mCopy.Int16 to be 0 but was %d", mCopy.Int16)
	}
	if mCopy.Int32 != 0 {
		t.Errorf("Expected mCopy.Int32 to be 0 but was %d", mCopy.Int32)
	}
	if mCopy.Int64 != 0 {
		t.Errorf("Expected mCopy.Int64 to be 0 but was %d", mCopy.Int64)
	}
	if mCopy.Float32 != 0.0 {
		t.Errorf("Expected mCopy.Float32 to be 0 but was %1.1f", mCopy.Float32)
	}
	if mCopy.Float64 != 0.0 {
		t.Errorf("Expected mCopy.Float64 to be 0 but was %1.1f", mCopy.Float64)
	}
	if mCopy.Byte != byte(0) {
		t.Errorf("Expected mCopy.Byte to be 0 but was %d", mCopy.Byte)
	}
	if mCopy.Rune != rune(0) {
		t.Errorf("Expected mCopy.Rune to be 0 but was %d", mCopy.Rune)
	}
	if mCopy.Bool != false {
		t.Errorf("Expected mCopy.Bool to be false but was %t", mCopy.Bool)
	}
}

func TestQueryAllExcludes(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms, err := newPrimativeTypesModels(1)
	if err != nil {
		t.Error(err)
	}
	m := ms[0]
	if err := Save(m); err != nil {
		t.Error(err)
	}

	mCopies := make([]*primativeTypesModel, 0)
	if err := NewQuery("primativeTypesModel").Exclude("Int", "String").Scan(&mCopies); err != nil {
		t.Error(err)
	}
	mCopy := mCopies[0]

	// The Int and String fields, which were excluded, should be blank.
	if mCopy.Int != 0 {
		t.Errorf("Expected mCopy.Int to be 0 but was %d", mCopy.Int)
	}
	if mCopy.String != "" {
		t.Errorf("Expected mCopy.String to be '' but was %s", mCopy.String)
	}

	// The remaining fields in mCopy should be blank
	if mCopy.Uint != m.Uint {
		t.Errorf("mCopy.Uint was incorrect.\nExpected: %d\nGot: %d", m.Uint, mCopy.Uint)
	}
	if mCopy.Uint8 != m.Uint8 {
		t.Errorf("mCopy.Uint8 was incorrect.\nExpected: %d\nGot: %d", m.Uint8, mCopy.Uint8)
	}
	if mCopy.Uint16 != m.Uint16 {
		t.Errorf("mCopy.Uint16 was incorrect.\nExpected: %d\nGot: %d", m.Uint16, mCopy.Uint16)
	}
	if mCopy.Uint32 != m.Uint32 {
		t.Errorf("mCopy.Uint32 was incorrect.\nExpected: %d\nGot: %d", m.Uint32, mCopy.Uint32)
	}
	if mCopy.Uint64 != m.Uint64 {
		t.Errorf("mCopy.Uint64 was incorrect.\nExpected: %d\nGot: %d", m.Uint64, mCopy.Uint64)
	}
	if mCopy.Int8 != m.Int8 {
		t.Errorf("mCopy.Int8 was incorrect.\nExpected: %d\nGot: %d", m.Int8, mCopy.Int8)
	}
	if mCopy.Int16 != m.Int16 {
		t.Errorf("mCopy.Int16 was incorrect.\nExpected: %d\nGot: %d", m.Int16, mCopy.Int16)
	}
	if mCopy.Int32 != m.Int32 {
		t.Errorf("mCopy.Int32 was incorrect.\nExpected: %d\nGot: %d", m.Int32, mCopy.Int32)
	}
	if mCopy.Int64 != m.Int64 {
		t.Errorf("mCopy.Int64 was incorrect.\nExpected: %d\nGot: %d", m.Int64, mCopy.Int64)
	}
	if mCopy.Float32 != m.Float32 {
		t.Errorf("mCopy.Float32 was incorrect.\nExpected: %1.1f\nGot: %1.1f\n", m.Float32, mCopy.Float32)
	}
	if mCopy.Float64 != m.Float64 {
		t.Errorf("mCopy.Float64 was incorrect.\nExpected: %1.1f\nGot: %1.1f\n", m.Float64, mCopy.Float64)
	}
	if mCopy.Byte != m.Byte {
		t.Errorf("mCopy.Byte was incorrect.\nExpected: %d\nGot: %d\n", m.Byte, mCopy.Byte)
	}
	if mCopy.Rune != m.Rune {
		t.Errorf("mCopy.Rune was incorrect.\nExpected: %d\nGot: %d\n", m.Rune, mCopy.Rune)
	}
	if mCopy.Bool != m.Bool {
		t.Errorf("mCopy.Bool was incorrect.\nExpected: %t\nGot: %t\n", m.Bool, mCopy.Bool)
	}
}

func createOrderableNumericModels(num int) ([]*indexedPrimativesModel, error) {
	ms := []*indexedPrimativesModel{}
	for i := 0; i < num; i++ {
		m := &indexedPrimativesModel{
			Int: i,
		}
		ms = append(ms, m)
	}

	if err := MSave(Models(ms)); err != nil {
		return ms, err
	}
	return ms, nil
}

func TestOrderNumericAsc(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	if ms, err := createOrderableNumericModels(5); err != nil {
		t.Error(err)
	} else {
		q := NewQuery("indexedPrimativesModel").Order("Int")
		testQueryWithExpectedIds(t, q, modelIds(Models(ms)), true)
	}
}

func TestOrderNumericDesc(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms, err := createOrderableNumericModels(5)
	if err != nil {
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

// only create 2 here. It's much easier to test
func createOrderableBooleanModels() ([]*indexedPrimativesModel, error) {
	ms := make([]*indexedPrimativesModel, 2)
	ms[0] = &indexedPrimativesModel{
		Bool: false,
	}
	ms[1] = &indexedPrimativesModel{
		Bool: true,
	}

	if err := MSave(Models(ms)); err != nil {
		return ms, err
	}
	return ms, nil
}

func TestOrderBooleanAsc(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	if ms, err := createOrderableBooleanModels(); err != nil {
		t.Error(err)
	} else {
		q := NewQuery("indexedPrimativesModel").Order("Bool")
		testQueryWithExpectedIds(t, q, modelIds(Models(ms)), true)
	}
}

func TestOrderBooleanDesc(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms, err := createOrderableBooleanModels()
	if err != nil {
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

func createOrderableAlphaModels(num int) ([]*indexedPrimativesModel, error) {
	ms := []*indexedPrimativesModel{}
	for i := 0; i < num; i++ {
		m := &indexedPrimativesModel{
			String: strconv.Itoa(i),
		}
		ms = append(ms, m)
	}

	if err := MSave(Models(ms)); err != nil {
		return ms, err
	}
	return ms, nil
}

func TestOrderAlphaAsc(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	if ms, err := createOrderableAlphaModels(5); err != nil {
		t.Error(err)
	} else {
		q := NewQuery("indexedPrimativesModel").Order("String")
		testQueryWithExpectedIds(t, q, modelIds(Models(ms)), true)
	}
}

func TestOrderAlphaDesc(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms, err := createOrderableAlphaModels(5)
	if err != nil {
		t.Error(err)
	}

	// expected ids is reversed
	expectedIds := make([]string, len(ms))
	for i, j := 0, len(ms)-1; i <= j; i, j = i+1, j-1 {
		expectedIds[i], expectedIds[j] = ms[j].getId(), ms[i].getId()
	}

	q := NewQuery("indexedPrimativesModel").Order("-String")
	testQueryWithExpectedIds(t, q, expectedIds, true)
}

func TestNumericOrderAscLimitOffset(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	testNumericOrderLimitOffset(t, ascending)
}

func TestNumericOrderDescLimitOffset(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	testNumericOrderLimitOffset(t, descending)
}

func testNumericOrderLimitOffset(t *testing.T, typ orderType) {
	onms, err := createOrderableNumericModels(4)
	if err != nil {
		t.Error(err)
	}
	models := Models(onms)

	limits := []uint{
		0, 0, 0, 0, 1, 1, 1, 1, 3, 3, 3, 3, 4, 4, 4, 4, 5, 5, 5, 5,
	}
	offsets := []uint{
		0, 1, 3, 5, 0, 1, 3, 5, 0, 1, 3, 5, 0, 1, 3, 5, 0, 1, 3, 5,
	}
	empty := Models([]*basicModel{})

	if typ == descending {
		// reverse models
		for i, j := 0, len(models)-1; i <= j; i, j = i+1, j-1 {
			models[i], models[j] = models[j], models[i]
		}
	}
	expecteds := [][]Model{
		models,
		models[1:],
		models[3:],
		empty,
		models[0:1],
		models[1:2],
		models[3:4],
		empty,
		models[0:3],
		models[1:4],
		models[3:4],
		empty,
		models,
		models[1:4],
		models[3:4],
		empty,
		models,
		models[1:4],
		models[3:4],
		empty,
	}

	for i, limit := range limits {
		offset := offsets[i]
		expected := expecteds[i]
		var q *Query
		if typ == ascending {
			q = NewQuery("indexedPrimativesModel").Order("Int").Limit(limit).Offset(offset)
		} else if typ == descending {
			q = NewQuery("indexedPrimativesModel").Order("-Int").Limit(limit).Offset(offset)
		}
		if results, err := q.Run(); err != nil {
			t.Error(err)
		} else {
			if len(expected) == 0 && len(Models(results)) == 0 {
				continue
			}
			if equal, err := looseEquals(expected, results); err != nil {
				t.Error(err)
			} else if !equal {
				t.Errorf("Fail on iteration %d\nLimit = %d, Offset = %d\nExpected: %v\nGot %v\n", i, limit, offset, modelIds(expected), modelIds(Models(results)))
			}
		}
	}
}

func TestNumericOrderAscLimitOffsetCount(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	testNumericOrderLimitOffsetCount(t, ascending)
}

func TestNumericOrderDescLimitOffsetCount(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	testNumericOrderLimitOffsetCount(t, descending)
}

func TestFilterById(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms, err := newBasicModels(2)
	if err != nil {
		t.Error(err)
	}
	if err := MSave(Models(ms)); err != nil {
		t.Error(err)
	}

	q := NewQuery("basicModel").Fitler("Id =", ms[0].Id)
	testQueryWithExpectedModels(t, q, Models(ms[0:1]), false)
}

func TestNumericFilterEqual(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms := make([]*indexedPrimativesModel, 4)
	ms[0] = &indexedPrimativesModel{Int: 0}
	ms[1] = &indexedPrimativesModel{Int: 1}
	ms[2] = &indexedPrimativesModel{Int: 2}
	ms[3] = &indexedPrimativesModel{Int: 2}

	// only save the first 3 models
	if err := MSave(Models(ms[0:3])); err != nil {
		t.Error(err)
	}
	// run some test queries
	q := NewQuery("indexedPrimativesModel").Fitler("Int =", 0)
	testQueryWithExpectedModels(t, q, Models(ms[0:1]), false)
	q = NewQuery("indexedPrimativesModel").Fitler("Int =", 2)
	testQueryWithExpectedModels(t, q, Models(ms[2:3]), false)
	q = NewQuery("indexedPrimativesModel").Fitler("Int =", 3)
	testQueryWithExpectedModels(t, q, []Model{}, false)
	q = NewQuery("indexedPrimativesModel").Fitler("Int =", -1)
	testQueryWithExpectedModels(t, q, []Model{}, false)

	// now save the 4th model
	if err := Save(ms[3]); err != nil {
		t.Error(err)
	}
	// now there should be two models with Int = 2
	q = NewQuery("indexedPrimativesModel").Fitler("Int =", 2)
	testQueryWithExpectedModels(t, q, Models(ms[2:4]), false)
}

func TestBooleanFilterEqual(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms := make([]*indexedPrimativesModel, 3)
	ms[0] = &indexedPrimativesModel{Bool: false}
	ms[1] = &indexedPrimativesModel{Bool: true}
	ms[2] = &indexedPrimativesModel{Bool: true}

	// save the first model
	if err := Save(ms[0]); err != nil {
		t.Error(err)
	}
	// run some test queries
	q := NewQuery("indexedPrimativesModel").Fitler("Bool =", true)
	testQueryWithExpectedModels(t, q, []Model{}, false)

	// only save the 2nd model
	if err := Save(ms[1]); err != nil {
		t.Error(err)
	}
	// run some test queries
	q = NewQuery("indexedPrimativesModel").Fitler("Bool =", false)
	testQueryWithExpectedModels(t, q, Models(ms[0:1]), false)
	q = NewQuery("indexedPrimativesModel").Fitler("Bool =", true)
	testQueryWithExpectedModels(t, q, Models(ms[1:2]), false)

	// now save the 3rd model
	if err := Save(ms[2]); err != nil {
		t.Error(err)
	}
	// now there should be two models with Bool = true
	q = NewQuery("indexedPrimativesModel").Fitler("Bool =", true)
	testQueryWithExpectedModels(t, q, Models(ms[1:3]), false)
}

func TestAlphaFilterEqual(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms := make([]*indexedPrimativesModel, 4)
	ms[0] = &indexedPrimativesModel{String: "a"}
	ms[1] = &indexedPrimativesModel{String: "b"}
	ms[2] = &indexedPrimativesModel{String: "c"}
	ms[3] = &indexedPrimativesModel{String: "c"}

	// only save the first 3 models
	if err := MSave(Models(ms[0:3])); err != nil {
		t.Error(err)
	}
	// run some test queries
	q := NewQuery("indexedPrimativesModel").Fitler("String =", "a")
	testQueryWithExpectedModels(t, q, Models(ms[0:1]), false)
	q = NewQuery("indexedPrimativesModel").Fitler("String =", "c")
	testQueryWithExpectedModels(t, q, Models(ms[2:3]), false)
	q = NewQuery("indexedPrimativesModel").Fitler("String =", "d")
	testQueryWithExpectedModels(t, q, []Model{}, false)
	q = NewQuery("indexedPrimativesModel").Fitler("String =", "å")
	testQueryWithExpectedModels(t, q, []Model{}, false)

	// now save the 4th model
	if err := Save(ms[3]); err != nil {
		t.Error(err)
	}
	// now there should be two models with String = "c"
	q = NewQuery("indexedPrimativesModel").Fitler("String =", "c")
	testQueryWithExpectedModels(t, q, Models(ms[2:4]), false)
}

func testNumericOrderLimitOffsetCount(t *testing.T, typ orderType) {
	_, err := createOrderableNumericModels(4)
	if err != nil {
		t.Error(err)
	}

	limits := []uint{
		0, 0, 0, 0, 1, 1, 1, 1, 3, 3, 3, 3, 4, 4, 4, 4, 5, 5, 5, 5,
	}
	offsets := []uint{
		0, 1, 3, 5, 0, 1, 3, 5, 0, 1, 3, 5, 0, 1, 3, 5, 0, 1, 3, 5,
	}
	expecteds := []int{
		4, 3, 1, 0, 1, 1, 1, 0, 3, 3, 1, 0, 4, 3, 1, 0, 4, 3, 1, 0,
	}

	for i, limit := range limits {
		offset := offsets[i]
		expected := expecteds[i]
		var q *Query
		if typ == ascending {
			q = NewQuery("indexedPrimativesModel").Order("Int").Limit(limit).Offset(offset)
		} else if typ == descending {
			q = NewQuery("indexedPrimativesModel").Order("-Int").Limit(limit).Offset(offset)
		}
		if count, err := q.Count(); err != nil {
			t.Error(err)
		} else {
			if count != expected {
				t.Errorf("Fail on iteration %d\nLimit = %d, Offset = %d\nExpected count to be %d, but got %d\n", i, limit, offset, expected, count)
			}
		}
	}
}

func testQueryWithExpectedModels(t *testing.T, query RunScanner, expected []Model, orderMatters bool) {
	queryTester(t, query, func(t *testing.T, results interface{}) {
		// make sure results is the right length
		if reflect.ValueOf(results).Len() != len(expected) {
			t.Errorf("results was not the right length. Expected: %d. Got: %d.\n", len(expected), reflect.ValueOf(results).Len())
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
			t.Errorf("results was not the right length. Expected: %d. Got: %d.\n", len(expected), len(gotIds))
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
