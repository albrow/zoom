// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

package zoom

import (
	"reflect"
	"testing"
)

func TestIndexOfStringSlice(t *testing.T) {
	slice := []string{"one", "two", "three"}
	index := indexOfStringSlice("two", slice)
	if index != 1 {
		t.Errorf("index was incorrect.\nExpected: %d\nGot: %d\n", 1, index)
	}
	index = indexOfStringSlice("four", slice)
	if index != -1 {
		t.Errorf("index was incorrect.\nExpected: %d\nGot: %d\n", -1, index)
	}
}

func TestStringSliceContains(t *testing.T) {
	slice := []string{"one", "two", "three"}
	contains := stringSliceContains("two", slice)
	if contains != true {
		t.Errorf("contains was incorrect.\nExpected: %t\nGot: %t\n", true, contains)
	}
	contains = stringSliceContains("four", slice)
	if contains != false {
		t.Errorf("contains was incorrect.\nExpected: %t\nGot: %t\n", false, contains)
	}
}

func TestRemoveFromStringSlice(t *testing.T) {
	start := []string{"one", "two", "three", "four"}
	got := removeFromStringSlice(start, 2)
	expected := []string{"one", "two", "four"}
	if !reflect.DeepEqual(expected, got) {
		t.Errorf("result was incorrect.\nExpected: %v\nGot: %v\n", expected, got)
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
	if equal, _ := compareAsStringSet(a, c); equal {
		t.Errorf("equal was incorrect.\nExpected: %t\nGot: %t\n", false, equal)
	}
	if equal, _ := compareAsStringSet(a, d); equal {
		t.Errorf("equal was incorrect.\nExpected: %t\nGot: %t\n", false, equal)
	}
}

func TestIndexOfSlice(t *testing.T) {
	slice := []string{"one", "two", "three"}
	index := indexOfSlice("two", slice)
	if index != 1 {
		t.Errorf("index was incorrect.\nExpected: %d\nGot: %d\n", 1, index)
	}
	index = indexOfSlice("four", slice)
	if index != -1 {
		t.Errorf("index was incorrect.\nExpected: %d\nGot: %d\n", -1, index)
	}
}

func TestSliceContains(t *testing.T) {
	slice := []string{"one", "two", "three"}
	contains := sliceContains("two", slice)
	if contains != true {
		t.Errorf("contains was incorrect.\nExpected: %t\nGot: %t\n", true, contains)
	}
	contains = sliceContains("four", slice)
	if contains != false {
		t.Errorf("contains was incorrect.\nExpected: %t\nGot: %t\n", false, contains)
	}
}

func TestCompareAsSet(t *testing.T) {
	a := []string{"one", "two", "three"}
	b := []string{"two", "three", "one"}
	c := []string{"three", "one"}
	d := []string{"four", "one", "three", "two"}
	if equal, msg := compareAsSet(a, b); !equal {
		t.Errorf("equal was incorrect.\nExpected: %t\nGot: %t\nMsg: %s\n", true, equal, msg)
	}
	if equal, msg := compareAsSet(a, c); equal {
		t.Errorf("equal was incorrect.\nExpected: %t\nGot: %t\nMsg: %s\n", false, equal, msg)
	}
	if equal, msg := compareAsSet(a, d); equal {
		t.Errorf("equal was incorrect.\nExpected: %t\nGot: %t\nMsg: %s\n", false, equal, msg)
	}
}
