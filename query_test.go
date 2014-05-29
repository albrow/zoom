// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File query_test.go tests the query abstraction (query.go)

package zoom

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

func TestQueryAll(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms, err := createFullModels(5)
	if err != nil {
		t.Error(err)
	}
	if err := MSave(Models(ms)); err != nil {
		t.Error(err)
	}

	q := NewQuery("indexedPrimativesModel")
	testQuery(t, q, ms)
}

func TestQueryOrderNumeric(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// create models which we will try to sort
	models, err := createFullModels(10)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// use reflection to get all the numeric field names for *indexedPrimativesModel
	fieldNames := make([]string, 0)
	typ := reflect.TypeOf(indexedPrimativesModel{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.Anonymous {
			if typeIsNumeric(field.Type) {
				fieldNames = append(fieldNames, field.Name)
			}
		}
	}

	// create some test queries to sort the models by all possible numeric fields
	for _, fieldName := range fieldNames {
		q1 := NewQuery("indexedPrimativesModel").Order(fieldName)
		testQuery(t, q1, models)
		q2 := NewQuery("indexedPrimativesModel").Order("-" + fieldName)
		testQuery(t, q2, models)
	}
}

func TestQueryOrderBoolean(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// create models which we will try to sort
	models, err := createFullModels(10)
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
	models, err := createFullModels(10)
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

func TestQueryFilterNumeric(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// create models which we will try to filter
	models, err := createFullModels(10)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// use reflection to get all the numeric field names for *indexedPrimativesModel
	fieldNames := make([]string, 0)
	filterValues := make([]interface{}, 0)
	typ := reflect.TypeOf(indexedPrimativesModel{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.Anonymous {
			if typeIsNumeric(field.Type) {
				fieldNames = append(fieldNames, field.Name)
				fv := reflect.ValueOf(5).Convert(field.Type).Interface()
				filterValues = append(filterValues, fv)
			}
		}
	}

	// create some test queries to filter the models using all possible numeric filters
	operators := []string{"=", "!=", ">", ">=", "<", "<="}
	for i, fieldName := range fieldNames {
		val := filterValues[i]
		for _, op := range operators {
			q := NewQuery("indexedPrimativesModel")
			q.Filter(fieldName+" "+op, val)
			testQuery(t, q, models)
		}
	}
}

func TestQueryFilterBoolean(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// create models which we will try to filter
	models, err := createFullModels(10)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// create some test queries to filter the models
	operators := []string{"=", "!=", ">", ">=", "<", "<="}
	for _, op := range operators {
		q1 := NewQuery("indexedPrimativesModel").Filter("Bool "+op, true)
		testQuery(t, q1, models)
		q2 := NewQuery("indexedPrimativesModel").Filter("Bool "+op, false)
		testQuery(t, q2, models)
	}
}

func TestQueryFilterAlpha(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// create models which we will try to filter
	// we create two with each letter of the alphabet so
	// we can test what happens when there are multiple models
	// with the same letter (the same String value)
	models, err := createFullModels(26 * 2)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// create some test queries to filter the models
	operators := []string{"=", "!=", ">", ">=", "<", "<="}
	for _, op := range operators {
		q := NewQuery("indexedPrimativesModel").Filter("String "+op, "k")
		testQuery(t, q, models)
	}
}

func TestQueryDoubleFilters(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// create enough models so that there are some
	// collisions
	models, err := createFullModels(100)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// create some test queries to filter the models
	fieldNames := []string{"Int", "Bool", "String"}
	filterValues := []interface{}{5, true, "k"}
	operators := []string{"=", "!=", ">", ">=", "<", "<="}
	for i, f1 := range fieldNames {
		v1 := filterValues[i]
		for j, f2 := range fieldNames {
			v2 := filterValues[j]
			for _, o1 := range operators {
				for _, o2 := range operators {
					if f1 == f2 && o1 == o2 {
						// no sense in doing the same filter twice
						continue
					}
					q := NewQuery("indexedPrimativesModel").Filter(f1+" "+o1, v1).Filter(f2+" "+o2, v2)
					testQuery(t, q, models)
				}
			}
		}
	}
}

func TestQueryLimitAndOffset(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	models, err := createFullModels(10)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	limits := []uint{0, 1, 9, 10}
	offsets := []uint{0, 1, 9, 10}
	for _, l := range limits {
		for _, o := range offsets {
			q := NewQuery("indexedPrimativesModel").Order("Int").Limit(l).Offset(o)
			testQuery(t, q, models)
		}
	}
}

func TestQueryCombos(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	models, err := createFullModels(50)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// use one numeric, one bool, and one string field
	// fieldNames := []string{"Int", "Bool", "String"}
	// filterValues := []interface{}{5, true, "k"}
	// TODO: re-add string values when alpha implementation is fixed
	fieldNames := []string{"Int", "Bool", "String"}
	filterValues := []interface{}{5, true, "k"}
	limits := []uint{0, 1, 5, 9, 10}
	offsets := []uint{0, 1, 5, 9, 10}

	// iterate and create queries for all possible combinations of filters and orders
	// with the fields and values specified above
	operators := []string{"=", "!=", ">", ">=", "<", "<="}
	for i, f1 := range fieldNames {
		val := filterValues[i]
		for _, op := range operators {
			for _, f2 := range fieldNames {
				for _, offset := range offsets {
					for _, limit := range limits {
						q := NewQuery("indexedPrimativesModel")
						q.Filter(f1+" "+op, val).Order(f2).Limit(limit).Offset(offset)
						testQuery(t, q, models)
						q = NewQuery("indexedPrimativesModel")
						q.Filter(f1+" "+op, val).Order("-" + f2).Limit(limit).Offset(offset)
						testQuery(t, q, models)
					}
				}
			}
		}
	}
}

// create a number of models with all fields filled out.
// we will use these to test a lot of different queries.
// on each iteration from i=0 to num-1 a model is created with:
// 	- numeric fields set to i (typecasted if needed)
//		- bool field set to true if i%2 = 0 and false otherwise
//		- string field set to the value at index i%26 from a slice of all lowercase letters a-z
//			(i.e. i=0 corresponds to a, i=26 corresponds to z, and i=27 corresponds to a)
func createFullModels(num int) ([]*indexedPrimativesModel, error) {
	// alphabet holds every letter from a to z
	alphabet := []string{}
	for c := 'a'; c < 'z'+1; c++ {
		alphabet = append(alphabet, string(c))
	}
	bools := []bool{true, false}

	ms := []*indexedPrimativesModel{}
	for i := 0; i < num; i++ {
		m := &indexedPrimativesModel{
			Uint:    uint(i),
			Uint8:   uint8(i),
			Uint16:  uint16(i),
			Uint32:  uint32(i),
			Uint64:  uint64(i),
			Int:     i,
			Int8:    int8(i),
			Int16:   int16(i),
			Int32:   int32(i),
			Int64:   int64(i),
			Float32: float32(i),
			Float64: float64(i),
			Byte:    byte(i),
			Rune:    rune(i),
			String:  alphabet[i%len(alphabet)],
			Bool:    bools[i%len(bools)],
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
		t.FailNow()
	}
	stable := !modelsContainDuplicatesForField(models, q.order.fieldName)
	testQueryScan(t, q, expected, stable)
	testQueryIdsOnly(t, q, expected, stable)
	testQueryCount(t, q, expected)
}

func testQueryScan(t *testing.T, q *Query, expected []*indexedPrimativesModel, stable bool) {
	got := make([]*indexedPrimativesModel, 0)
	if err := q.Scan(&got); err != nil {
		t.Error(err)
		t.FailNow()
	}

	if q.order.fieldName != "" && !stable {
		// since redis sorts are unstable, varying results from the query should
		// all be considered correct. In this case, the best we can do is make sure
		// that the length is rught and that the models were sorted correctly.
		if len(expected) != len(got) {
			t.Errorf("Length of results was not correct. Expected %d but got %d\n\tfor query %s", len(expected), len(got), q)
			t.FailNow()
		}

		// check that the models are in the correct order (if applicable)
		reverse := false
		if q.order.orderType == descending {
			reverse = true
		}
		if sorted, fields, err := modelsAreSortedByField(got, q.order.fieldName, reverse); err != nil {
			t.Error(err)
			t.FailNow()
		} else if !sorted {
			t.Errorf("models were not in the correct order. %v \n\tfor the query %s", fields, q)
			t.FailNow()
		}
	} else {
		// we can expect a stable sort since there are no duplicates. We can test
		// the order of the models exactly.
		orderMatters := false
		if q.order.fieldName != "" {
			orderMatters = true
		}
		match := compareModelSlices(t, expected, got, orderMatters)
		if !match {
			t.Errorf("\n\ttestQueryScan failed for query %s", q)
			if eJS, err := json.Marshal(expected); err != nil {
				t.Error(err)
			} else {
				t.Errorf("Expected JSON:\n\t%s", eJS)
			}
			if gJS, err := json.Marshal(got); err != nil {
				t.Error(err)
			} else {
				t.Errorf("Got JSON:\n\t%s", gJS)
			}
			t.FailNow()
		}
	}
}

func testQueryCount(t *testing.T, q *Query, expectedModels []*indexedPrimativesModel) {
	expected := len(expectedModels)
	if got, err := q.Count(); err != nil {
		t.Error(err)
		t.FailNow()
	} else if got != expected {
		t.Errorf("testQueryCount failed for query %s. Expected %d but got %d.", q, expected, got)
		t.FailNow()
	}
}

func testQueryIdsOnly(t *testing.T, q *Query, expectedModels []*indexedPrimativesModel, stable bool) {
	expected := modelIds(Models(expectedModels))
	got, err := q.IdsOnly()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if stable {
		// for queries with stable sorts we should expect an exact order
		if q.order.fieldName == "" {
			if match, msg := compareAsStringSet(expected, got); !match {
				t.Errorf("%s\ntestQueryIdsOnly failed for query %s.", msg, q)
				t.FailNow()
			}
		} else {
			if !reflect.DeepEqual(expected, got) {
				t.Errorf("testQueryIdsOnly failed for query %s.\nexpected %v but got %v", q, expected, got)
				t.FailNow()
			}
		}
	} else {
		// for queries with unstable sorts, we can only check that the length is correct :(
		if len(expected) != len(got) {
			t.Errorf("testQueryIdsOnly failed for query %s.\nLengths did not match. Expected %s but got %s.", q, len(expected), len(got))
		}
	}
}

func expectedResultsForQuery(q *Query, models []*indexedPrimativesModel) ([]*indexedPrimativesModel, error) {
	expected := make([]*indexedPrimativesModel, len(models))
	copy(expected, models)

	// apply filters
	for _, f := range q.filters {
		fmodels, err := filterModels(models, f.fieldName, f.filterType, f.filterValue.Interface(), f.indexType)
		if err != nil {
			return nil, err
		}
		expected = orderedIntersectModels(fmodels, expected)
	}

	if q.order.fieldName != "" && !modelsContainDuplicatesForField(expected, q.order.fieldName) {
		expected = sortModels(expected, q.order.fieldName, q.order.orderType == descending)
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
				fieldVal := reflect.ValueOf(m).Elem().FieldByName(fieldName).Convert(reflect.TypeOf(0.0)).Float()
				filterVal := reflect.ValueOf(fVal).Convert(reflect.TypeOf(0.0)).Float()
				return fieldVal == filterVal, nil
			}
		case notEqual:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldVal := reflect.ValueOf(m).Elem().FieldByName(fieldName).Convert(reflect.TypeOf(0.0)).Float()
				filterVal := reflect.ValueOf(fVal).Convert(reflect.TypeOf(0.0)).Float()
				return fieldVal != filterVal, nil
			}
		case greater:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldVal := reflect.ValueOf(m).Elem().FieldByName(fieldName).Convert(reflect.TypeOf(0.0)).Float()
				filterVal := reflect.ValueOf(fVal).Convert(reflect.TypeOf(0.0)).Float()
				return fieldVal > filterVal, nil
			}
		case less:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldVal := reflect.ValueOf(m).Elem().FieldByName(fieldName).Convert(reflect.TypeOf(0.0)).Float()
				filterVal := reflect.ValueOf(fVal).Convert(reflect.TypeOf(0.0)).Float()
				return fieldVal < filterVal, nil
			}
		case greaterOrEqual:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldVal := reflect.ValueOf(m).Elem().FieldByName(fieldName).Convert(reflect.TypeOf(0.0)).Float()
				filterVal := reflect.ValueOf(fVal).Convert(reflect.TypeOf(0.0)).Float()
				return fieldVal >= filterVal, nil
			}
		case lessOrEqual:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldVal := reflect.ValueOf(m).Elem().FieldByName(fieldName).Convert(reflect.TypeOf(0.0)).Float()
				filterVal := reflect.ValueOf(fVal).Convert(reflect.TypeOf(0.0)).Float()
				return fieldVal <= filterVal, nil
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
				return fieldString > valueString, nil
			}
		case less:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldString := reflect.ValueOf(m).Elem().FieldByName(fieldName).String()
				valueString := reflect.ValueOf(fVal).String()
				return fieldString < valueString, nil
			}
		case greaterOrEqual:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldString := reflect.ValueOf(m).Elem().FieldByName(fieldName).String()
				valueString := reflect.ValueOf(fVal).String()
				return fieldString >= valueString, nil
			}
		case lessOrEqual:
			s = func(m *indexedPrimativesModel) (bool, error) {
				fieldString := reflect.ValueOf(m).Elem().FieldByName(fieldName).String()
				valueString := reflect.ValueOf(fVal).String()
				return fieldString <= valueString, nil
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
		if got, err := filterModels(models, "String", tc.fType, tc.fVal, indexAlpha); err != nil {
			t.Error(err)
		} else {
			if eql, msg := looseEquals(tc.expected, got); !eql {
				t.Errorf("Test failed on iteration %d (%s)\nExpected: %v\nGot %v\n%s\n", i, tc.name, tc.expected, got, msg)
			}
		}
	}
}

// sorting interfaces. UGH. This is incredibly annoying, but this is the best
// way I could find of doing it. based on the http://golang.org/pkg/sort/
// example

type lessFunc func(m1, m2 *indexedPrimativesModel) bool

// multiSorter implements the Sort interface, sorting the models within.
type multiSorter struct {
	models []*indexedPrimativesModel
	less   []lessFunc
}

// Sort sorts the argument slice according to the less functions passed to OrderedBy.
func (ms *multiSorter) Sort(models []*indexedPrimativesModel) {
	ms.models = models
	sort.Sort(ms)
}

// OrderedBy returns a Sorter that sorts using the less functions, in order.
// Call its Sort method to sort the data.
func OrderedBy(less ...lessFunc) *multiSorter {
	return &multiSorter{
		less: less,
	}
}

// Len is part of sort.Interface.
func (ms *multiSorter) Len() int {
	return len(ms.models)
}

// Swap is part of sort.Interface.
func (ms *multiSorter) Swap(i, j int) {
	ms.models[i], ms.models[j] = ms.models[j], ms.models[i]
}

// Less is part of sort.Interface. It is implemented by looping along the
// less functions until it finds a comparison that is either Less or
// !Less. Note that it can call the less functions twice per call. We
// could change the functions to return -1, 0, 1 and reduce the
// number of calls for greater efficiency: an exercise for the reader.
func (ms *multiSorter) Less(i, j int) bool {
	p, q := ms.models[i], ms.models[j]
	// Try all but the last comparison.
	var k int
	for k = 0; k < len(ms.less)-1; k++ {
		less := ms.less[k]
		switch {
		case less(p, q):
			// p < q, so we have a decision.
			return true
		case less(q, p):
			// p > q, so we have a decision.
			return false
		}
		// p == q; try the next comparison.
	}
	// All comparisons to here said "equal", so just return whatever
	// the final comparison reports.
	return ms.less[k](p, q)
}

// Closures that order the Model structure.
var lessFuncs map[string]lessFunc = map[string]lessFunc{
	"Uint": func(m1, m2 *indexedPrimativesModel) bool {
		return m1.Uint < m2.Uint
	},
	"Uint8": func(m1, m2 *indexedPrimativesModel) bool {
		return m1.Uint8 < m2.Uint8
	},
	"Uint16": func(m1, m2 *indexedPrimativesModel) bool {
		return m1.Uint16 < m2.Uint16
	},
	"Uint32": func(m1, m2 *indexedPrimativesModel) bool {
		return m1.Uint32 < m2.Uint32
	},
	"Uint64": func(m1, m2 *indexedPrimativesModel) bool {
		return m1.Uint64 < m2.Uint64
	},
	"Int": func(m1, m2 *indexedPrimativesModel) bool {
		return m1.Int < m2.Int
	},
	"Int8": func(m1, m2 *indexedPrimativesModel) bool {
		return m1.Int8 < m2.Int8
	},
	"Int16": func(m1, m2 *indexedPrimativesModel) bool {
		return m1.Int16 < m2.Int16
	},
	"Int32": func(m1, m2 *indexedPrimativesModel) bool {
		return m1.Int32 < m2.Int32
	},
	"Int64": func(m1, m2 *indexedPrimativesModel) bool {
		return m1.Int64 < m2.Int64
	},
	"Float32": func(m1, m2 *indexedPrimativesModel) bool {
		return m1.Float32 < m2.Float32
	},
	"Float64": func(m1, m2 *indexedPrimativesModel) bool {
		return m1.Float64 < m2.Float64
	},
	"Byte": func(m1, m2 *indexedPrimativesModel) bool {
		return m1.Byte < m2.Byte
	},
	"Rune": func(m1, m2 *indexedPrimativesModel) bool {
		return m1.Rune < m2.Rune
	},
	"String": func(m1, m2 *indexedPrimativesModel) bool {
		return m1.String < m2.String
	},
	"Bool": func(m1, m2 *indexedPrimativesModel) bool {
		return m1.Bool == false && m2.Bool == true
	},
}

// sortModels sorts the set of models by the given fieldName. Returns a copy,
// so the original is unchanged.
func sortModels(models []*indexedPrimativesModel, fieldName string, reverse bool) []*indexedPrimativesModel {
	results := make([]*indexedPrimativesModel, len(models))
	copy(results, models)
	OrderedBy(lessFuncs[fieldName]).Sort(results)
	if reverse {
		return reverseModels(results)
	} else {
		return results
	}
}

// returns true iff the given models are sorted by field
// redis sorts can be unstable, so here it is considered sorted even if it is unstable.
// if revers is true, will check if the models are sorted in reverse (descending) order.
func modelsAreSortedByField(models []*indexedPrimativesModel, fieldName string, reverse bool) (bool, []interface{}, error) {
	ms := make([]*indexedPrimativesModel, len(models))
	copy(ms, models)
	typ, err := getRegisteredTypeFromName("indexedPrimativesModel")
	if err != nil {
		return false, nil, err
	}
	field, found := typ.Elem().FieldByName(fieldName)
	if !found {
		err := fmt.Errorf("Could not find field named %s in type %s", fieldName, typ.String())
		return false, nil, err
	}
	fType := field.Type
	if typeIsNumeric(fType) {
		return modelsAreSortedByNumericField(ms, fieldName, reverse)
	} else if typeIsBool(fType) {
		return modelsAreSortedByBooleanField(ms, fieldName, reverse)
	} else if typeIsString(fType) {
		return modelsAreSortedByStringField(ms, fieldName, reverse)
	}
	return false, nil, fmt.Errorf("Don't know how to classify field type %s! Was not numeric, bool, or string.", fType.String())
}

// Note: it's okay to mutate models here because it was copied in the previous function.
func modelsAreSortedByNumericField(models []*indexedPrimativesModel, fieldName string, reverse bool) (bool, []interface{}, error) {
	fieldsAsFloats := make([]float64, len(models))
	fieldsAsInterfaces := make([]interface{}, len(models))
	floatType := reflect.TypeOf(0.0)
	for i, m := range models {
		val := reflect.ValueOf(m)
		fVal := val.Elem().FieldByName(fieldName)
		fValFloat := fVal.Convert(floatType)
		fFloat := fValFloat.Float()
		fieldsAsFloats[i] = fFloat
		fieldsAsInterfaces[i] = fValFloat.Interface()
	}
	if reverse {
		for i, j := 0, len(fieldsAsFloats)-1; i <= j; i, j = i+1, j-1 {
			fieldsAsFloats[i], fieldsAsFloats[j] = fieldsAsFloats[j], fieldsAsFloats[i]
		}
	}
	if sort.Float64sAreSorted(fieldsAsFloats) {
		return true, fieldsAsInterfaces, nil
	} else {
		return false, fieldsAsInterfaces, nil
	}
}

// Note: it's okay to mutate models here because it was copied in the previous function.
func modelsAreSortedByBooleanField(models []*indexedPrimativesModel, fieldName string, reverse bool) (bool, []interface{}, error) {
	// let false = 0, true = 1
	fieldsAsInts := make([]int, len(models))
	fieldsAsInterfaces := make([]interface{}, len(models))
	for i, m := range models {
		if m.Bool {
			fieldsAsInts[i] = 1
			fieldsAsInterfaces[i] = 1
		} else {
			fieldsAsInts[i] = 0
			fieldsAsInterfaces[i] = 0
		}
	}
	if reverse {
		for i, j := 0, len(fieldsAsInts)-1; i <= j; i, j = i+1, j-1 {
			fieldsAsInts[i], fieldsAsInts[j] = fieldsAsInts[j], fieldsAsInts[i]
		}
	}
	if sort.IntsAreSorted(fieldsAsInts) {
		return true, fieldsAsInterfaces, nil
	} else {
		return false, fieldsAsInterfaces, nil
	}
}

// Note: it's okay to mutate models here because it was copied in the previous function.
func modelsAreSortedByStringField(models []*indexedPrimativesModel, fieldName string, reverse bool) (bool, []interface{}, error) {
	fieldsAsStrings := make([]string, len(models))
	fieldsAsInterfaces := make([]interface{}, len(models))
	for i, m := range models {
		fieldsAsStrings[i] = m.String
		fieldsAsInterfaces[i] = m.String
	}
	if reverse {
		for i, j := 0, len(fieldsAsStrings)-1; i <= j; i, j = i+1, j-1 {
			fieldsAsStrings[i], fieldsAsStrings[j] = fieldsAsStrings[j], fieldsAsStrings[i]
		}
	}
	if sort.StringsAreSorted(fieldsAsStrings) {
		return true, fieldsAsInterfaces, nil
	} else {
		return false, fieldsAsInterfaces, nil
	}
}

// Test our model sort implementation for numeric fields
func TestInternalSortModelsNumeric(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// make models which are sorted
	presorted, _ := createFullModels(10)
	models := shuffleModels(presorted)
	fieldNames := []string{"Uint", "Uint8", "Uint16", "Uint32", "Uint64", "Int", "Int8", "Int16", "Int32", "Int64", "Float32", "Float64", "Byte", "Rune"}
	for _, fn := range fieldNames {
		asc := sortModels(models, fn, false)
		if sorted, _, err := modelsAreSortedByField(asc, fn, false); err != nil {
			t.Error(err)
		} else if !sorted {
			t.Errorf("Models were not correctly sorted by %s", fn)
		}
		des := sortModels(models, fn, true)
		if sorted, _, err := modelsAreSortedByField(des, fn, true); err != nil {
			t.Error(err)
		} else if !sorted {
			t.Errorf("Models were not correctly sorted by -%s", fn)
		}
	}
}

func TestInternalSortModelsAlpha(t *testing.T) {
	testingSetUp()
	defer testingTearDown()
	presorted, _ := createFullModels(10)
	models := shuffleModels(presorted)

	asc := sortModels(models, "String", false)
	if sorted, _, err := modelsAreSortedByField(asc, "String", false); err != nil {
		t.Error(err)
	} else if !sorted {
		t.Error("Models were not correctly sorted by String")
	}
	des := sortModels(models, "String", true)
	if sorted, _, err := modelsAreSortedByField(des, "String", true); err != nil {
		t.Error(err)
	} else if !sorted {
		t.Error("Models were not correctly sorted by -String")
	}
}

func TestInternalSortModelsBoolean(t *testing.T) {
	// NOTE: we only get to create two models here if we want a stable sort.
	// To rule out the possibility of the two models being sorted by chance (50%!),
	// we repeat the test 10 times.
	for i := 0; i < 10; i++ {
		testingSetUp()
		presorted, _ := createFullModels(2)
		models := shuffleModels(presorted)

		asc := sortModels(models, "Bool", false)
		if sorted, _, err := modelsAreSortedByField(asc, "Bool", false); err != nil {
			t.Error(err)
		} else if !sorted {
			t.Error("Models were not correctly sorted by Bool")
		}
		des := sortModels(models, "Bool", true)
		if sorted, _, err := modelsAreSortedByField(des, "Bool", true); err != nil {
			t.Error(err)
		} else if !sorted {
			t.Error("Models were not correctly sorted by -Bool")
		}
		testingTearDown()
	}
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

// randomizes the order of models. Good for testing sorts
func shuffleModels(models []*indexedPrimativesModel) []*indexedPrimativesModel {
	results := make([]*indexedPrimativesModel, len(models))
	perm := rand.Perm(len(models))
	for i, v := range perm {
		results[v] = models[i]
	}
	return results
}

// returns true iff the set of models contains duplicate values for the given field name
func modelsContainDuplicatesForField(models []*indexedPrimativesModel, fieldName string) bool {
	prev := []interface{}{}
	for _, m := range models {
		mVal := reflect.ValueOf(m).Elem()
		fVal := mVal.FieldByName(fieldName)
		if !fVal.IsValid() {
			continue
		}
		val := fVal.Interface()
		for _, p := range prev {
			if reflect.DeepEqual(p, val) {
				return true
			}
		}
		prev = append(prev, val)
	}
	return false
}

func TestInternalModelsContainDuplicates(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// should contain no duplicates
	models, _ := createFullModels(2)
	expecteds := map[string]bool{
		"Bool":   false,
		"Int":    false,
		"String": false,
	}
	for fn, expected := range expecteds {
		got := modelsContainDuplicatesForField(models, fn)
		if expected != got {
			t.Errorf("Duplicate check was incorrect. Expected %v but got %v", expected, got)
		}
	}

	// only Bool fields should be duplicated
	models, _ = createFullModels(3)
	expecteds = map[string]bool{
		"Bool":   true,
		"Int":    false,
		"String": false,
	}
	for fn, expected := range expecteds {
		got := modelsContainDuplicatesForField(models, fn)
		if expected != got {
			t.Errorf("Duplicate check was incorrect. Expected %v but got %v", expected, got)
		}
	}

	// only Bool and String fields should be duplicated
	models, _ = createFullModels(27)
	expecteds = map[string]bool{
		"Bool":   true,
		"Int":    false,
		"String": true,
	}
	for fn, expected := range expecteds {
		got := modelsContainDuplicatesForField(models, fn)
		if expected != got {
			t.Errorf("Duplicate check was incorrect. Expected %v but got %v", expected, got)
		}
	}
}
