// Copyright 2015 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// file util_test.go contains unit tests for the functions
// in util.go

package zoom

import (
	"reflect"
	"testing"
)

func TestIndexOfStringSlice(t *testing.T) {
	strings := []string{"one", "two", "three"}
	index := indexOfStringSlice(strings, "two")
	if index != 1 {
		t.Errorf("index was incorrect.\nExpected: %d\nGot: %d\n", 1, index)
	}
	index = indexOfStringSlice(strings, "four")
	if index != -1 {
		t.Errorf("index was incorrect.\nExpected: %d\nGot: %d\n", -1, index)
	}
}

func TestStringSliceContains(t *testing.T) {
	strings := []string{"one", "two", "three"}
	contains := stringSliceContains(strings, "two")
	if contains != true {
		t.Errorf("contains was incorrect.\nExpected: %t\nGot: %t\n", true, contains)
	}
	contains = stringSliceContains(strings, "four")
	if contains != false {
		t.Errorf("contains was incorrect.\nExpected: %t\nGot: %t\n", false, contains)
	}
}

func TestRemoveElementFromStringSlice(t *testing.T) {
	start := []string{"one", "two", "three", "four"}
	got := removeElementFromStringSlice(start, "three")
	expected := []string{"one", "two", "four"}
	if !reflect.DeepEqual(expected, got) {
		t.Errorf("result was incorrect.\nExpected: %v\nGot: %v\n", expected, got)
	}
}

func TestCompareAsStringSet(t *testing.T) {
	a := []string{"one", "two", "three"}
	b := []string{"two", "three", "one"}
	c := []string{"three", "one"}
	d := []string{"four", "one", "three", "two"}

	if equal, _ := compareAsStringSet(a, b); !equal {
		t.Errorf("equal was incorrect.\nExpected: %t\nGot: %t\n", true, equal)
	}
	if !reflect.DeepEqual(a, []string{"one", "two", "three"}) {
		t.Error("a was modified after the compareAsStringSet operation!\ncurrent value: ", a)
	}

	if equal, _ := compareAsStringSet(a, c); equal {
		t.Errorf("equal was incorrect.\nExpected: %t\nGot: %t\n", false, equal)
	}
	if !reflect.DeepEqual(a, []string{"one", "two", "three"}) {
		t.Error("a was modified after the compareAsStringSet operation!\ncurrent value: ", a)
	}

	if equal, _ := compareAsStringSet(a, d); equal {
		t.Errorf("equal was incorrect.\nExpected: %t\nGot: %t\n", false, equal)
	}
	if !reflect.DeepEqual(a, []string{"one", "two", "three"}) {
		t.Error("a was modified after the compareAsStringSet operation!\ncurrent value: ", a)
	}
}

// TODO: test other functions which may be mising from here!
