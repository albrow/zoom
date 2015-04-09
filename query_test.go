// Copyright 2014 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File query_test.go tests the query abstraction (query.go)

package zoom

import (
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

// There's a huge amount of test cases to cover above.
// Below is some code that makes it easier, but needs to be
// tested itself. Testing for correctness using a brute force
// approach (obviously slow compared to what Zoom is actually doing) is
// fine because the tests in this file will typically use only a handful
// of models. The brute force approach is also easier becuase you can apply
// query modifiers independently, in any order. (Whereas behind the scenes
// zoom actually does some clever optimization and changing any single paramater
// or modifier of the query could completely change the command sent to Redis).
// We're assuming that for all tests in this file, the indexedTestModel type will
// be used.

// testQuery compares the results of the Query run by Zoom with the results
// of a simpler implementation which doesn't touch the database. If the results match,
// then the query was correct and the test will pass. models should be an array of all
// the models which are being queried against.
func testQuery(t *testing.T, q *Query, models []*indexedTestModel) {
	expected := expectedResultsForQuery(q, models)
	testQueryRun(t, q, expected)
	testQueryIds(t, q, expected)
	testQueryCount(t, q, expected)
}

func testQueryRun(t *testing.T, q *Query, expected []*indexedTestModel) {
	got := make([]*indexedTestModel, 0)
	if err := q.Run(&got); err != nil {
		t.Error(err)
	}

	// TODO: fill this in
}

func testQueryCount(t *testing.T, q *Query, expectedModels []*indexedTestModel) {
	expected := len(expectedModels)
	if got, err := q.Count(); err != nil {
		t.Error(err)
	} else if got != expected {
		t.Errorf("testQueryCount failed for query %s. Expected %d but got %d.", q, expected, got)
	}
}

func testQueryIds(t *testing.T, q *Query, expectedModels []*indexedTestModel) {
	// TODO: fill this in
}

// expectedResultsForQuery returns the expected results for q on the given set of models.
// It computes the models that should be returned in-memory, without touching the database,
// and without the same optimizations that database queries have. It can be used to test for
// the correctness of database queries.
func expectedResultsForQuery(q *Query, models []*indexedTestModel) []*indexedTestModel {
	expected := make([]*indexedTestModel, len(models))
	copy(expected, models)

	// apply filters
	for _, filter := range q.filters {
		expected = orderedIntersectModels(applyFilter(expected, filter), expected)
	}

	// apply order (if applicable)
	if q.order.fieldName != "" {
		expected = applyOrder(expected, q.order)
	}

	// apply limit/offset
	expected = applyLimitAndOffset(expected, q.limit, q.offset)

	// apply includes/excludes
	if len(q.includes) > 0 {
		expected = applyIncludes(expected, q.includes)
	} else if len(q.excludes) > 0 {
		expected = applyExcludes(expected, q.excludes)
	}

	return expected
}

// applyFilter returns only the models which pass the filter criteria.
func applyFilter(models []*indexedTestModel, filter filter) []*indexedTestModel {

	var filterFunc func(m *indexedTestModel) bool

	switch filter.indexKind {

	case numericIndex:
		filterFunc = func(m *indexedTestModel) bool {
			fieldVal := reflect.ValueOf(m).Elem().FieldByName(filter.fieldName).Convert(reflect.TypeOf(0.0)).Float()
			filterVal := numericScore(filter.value)
			switch filter.kind {
			case equalFilter:
				return fieldVal == filterVal
			case notEqualFilter:
				return fieldVal != filterVal
			case greaterFilter:
				return fieldVal > filterVal
			case lessFilter:
				return fieldVal < filterVal
			case greaterOrEqualFilter:
				return fieldVal >= filterVal
			case lessOrEqualFilter:
				return fieldVal <= filterVal
			}
			return false
		}

	case booleanIndex:
		filterFunc = func(m *indexedTestModel) bool {
			fieldVal := reflect.ValueOf(m).Elem().FieldByName(filter.fieldName)
			filterVal := filter.value
			switch filter.kind {
			case equalFilter:
				return fieldVal.Bool() == filterVal.Bool()
			case notEqualFilter:
				return fieldVal.Bool() != filterVal.Bool()
			case greaterFilter:
				return boolScore(fieldVal) > boolScore(filterVal)
			case lessFilter:
				return boolScore(fieldVal) < boolScore(filterVal)
			case greaterOrEqualFilter:
				return boolScore(fieldVal) >= boolScore(filterVal)
			case lessOrEqualFilter:
				return boolScore(fieldVal) <= boolScore(filterVal)
			}
			return false
		}

	case stringIndex:
		filterFunc = func(m *indexedTestModel) bool {
			fieldVal := reflect.ValueOf(m).Elem().FieldByName(filter.fieldName).String()
			filterVal := filter.value.String()
			switch filter.kind {
			case equalFilter:
				return fieldVal == filterVal
			case notEqualFilter:
				return fieldVal != filterVal
			case greaterFilter:
				return fieldVal > filterVal
			case lessFilter:
				return fieldVal < filterVal
			case greaterOrEqualFilter:
				return fieldVal >= filterVal
			case lessOrEqualFilter:
				return fieldVal <= filterVal
			}
			return false
		}
	}

	return filterModels(models, filterFunc)
}

// filterModels returns only the models which return true when passed through
// the filter function.
func filterModels(models []*indexedTestModel, f func(*indexedTestModel) bool) []*indexedTestModel {
	results := make([]*indexedTestModel, 0)
	for _, m := range models {
		if f(m) {
			results = append(results, m)
		}
	}
	return results
}

// orderedIntersectModels intersects two model slices. The order
// will be preserved with respect to the first slice. (The first
// slice is used in the outer loop). The return value is a copy,
// so neither the first or second slice will be mutated.
func orderedIntersectModels(first []*indexedTestModel, second []*indexedTestModel) []*indexedTestModel {
	results := make([]*indexedTestModel, 0)
	memo := make(map[*indexedTestModel]struct{})
	for _, m := range second {
		memo[m] = struct{}{}
	}
	for _, m := range first {
		if _, found := memo[m]; found {
			results = append(results, m)
		}
	}
	return results
}

// TestApplyFilterNumeric tests our internal model filter (i.e. the applyFilters function)
// with numeric type indexes
func TestApplyFilterNumeric(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	models := createIndexedTestModels(5)
	models[0].Int = -4
	models[1].Int = 1
	models[2].Int = 1
	models[3].Int = 2
	models[4].Int = 3

	testCases := []struct {
		filterKind filterKind
		filterVal  interface{}
		expected   []*indexedTestModel
	}{
		{
			equalFilter,
			1,
			models[1:3],
		},
		{
			notEqualFilter,
			3,
			models[0:4],
		},
		{
			lessFilter,
			-4,
			[]*indexedTestModel{},
		},
		{
			lessFilter,
			2,
			models[0:3],
		},
		{
			lessFilter,
			4,
			models,
		},
		{
			greaterFilter,
			4,
			[]*indexedTestModel{},
		},
		{
			greaterFilter,
			1,
			models[3:5],
		},
		{
			greaterFilter,
			-5,
			models,
		},
		{
			lessOrEqualFilter,
			-5,
			[]*indexedTestModel{},
		},
		{
			lessOrEqualFilter,
			1,
			models[0:3],
		},
		{
			lessOrEqualFilter,
			3,
			models,
		},
		{
			greaterOrEqualFilter,
			4,
			[]*indexedTestModel{},
		},
		{
			greaterOrEqualFilter,
			2,
			models[3:5],
		},
		{
			greaterOrEqualFilter,
			-4,
			models,
		},
	}

	for i, tc := range testCases {
		filter := filter{
			fieldName: "Int",
			redisName: "Int",
			kind:      tc.filterKind,
			value:     reflect.ValueOf(tc.filterVal),
			indexKind: numericIndex,
		}
		got := applyFilter(models, filter)
		if !reflect.DeepEqual(tc.expected, got) {
			t.Errorf("Test failed on iteration %d: %s\nExpected: %#v\nGot:  %#v", i, filter, tc.expected, got)
		}
	}
}

// TestApplyFilterBool tests our internal model filter (i.e. the applyFilters function)
// with boolean type indexes
func TestApplyFilterBool(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	models := createIndexedTestModels(2)
	models[0].Bool = false
	models[1].Bool = true

	testCases := []struct {
		filterKind filterKind
		filterVal  interface{}
		expected   []*indexedTestModel
	}{
		{
			equalFilter,
			true,
			models[1:2],
		},
		{
			notEqualFilter,
			true,
			models[0:1],
		},
		{
			lessFilter,
			false,
			[]*indexedTestModel{},
		},
		{
			lessFilter,
			true,
			models[0:1],
		},
		{
			greaterFilter,
			true,
			[]*indexedTestModel{},
		},
		{
			greaterFilter,
			false,
			models[1:2],
		},
		{
			lessOrEqualFilter,
			false,
			models[0:1],
		},
		{
			lessOrEqualFilter,
			true,
			models,
		},
		{
			greaterOrEqualFilter,
			true,
			models[1:2],
		},
		{
			greaterOrEqualFilter,
			false,
			models,
		},
	}

	for i, tc := range testCases {
		filter := filter{
			fieldName: "Bool",
			redisName: "Bool",
			kind:      tc.filterKind,
			value:     reflect.ValueOf(tc.filterVal),
			indexKind: booleanIndex,
		}
		got := applyFilter(models, filter)
		if !reflect.DeepEqual(tc.expected, got) {
			t.Errorf("Test failed on iteration %d: %s\nExpected: %+v\nGot:  %+v", i, filter, tc.expected, got)
		}
	}
}

// TestApplyFilterString tests our internal model filter (i.e. the applyFilters function)
// with string type indexes
func TestApplyFilterString(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	models := createIndexedTestModels(5)
	models[0].String = "b"
	models[1].String = "c"
	models[2].String = "c"
	models[3].String = "d"
	models[4].String = "e"

	testCases := []struct {
		filterKind filterKind
		filterVal  interface{}
		expected   []*indexedTestModel
	}{
		{
			equalFilter,
			"c",
			models[1:3],
		},
		{
			notEqualFilter,
			"e",
			models[0:4],
		},
		{
			lessFilter,
			"b",
			[]*indexedTestModel{},
		},
		{
			lessFilter,
			"d",
			models[0:3],
		},
		{
			lessFilter,
			"f",
			models,
		},
		{
			greaterFilter,
			"e",
			[]*indexedTestModel{},
		},
		{
			greaterFilter,
			"c",
			models[3:5],
		},
		{
			greaterFilter,
			"a",
			models,
		},
		{
			lessOrEqualFilter,
			"a",
			[]*indexedTestModel{},
		},
		{
			lessOrEqualFilter,
			"c",
			models[0:3],
		},
		{
			lessOrEqualFilter,
			"e",
			models,
		},
		{
			greaterOrEqualFilter,
			"f",
			[]*indexedTestModel{},
		},
		{
			greaterOrEqualFilter,
			"d",
			models[3:5],
		},
		{
			greaterOrEqualFilter,
			"b",
			models,
		},
	}

	for i, tc := range testCases {
		filter := filter{
			fieldName: "String",
			redisName: "String",
			kind:      tc.filterKind,
			value:     reflect.ValueOf(tc.filterVal),
			indexKind: stringIndex,
		}
		got := applyFilter(models, filter)
		if !reflect.DeepEqual(tc.expected, got) {
			t.Errorf("Test failed on iteration %d: %s\nExpected: %+v\nGot:  %+v", i, filter, tc.expected, got)
		}
	}
}

// lessFunc is a function type that returns true if m1 should be considered less than m2.
// Typically a lessFunc will determine this by looking at a specific field value.
type lessFunc func(m1, m2 *indexedTestModel) bool

// modelSorter implements the Sort interface for sorting models. It uses lessFunc
// to implement the Less method.
type modelSorter struct {
	models   []*indexedTestModel
	lessFunc lessFunc
}

// newModelSorter creates and returns a modelSorter with the given models and fieldName.
func newModelSorter(models []*indexedTestModel, fieldName string) *modelSorter {
	// lessFuncs is a map of fieldName to a less function
	// that returns true iff m1.field < m2.field
	lessFuncs := map[string]lessFunc{
		"Int": func(m1, m2 *indexedTestModel) bool {
			if m1.Int == m2.Int {
				// Redis sorts by member if the scores are equal.
				// Which means all models have a secondary order: the Id field.
				return m1.Id < m2.Id
			}
			return m1.Int < m2.Int
		},
		"String": func(m1, m2 *indexedTestModel) bool {
			if m1.String == m2.String {
				return m1.Id < m2.Id
			}
			return m1.String < m2.String
		},
		"Bool": func(m1, m2 *indexedTestModel) bool {
			if m1.Bool == m2.Bool {
				return m1.Id < m2.Id
			}
			return m1.Bool == false && m2.Bool == true
		},
	}
	return &modelSorter{
		models:   models,
		lessFunc: lessFuncs[fieldName],
	}
}

// Len is part of sort.Interface.
func (sorter *modelSorter) Len() int {
	return len(sorter.models)
}

// Swap is part of sort.Interface.
func (sorter *modelSorter) Swap(i, j int) {
	sorter.models[i], sorter.models[j] = sorter.models[j], sorter.models[i]
}

// Less is part of sort.Interface. It is implemented by calling the modelSorter's
// lessFunc.
func (sorter *modelSorter) Less(i, j int) bool {
	return sorter.lessFunc(sorter.models[i], sorter.models[j])
}

// Sort returns the models sorted in ascending order by the sorter's fieldName.
func (sorter *modelSorter) Sort() []*indexedTestModel {
	sort.Sort(sorter)
	return sorter.models
}

// Sort returns the models sorted in descending order by the sorter's fieldName.
func (sorter *modelSorter) ReverseSort() []*indexedTestModel {
	sort.Sort(sort.Reverse(sorter))
	return sorter.models
}

// sortModels sorts the set of models by the given fieldName. Returns a copy,
// so the original is unchanged.
func sortModels(models []*indexedTestModel, fieldName string, orderKind orderKind) []*indexedTestModel {
	results := make([]*indexedTestModel, len(models))
	copy(results, models)
	sorter := newModelSorter(models, fieldName)
	if orderKind == ascendingOrder {
		return sorter.Sort()
	} else {
		return sorter.ReverseSort()
	}
}

func applyOrder(models []*indexedTestModel, order order) []*indexedTestModel {
	return sortModels(models, order.fieldName, order.kind)
}

func TestApplyOrderNumeric(t *testing.T) {
	expected := createIndexedTestModels(5)
	expected[0].Int = 1
	expected[1].Int = 2
	expected[2].Int = 3
	expected[3].Int = 4
	expected[4].Int = 5
	testApplyOrder(t, "Int", shuffleModels(expected), expected)
}

func TestApplyOrderString(t *testing.T) {
	expected := createIndexedTestModels(5)
	expected[0].String = "aaa"
	expected[1].String = "bbb"
	expected[2].String = "ccc"
	expected[3].String = "ddd"
	expected[4].String = "eee"
	testApplyOrder(t, "String", shuffleModels(expected), expected)
}

func TestApplyOrderBool(t *testing.T) {
	expected := createIndexedTestModels(2)
	expected[0].Bool = false
	expected[1].Bool = true
	// NOTE: there are only two models because there are only two bool values.
	// Using reverse instead of shuffle guarantees the shuffledModels arg is
	// in the wrong order.
	testApplyOrder(t, "Bool", reverseModels(expected), expected)
}

func testApplyOrder(t *testing.T, fieldName string, shuffledModels, expectedAscending []*indexedTestModel) {
	gotAscending := sortModels(shuffledModels, fieldName, ascendingOrder)
	if !reflect.DeepEqual(gotAscending, expectedAscending) {
		t.Errorf("Models were not sorted by %s in ascending order.\nExpected: %#v\nGot:  %#v", fieldName, expectedAscending, gotAscending)
	}
	gotDescending := sortModels(shuffledModels, fieldName, descendingOrder)
	expectedDescending := reverseModels(expectedAscending)
	if !reflect.DeepEqual(gotDescending, expectedDescending) {
		t.Errorf("Models were not sorted by %s in descending order.\nExpected: %#v\nGot:  %#v", fieldName, expectedDescending, gotDescending)
	}
}

// suffleModels randomizes the order of models. It returns a copy, so the original slice is
// left in tact. Good for testing sorts.
func shuffleModels(models []*indexedTestModel) []*indexedTestModel {
	results := make([]*indexedTestModel, len(models))
	perm := rand.Perm(len(models))
	for i, v := range perm {
		results[v] = models[i]
	}
	return results
}

// reverseModels reverses the order of models. It returns a copy, so the original slice is
// left in tact. Good for testing sorts.
func reverseModels(models []*indexedTestModel) []*indexedTestModel {
	results := make([]*indexedTestModel, len(models))
	copy(results, models)
	for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
		results[i], results[j] = results[j], results[i]
	}
	return results
}

// // modelsContainDuplicatesForField returns true iff the set of models contains
// // duplicate values for the given fieldName. That is, it returns true iff any value
// // of the given field occurs more than once.
// func modelsContainDuplicatesForField(models []*indexedTestModel, fieldName string) bool {
// 	// alreadySeen is a linked list which stores the field values we have already seen
// 	alreadySeen := list.New()
// 	for _, m := range models {
// 		mVal := reflect.ValueOf(m).Elem()
// 		fVal := mVal.FieldByName(fieldName)
// 		if !fVal.IsValid() {
// 			continue
// 		}
// 		val := fVal.Interface()
// 		for seen := alreadySeen.Front(); seen != nil; seen = seen.Next() {
// 			if reflect.DeepEqual(seen.Value, val) {
// 				// If we have already seen val, then there are duplicate field values
// 				return true
// 			}
// 		}
// 		alreadySeen.PushFront(val)
// 	}
// 	return false
// }

// func TestModelsContainDuplicatesForField(t *testing.T) {
// 	testingSetUp()
// 	defer testingTearDown()

// 	testCases := []struct {
// 		models    []*indexedTestModel
// 		expecteds map[string]bool
// 	}{
// 		// No duplicate field values
// 		{
// 			models: []*indexedTestModel{
// 				{Int: 0, Bool: false, String: "a"},
// 				{Int: 1, Bool: true, String: "b"},
// 			},
// 			expecteds: map[string]bool{
// 				"Int":    false,
// 				"Bool":   false,
// 				"String": false,
// 			},
// 		},
// 		// Duplicate Int field values
// 		{
// 			models: []*indexedTestModel{
// 				{Int: 5, Bool: true, String: "a"},
// 				{Int: 5, Bool: false, String: "b"},
// 			},
// 			expecteds: map[string]bool{
// 				"Int":    true,
// 				"Bool":   false,
// 				"String": false,
// 			},
// 		},
// 		// Duplicate Bool field values
// 		{
// 			models: []*indexedTestModel{
// 				{Int: 0, Bool: true, String: "a"},
// 				{Int: 1, Bool: true, String: "b"},
// 			},
// 			expecteds: map[string]bool{
// 				"Int":    false,
// 				"Bool":   true,
// 				"String": false,
// 			},
// 		},
// 		// Duplicate String field values
// 		{
// 			models: []*indexedTestModel{
// 				{Int: 0, Bool: true, String: "a"},
// 				{Int: 1, Bool: false, String: "a"},
// 			},
// 			expecteds: map[string]bool{
// 				"Int":    false,
// 				"Bool":   false,
// 				"String": true,
// 			},
// 		},
// 		// Multiple duplicate field values
// 		{
// 			models: []*indexedTestModel{
// 				{Int: 0, Bool: true, String: "a"},
// 				{Int: 0, Bool: false, String: "b"},
// 				{Int: 2, Bool: true, String: "c"},
// 				{Int: 3, Bool: false, String: "d"},
// 				{Int: 4, Bool: true, String: "e"},
// 				{Int: 5, Bool: false, String: "e"},
// 			},
// 			expecteds: map[string]bool{
// 				"Int":    true,
// 				"Bool":   true,
// 				"String": true,
// 			},
// 		},
// 	}
// 	for i, tc := range testCases {
// 		for fieldName, expected := range tc.expecteds {
// 			got := modelsContainDuplicatesForField(tc.models, fieldName)
// 			if got != expected {
// 				t.Errorf("Test case %d failed. Expected %v for %s field but got %v", i, expected, fieldName, got)
// 			}
// 		}
// 	}
// }

// applyIncludes applies includes to all models. That is, it zeroes out the fields which have field names
// not in includes. applyIncludes returns a copy, so the original slice is left intact.
func applyIncludes(models []*indexedTestModel, includes []string) []*indexedTestModel {
	results := make([]*indexedTestModel, len(models))
	for i, m := range models {
		result := &indexedTestModel{}
		resVal := reflect.ValueOf(result).Elem()
		origVal := reflect.ValueOf(m).Elem()
		for fieldIndex := 0; fieldIndex < origVal.NumField(); fieldIndex++ {
			fieldType := origVal.Type().Field(fieldIndex)
			if fieldType.Name == "DefaultData" {
				// DefaultData is a special case
				resVal.Field(fieldIndex).Set(origVal.Field(fieldIndex))
			}
			if stringSliceContains(includes, fieldType.Name) {
				resVal.Field(fieldIndex).Set(origVal.Field(fieldIndex))
			}
		}
		results[i] = result
	}
	return results
}

func TestApplyIncludes(t *testing.T) {
	models := []*indexedTestModel{
		{
			Int:    1,
			String: "a",
			Bool:   false,
		},
		{
			Int:    2,
			String: "b",
			Bool:   true,
		},
	}
	testCases := []struct {
		includes []string
		expected []*indexedTestModel
	}{
		{
			// Include only Int
			[]string{"Int"},
			[]*indexedTestModel{
				{
					Int:    1,
					String: "",
					Bool:   false,
				},
				{
					Int:    2,
					String: "",
					Bool:   false,
				},
			},
		},
		{
			// Include only String
			[]string{"String"},
			[]*indexedTestModel{
				{
					Int:    0,
					String: "a",
					Bool:   false,
				},
				{
					Int:    0,
					String: "b",
					Bool:   false,
				},
			},
		},
		{
			// Include only Bool
			[]string{"Bool"},
			[]*indexedTestModel{
				{
					Int:    0,
					String: "",
					Bool:   false,
				},
				{
					Int:    0,
					String: "",
					Bool:   true,
				},
			},
		},
		{
			// Include everything
			[]string{"Int", "String", "Bool"},
			[]*indexedTestModel{
				{
					Int:    1,
					String: "a",
					Bool:   false,
				},
				{
					Int:    2,
					String: "b",
					Bool:   true,
				},
			},
		},
	}
	for i, tc := range testCases {
		got := applyIncludes(models, tc.includes)
		if !reflect.DeepEqual(got, tc.expected) {
			t.Errorf("Incorrect result for applyIncludes for test case %d: %v\nExpected: %#v\nGot:  %#v", i, tc.includes, tc.expected, got)
		}
	}
}

// applyExcludes applies excludes to all models. That is, it zeroes out the fields which have field names
// in excludes. applyExcludes returns a copy, so the original slice is left intact.
func applyExcludes(models []*indexedTestModel, excludes []string) []*indexedTestModel {
	results := make([]*indexedTestModel, len(models))
	for i, m := range models {
		result := &indexedTestModel{}
		resVal := reflect.ValueOf(result).Elem()
		origVal := reflect.ValueOf(m).Elem()
		for fieldIndex := 0; fieldIndex < origVal.NumField(); fieldIndex++ {
			fieldType := origVal.Type().Field(fieldIndex)
			if fieldType.Name == "DefaultData" {
				// DefaultData is a special case
				resVal.Field(fieldIndex).Set(origVal.Field(fieldIndex))
			}
			if !stringSliceContains(excludes, fieldType.Name) {
				resVal.Field(fieldIndex).Set(origVal.Field(fieldIndex))
			}
		}
		results[i] = result
	}
	return results
}

func TestApplyExcludes(t *testing.T) {
	models := []*indexedTestModel{
		{
			Int:    1,
			String: "a",
			Bool:   false,
		},
		{
			Int:    2,
			String: "b",
			Bool:   true,
		},
	}
	testCases := []struct {
		excludes []string
		expected []*indexedTestModel
	}{
		{
			// Exclude only Int
			[]string{"Int"},
			[]*indexedTestModel{
				{
					Int:    0,
					String: "a",
					Bool:   false,
				},
				{
					Int:    0,
					String: "b",
					Bool:   true,
				},
			},
		},
		{
			// Exclude only String
			[]string{"String"},
			[]*indexedTestModel{
				{
					Int:    1,
					String: "",
					Bool:   false,
				},
				{
					Int:    2,
					String: "",
					Bool:   true,
				},
			},
		},
		{
			// Exclude only Bool
			[]string{"Bool"},
			[]*indexedTestModel{
				{
					Int:    1,
					String: "a",
					Bool:   false,
				},
				{
					Int:    2,
					String: "b",
					Bool:   false,
				},
			},
		},
		{
			// Exclude everything
			[]string{"Int", "String", "Bool"},
			[]*indexedTestModel{
				{
					Int:    0,
					String: "",
					Bool:   false,
				},
				{
					Int:    0,
					String: "",
					Bool:   false,
				},
			},
		},
	}
	for i, tc := range testCases {
		got := applyExcludes(models, tc.excludes)
		if !reflect.DeepEqual(got, tc.expected) {
			t.Errorf("Incorrect result for applyExcludes for test case %d: %v\nExpected: %#v\nGot:  %#v", i, tc.excludes, tc.expected, got)
		}
	}
}

func applyLimitAndOffset(models []*indexedTestModel, limit, offset uint) []*indexedTestModel {
	expected := make([]*indexedTestModel, len(models))
	copy(expected, models)
	start := offset
	var end uint
	if limit == 0 {
		end = uint(len(expected))
	} else {
		end = start + limit
	}
	if int(start) > len(expected) {
		return []*indexedTestModel{}
	} else if int(end) > len(expected) {
		return expected[start:]
	} else {
		return expected[start:end]
	}
	return expected
}

func TestApplyLimitAndOffset(t *testing.T) {
	models := createIndexedTestModels(10)

	testCases := []struct {
		limit    uint
		offset   uint
		expected []*indexedTestModel
	}{
		{
			limit:    0,
			offset:   0,
			expected: models,
		},
		{
			limit:    0,
			offset:   5,
			expected: models[5:],
		},
		{
			limit:    0,
			offset:   10,
			expected: []*indexedTestModel{},
		},
		{
			limit:    0,
			offset:   0,
			expected: models,
		},
		{
			limit:    1,
			offset:   0,
			expected: models[:1],
		},
		{
			limit:    5,
			offset:   0,
			expected: models[:5],
		},
		{
			limit:    10,
			offset:   0,
			expected: models,
		},
		{
			limit:    11,
			offset:   0,
			expected: models,
		},
		{
			limit:    3,
			offset:   3,
			expected: models[3:6],
		},
		{
			limit:    5,
			offset:   5,
			expected: models[5:10],
		},
		{
			limit:    5,
			offset:   10,
			expected: []*indexedTestModel{},
		},
		{
			limit:    10,
			offset:   10,
			expected: []*indexedTestModel{},
		},
	}

	for i, tc := range testCases {
		got := applyLimitAndOffset(models, tc.limit, tc.offset)
		if !reflect.DeepEqual(got, tc.expected) {
			t.Errorf("Incorrect result for applyLimitAndOffset for test case %d (limit: %d, offset: %d)\nExpected: %#v\nGot:  %#v",
				i, tc.limit, tc.offset, tc.expected, got)
		}
	}
}
