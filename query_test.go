// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File query_test.go tests the query abstraction (query.go)

package zoom

import (
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"testing"
)

// TODO:
// 	- Write high-level tests for every possible combination of query modifiers

func TestQueryAll(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms, err := newIndexedPrimativesModels(5)
	if err != nil {
		t.Error(err)
	}
	if err := MSave(Models(ms)); err != nil {
		t.Error(err)
	}

	q := NewQuery("indexedPrimativesModel")
	testQuery(t, q, ms)
}

// === Test the Order modifier

func TestQueryOrderNumeric(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// create models which we will try to sort
	models, err := createOrderableNumericModels(9)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// create some test queries to sort the models
	queries := []*Query{
		NewQuery("indexedPrimativesModel").Order("Int"),
		NewQuery("indexedPrimativesModel").Order("-Int"),
	}

	for _, q := range queries {
		testQuery(t, q, models)
	}
}

func TestQueryOrderBoolean(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// create models which we will try to sort
	models, err := createOrderableBooleanModels()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// create some test queries to sort the models
	queries := []*Query{
		NewQuery("indexedPrimativesModel").Order("Bool"),
		NewQuery("indexedPrimativesModel").Order("-Bool"),
	}

	for _, q := range queries {
		testQuery(t, q, models)
	}
}

func TestQueryOrderAlpha(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// create models which we will try to sort
	models, err := createOrderableAlphaModels(9)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// create some test queries to sort the models
	queries := []*Query{
		NewQuery("indexedPrimativesModel").Order("String"),
		NewQuery("indexedPrimativesModel").Order("-String"),
	}

	for _, q := range queries {
		testQuery(t, q, models)
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

	// shuffle the order
	for i := range ms {
		j := rand.Intn(i + 1)
		ms[i], ms[j] = ms[j], ms[i]
	}

	if err := MSave(Models(ms)); err != nil {
		return ms, err
	}
	return ms, nil
}

// Only create 2 here. It's much easier to test.
// With more than two models the ordering is unstable
func createOrderableBooleanModels() ([]*indexedPrimativesModel, error) {
	ms := make([]*indexedPrimativesModel, 2)
	ms[0] = &indexedPrimativesModel{
		Bool: true,
	}
	ms[1] = &indexedPrimativesModel{
		Bool: false,
	}

	if err := MSave(Models(ms)); err != nil {
		return ms, err
	}
	return ms, nil
}

// NOTE: numbers > 9 might not work the way you'd expect
// From an alphanumeric perspective 123 < 2
func createOrderableAlphaModels(num int) ([]*indexedPrimativesModel, error) {
	ms := []*indexedPrimativesModel{}
	for i := 0; i < num; i++ {
		m := &indexedPrimativesModel{
			String: strconv.Itoa(i),
		}
		ms = append(ms, m)
	}

	// shuffle the order
	for i := range ms {
		j := rand.Intn(i + 1)
		ms[i], ms[j] = ms[j], ms[i]
	}

	if err := MSave(Models(ms)); err != nil {
		return ms, err
	}
	return ms, nil
}

// There's a huge amount of test cases to cover above.
// Below is some code that makes it easier, but needs to be
// tested itself. Testing for correctness using a brute force
// approach (obviously slow compared to what Zoom is actually doing) is
// fine because the tests in this file will typically use only a handful
// of models. The brute force approach is also easier becuase you can apply
// query modifiers independently, in any order. (Whereas behind the scenes
// zoom actually does some clever optimization and changing any single paramater
// or modifier of the query could completely change the command sent to Redis).
// We're assuming that for most tests the indexedPrimativesModel
// type will be used.

// testQuery compares the results of the Query run by Zoom with the results
// of a simpler implementation which doesn't touch the database. If the results match,
// then the query was correct and the test will pass. Models should be an array of all
// the models which are being queried against.
func testQuery(t *testing.T, q *Query, models []*indexedPrimativesModel) {
	expected, err := expectedResultsForQuery(q, models)
	if err != nil {
		t.Error(err)
	}

	got := make([]*indexedPrimativesModel, 0)
	if err := q.Scan(&got); err != nil {
		t.Error(err)
	}
	orderMatters := false
	if q.order.fieldName != "" {
		orderMatters = true
	}
	compareModelSlices(t, expected, got, orderMatters)
}

func expectedResultsForQuery(q *Query, models []*indexedPrimativesModel) ([]*indexedPrimativesModel, error) {
	expected := make([]*indexedPrimativesModel, len(models))
	copy(expected, models)

	// apply filters
	for _, f := range q.filters {
		fmodels, err := filterModels(models, f.fieldName, f.filterType, f.filterValue, f.indexType)
		if err != nil {
			return nil, err
		}
		expected = orderedIntersectModels(fmodels, expected)
	}

	if q.order.fieldName != "" {
		if q.order.orderType == descending {
			expected = reverseModels(orderModels(expected, q.order.fieldName))
		} else {
			expected = orderModels(expected, q.order.fieldName)
		}
	}

	start := q.offset
	var end uint
	if q.limit == 0 {
		end = uint(len(expected))
	} else {
		end = start + q.limit
	}

	if int(start) > len(expected) {
		expected = []*indexedPrimativesModel{}
	} else if int(end) > len(expected) {
		expected = expected[start:]
	} else {
		expected = expected[start:end]
	}

	return expected, nil
}

// filterModels returns only those models which pass the filter,
// or an error, if there was one. It constructs a selector function
// to pass to mapModels. It relies on reflection.
func filterModels(models []*indexedPrimativesModel, fieldName string, fType filterType, fVal interface{}, iType indexType) ([]*indexedPrimativesModel, error) {
	var s func(m *indexedPrimativesModel) (bool, error)

	switch iType {

	case indexNumeric:
		switch fType {
		case equal:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldInt := reflect.ValueOf(m).Elem().FieldByName(fieldName).Int()
				valueInt := reflect.ValueOf(fVal).Int()
				return fieldInt == valueInt, nil
			}
		case notEqual:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldInt := reflect.ValueOf(m).Elem().FieldByName(fieldName).Int()
				valueInt := reflect.ValueOf(fVal).Int()
				return fieldInt != valueInt, nil
			}
		case greater:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldInt := reflect.ValueOf(m).Elem().FieldByName(fieldName).Int()
				valueInt := reflect.ValueOf(fVal).Int()
				return fieldInt > valueInt, nil
			}
		case less:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldInt := reflect.ValueOf(m).Elem().FieldByName(fieldName).Int()
				valueInt := reflect.ValueOf(fVal).Int()
				return fieldInt < valueInt, nil
			}
		case greaterOrEqual:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldInt := reflect.ValueOf(m).Elem().FieldByName(fieldName).Int()
				valueInt := reflect.ValueOf(fVal).Int()
				return fieldInt >= valueInt, nil
			}
		case lessOrEqual:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldInt := reflect.ValueOf(m).Elem().FieldByName(fieldName).Int()
				valueInt := reflect.ValueOf(fVal).Int()
				return fieldInt <= valueInt, nil
			}
		}

	case indexBoolean:
		switch fType {
		case equal:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldBool := reflect.ValueOf(m).Elem().FieldByName(fieldName).Bool()
				valueBool := reflect.ValueOf(fVal).Bool()
				return fieldBool == valueBool, nil
			}
		case notEqual:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldBool := reflect.ValueOf(m).Elem().FieldByName(fieldName).Bool()
				valueBool := reflect.ValueOf(fVal).Bool()
				return fieldBool != valueBool, nil
			}
		case greater:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldBool := reflect.ValueOf(m).Elem().FieldByName(fieldName).Bool()
				valueBool := reflect.ValueOf(fVal).Bool()
				return boolToInt(fieldBool) > boolToInt(valueBool), nil
			}
		case less:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldBool := reflect.ValueOf(m).Elem().FieldByName(fieldName).Bool()
				valueBool := reflect.ValueOf(fVal).Bool()
				return boolToInt(fieldBool) < boolToInt(valueBool), nil
			}
		case greaterOrEqual:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldBool := reflect.ValueOf(m).Elem().FieldByName(fieldName).Bool()
				valueBool := reflect.ValueOf(fVal).Bool()
				return boolToInt(fieldBool) >= boolToInt(valueBool), nil
			}
		case lessOrEqual:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldBool := reflect.ValueOf(m).Elem().FieldByName(fieldName).Bool()
				valueBool := reflect.ValueOf(fVal).Bool()
				return boolToInt(fieldBool) <= boolToInt(valueBool), nil
			}

		}

	// NOTE: this implementation only considers the first letter of the
	// string! Makes it a lot easier to implement alphabetical sorting.
	case indexAlpha:
		switch fType {
		case equal:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldString := reflect.ValueOf(m).Elem().FieldByName(fieldName).String()
				valueString := reflect.ValueOf(fVal).String()
				return fieldString == valueString, nil
			}
		case notEqual:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldString := reflect.ValueOf(m).Elem().FieldByName(fieldName).String()
				valueString := reflect.ValueOf(fVal).String()
				return fieldString != valueString, nil
			}
		case greater:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldString := reflect.ValueOf(m).Elem().FieldByName(fieldName).String()
				valueString := reflect.ValueOf(fVal).String()
				return fieldString[0] > valueString[0], nil
			}
		case less:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldString := reflect.ValueOf(m).Elem().FieldByName(fieldName).String()
				valueString := reflect.ValueOf(fVal).String()
				return fieldString[0] < valueString[0], nil
			}
		case greaterOrEqual:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldString := reflect.ValueOf(m).Elem().FieldByName(fieldName).String()
				valueString := reflect.ValueOf(fVal).String()
				return fieldString[0] >= valueString[0], nil
			}
		case lessOrEqual:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldString := reflect.ValueOf(m).Elem().FieldByName(fieldName).String()
				valueString := reflect.ValueOf(fVal).String()
				return fieldString[0] <= valueString[0], nil
			}
		}
	}

	return mapModels(models, s)
}

// mapModels returns only those models which return true
// when passed through the selector function or an error,
// if there was one.
func mapModels(models []*indexedPrimativesModel, selector func(*indexedPrimativesModel) (bool, error)) ([]*indexedPrimativesModel, error) {
	results := make([]*indexedPrimativesModel, 0)
	for _, m := range models {
		if match, err := selector(m); err != nil {
			return results, err
		} else if match {
			results = append(results, m)
		}
	}
	return results, nil
}

// Test our internal model filter with numeric type indexes
func TestInternalFilterModelsNumeric(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	models, _ := newIndexedPrimativesModels(5)
	models[0].Int = -4
	models[1].Int = 1
	models[2].Int = 1
	models[3].Int = 2
	models[4].Int = 3

	type testCase struct {
		name     string
		fType    filterType
		fVal     interface{}
		expected []*indexedPrimativesModel
	}
	testCases := []testCase{
		testCase{
			"Equal",
			equal,
			1,
			models[1:3],
		},
		testCase{
			"Not Equal",
			notEqual,
			3,
			models[0:4],
		},
		testCase{
			"Less: none",
			less,
			-4,
			[]*indexedPrimativesModel{},
		},
		testCase{
			"Less: middle",
			less,
			2,
			models[0:3],
		},
		testCase{
			"Less: all",
			less,
			4,
			models,
		},
		testCase{
			"Greater: none",
			greater,
			4,
			[]*indexedPrimativesModel{},
		},
		testCase{
			"Greater: middle",
			greater,
			1,
			models[3:5],
		},
		testCase{
			"Greater: all",
			greater,
			-5,
			models,
		},
		testCase{
			"Less Or Equal: none",
			lessOrEqual,
			-5,
			[]*indexedPrimativesModel{},
		},
		testCase{
			"Less Or Equal: middle",
			lessOrEqual,
			1,
			models[0:3],
		},
		testCase{
			"Less Or Equal: all",
			lessOrEqual,
			3,
			models,
		},
		testCase{
			"Greater Or Equal: none",
			greaterOrEqual,
			4,
			[]*indexedPrimativesModel{},
		},
		testCase{
			"Greater Or Equal: middle",
			greaterOrEqual,
			2,
			models[3:5],
		},
		testCase{
			"Greater Or Equal: all",
			greaterOrEqual,
			-4,
			models,
		},
	}

	for i, tc := range testCases {
		fmt.Printf("testing numeric case %d (%s)...\n", i, tc.name)
		if got, err := filterModels(models, "Int", tc.fType, tc.fVal, indexNumeric); err != nil {
			t.Error(err)
		} else {
			if eql, msg := looseEquals(tc.expected, got); !eql {
				t.Errorf("Test failed on iteration %d (%s)\nExpected: %v\nGot %v\n%s\n", i, tc.name, tc.expected, got, msg)
			}
		}
	}
}

// Test our internal model filter with boolean type indexes
func TestInternalFilterModelsBoolean(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	models, _ := newIndexedPrimativesModels(2)
	models[0].Bool = false
	models[1].Bool = true

	type testCase struct {
		name     string
		fType    filterType
		fVal     interface{}
		expected []*indexedPrimativesModel
	}
	testCases := []testCase{
		testCase{
			"Equal",
			equal,
			true,
			models[1:2],
		},
		testCase{
			"Not Equal",
			notEqual,
			true,
			models[0:1],
		},
		testCase{
			"Less: none",
			less,
			false,
			[]*indexedPrimativesModel{},
		},
		testCase{
			"Less: middle",
			less,
			true,
			models[0:1],
		},
		testCase{
			"Greater: none",
			greater,
			true,
			[]*indexedPrimativesModel{},
		},
		testCase{
			"Greater: middle",
			greater,
			false,
			models[1:2],
		},
		testCase{
			"Less Or Equal: middle",
			lessOrEqual,
			false,
			models[0:1],
		},
		testCase{
			"Less Or Equal: all",
			lessOrEqual,
			true,
			models,
		},
		testCase{
			"Greater Or Equal: middle",
			greaterOrEqual,
			true,
			models[1:2],
		},
		testCase{
			"Greater Or Equal: all",
			greaterOrEqual,
			false,
			models,
		},
	}

	for i, tc := range testCases {
		fmt.Printf("testing boolean case %d (%s)...\n", i, tc.name)
		if got, err := filterModels(models, "Bool", tc.fType, tc.fVal, indexBoolean); err != nil {
			t.Error(err)
		} else {
			if eql, msg := looseEquals(tc.expected, got); !eql {
				t.Errorf("Test failed on iteration %d (%s)\nExpected: %v\nGot %v\n%s\n", i, tc.name, tc.expected, got, msg)
			}
		}
	}
}

// Test our internal model filter with alpha (string) type indexes
func TestInternalFilterModelsAlpha(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	models, _ := newIndexedPrimativesModels(5)
	models[0].String = "b"
	models[1].String = "c"
	models[2].String = "c"
	models[3].String = "d"
	models[4].String = "e"

	type testCase struct {
		name     string
		fType    filterType
		fVal     interface{}
		expected []*indexedPrimativesModel
	}
	testCases := []testCase{
		testCase{
			"Equal",
			equal,
			"c",
			models[1:3],
		},
		testCase{
			"Not Equal",
			notEqual,
			"e",
			models[0:4],
		},
		testCase{
			"Less: none",
			less,
			"b",
			[]*indexedPrimativesModel{},
		},
		testCase{
			"Less: middle",
			less,
			"d",
			models[0:3],
		},
		testCase{
			"Less: all",
			less,
			"f",
			models,
		},
		testCase{
			"Greater: none",
			greater,
			"e",
			[]*indexedPrimativesModel{},
		},
		testCase{
			"Greater: middle",
			greater,
			"c",
			models[3:5],
		},
		testCase{
			"Greater: all",
			greater,
			"a",
			models,
		},
		testCase{
			"Less Or Equal: none",
			lessOrEqual,
			"a",
			[]*indexedPrimativesModel{},
		},
		testCase{
			"Less Or Equal: middle",
			lessOrEqual,
			"c",
			models[0:3],
		},
		testCase{
			"Less Or Equal: all",
			lessOrEqual,
			"e",
			models,
		},
		testCase{
			"Greater Or Equal: none",
			greaterOrEqual,
			"f",
			[]*indexedPrimativesModel{},
		},
		testCase{
			"Greater Or Equal: middle",
			greaterOrEqual,
			"d",
			models[3:5],
		},
		testCase{
			"Greater Or Equal: all",
			greaterOrEqual,
			"b",
			models,
		},
	}

	for i, tc := range testCases {
		fmt.Printf("testing alpha case %d (%s)...\n", i, tc.name)
		if got, err := filterModels(models, "String", tc.fType, tc.fVal, indexAlpha); err != nil {
			t.Error(err)
		} else {
			if eql, msg := looseEquals(tc.expected, got); !eql {
				t.Errorf("Test failed on iteration %d (%s)\nExpected: %v\nGot %v\n%s\n", i, tc.name, tc.expected, got, msg)
			}
		}
	}
}

// Some functions to be used by the builtin sort package
type ById []*indexedPrimativesModel

func (a ById) Len() int {
	return len(a)
}
func (a ById) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a ById) Less(i, j int) bool {
	return a[i].Id < a[j].Id
}

type ByInt []*indexedPrimativesModel

func (a ByInt) Len() int {
	return len(a)
}
func (a ByInt) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a ByInt) Less(i, j int) bool {
	return a[i].Int < a[j].Int
}

type ByBool []*indexedPrimativesModel

func (a ByBool) Len() int {
	return len(a)
}
func (a ByBool) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a ByBool) Less(i, j int) bool {
	return boolToInt(a[i].Bool) < boolToInt(a[j].Bool)
}

// NOTE: This implementation only sorts by the first letter of the string!
type ByString []*indexedPrimativesModel

func (a ByString) Len() int {
	return len(a)
}
func (a ByString) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a ByString) Less(i, j int) bool {
	return a[i].String < a[j].String
}

// orderModels uses the sort package to order the models by the
// given field and index type Assumes that indexNumeric corresponds
// to the Int field, indexBoolean corresponds to the Bool field,
// and indexAlpha corresponds to the String field.
// Returns a copy of the model slice, so the original is unchanged.
func orderModels(models []*indexedPrimativesModel, fieldName string) []*indexedPrimativesModel {
	results := make([]*indexedPrimativesModel, len(models))
	copy(results, models)
	switch fieldName {
	case "Id":
		sort.Sort(ById(results))
	case "Int":
		sort.Sort(ByInt(results))
	case "Bool":
		sort.Sort(ByBool(results))
	case "String":
		sort.Sort(ByString(results))
	default:
		log.Fatalf("Don't know how to order by the field named %s\n", fieldName)
	}
	return results
}

// reverseModels reverses the order of the models slice. Returns a copy,
// so the original is unchanged.
func reverseModels(models []*indexedPrimativesModel) []*indexedPrimativesModel {
	results := make([]*indexedPrimativesModel, len(models))
	copy(results, models)
	for i, j := 0, len(results)-1; i <= j; i, j = i+1, j-1 {
		results[i], results[j] = results[j], results[i]
	}
	return results
}

// Test our internal orderModels function for the Id field
func TestInternalOrderModelsNumeric(t *testing.T) {
	expected, _ := newIndexedPrimativesModels(4)
	expected[0].Int = 0
	expected[1].Int = 1
	expected[2].Int = 2
	expected[3].Int = 3
	models := []*indexedPrimativesModel{expected[3], expected[0], expected[2], expected[1]}

	got := orderModels(models, "Int")
	if eql, msg := looseEquals(expected, got); !eql {
		t.Errorf("Expected: %v\nGot %v\n%s\n", expected, got, msg)
	}
}

// Test our internal orderModels function for the Int field
func TestInternalOrderModelsId(t *testing.T) {
	expected, _ := newIndexedPrimativesModels(4)
	expected[0].Id = "a"
	expected[1].Id = "b"
	expected[2].Id = "c"
	expected[3].Id = "d"
	models := []*indexedPrimativesModel{expected[3], expected[0], expected[2], expected[1]}

	got := orderModels(models, "Id")
	if eql, msg := looseEquals(expected, got); !eql {
		t.Errorf("Expected: %v\nGot %v\n%s\n", expected, got, msg)
	}
}

// Test our internal orderModels function for the Bool field
func TestInternalOrderModelsBoolean(t *testing.T) {
	expected, _ := newIndexedPrimativesModels(2)
	expected[0].Bool = false
	expected[1].Bool = true
	models := []*indexedPrimativesModel{expected[1], expected[0]}
	got := orderModels(models, "Bool")
	if eql, msg := looseEquals(expected, got); !eql {
		t.Errorf("Expected: %v\nGot %v\n%s\n", expected, got, msg)
	}
}

// Test our internal orderModels function for the String field
func TestInternalOrderModelsAlpha(t *testing.T) {
	expected, _ := newIndexedPrimativesModels(4)
	expected[0].String = "a"
	expected[1].String = "b"
	expected[2].String = "c"
	expected[3].String = "d"
	models := []*indexedPrimativesModel{expected[3], expected[0], expected[2], expected[1]}

	got := orderModels(models, "String")
	if eql, msg := looseEquals(expected, got); !eql {
		t.Errorf("Expected: %v\nGot %v\n%s\n", expected, got, msg)
	}
}

// Test our internal reverseModels function for numeric index types
func TestInternalReverseModelsNumeric(t *testing.T) {
	models, _ := newIndexedPrimativesModels(4)
	models[0].Int = 0
	models[1].Int = 1
	models[2].Int = 2
	models[3].Int = 3
	expected := []*indexedPrimativesModel{models[3], models[2], models[1], models[0]}

	got := reverseModels(models)
	if eql, msg := looseEquals(expected, got); !eql {
		t.Errorf("Expected: %v\nGot %v\n%s\n", expected, got, msg)
	}
}

// Test our internal reverseModels function for boolean index types
func TestInternalReverseModelsBoolean(t *testing.T) {
	models, _ := newIndexedPrimativesModels(2)
	models[0].Bool = false
	models[1].Bool = true
	expected := []*indexedPrimativesModel{models[1], models[0]}

	got := reverseModels(models)
	if eql, msg := looseEquals(expected, got); !eql {
		t.Errorf("Expected: %v\nGot %v\n%s\n", expected, got, msg)
	}
}

// Test our internal reverseModels function for alpha index types
func TestInternalReverseModelsAlpha(t *testing.T) {
	models, _ := newIndexedPrimativesModels(4)
	models[0].String = "a"
	models[1].String = "b"
	models[2].String = "c"
	models[3].String = "d"
	expected := []*indexedPrimativesModel{models[3], models[2], models[1], models[0]}

	got := reverseModels(models)
	if eql, msg := looseEquals(expected, got); !eql {
		t.Errorf("Expected: %v\nGot %v\n%s\n", expected, got, msg)
	}
}
