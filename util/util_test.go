package util

import (
	"reflect"
	"testing"
)

func TestIndexOfStringSlice(t *testing.T) {
	slice := []string{"one", "two", "three"}
	index := IndexOfStringSlice("two", slice)
	if index != 1 {
		t.Errorf("index was incorrect.\nExpected: %d\nGot: %d\n", 1, index)
	}
	index = IndexOfStringSlice("four", slice)
	if index != -1 {
		t.Errorf("index was incorrect.\nExpected: %d\nGot: %d\n", -1, index)
	}
}

func TestStringSliceContains(t *testing.T) {
	slice := []string{"one", "two", "three"}
	contains := StringSliceContains("two", slice)
	if contains != true {
		t.Errorf("contains was incorrect.\nExpected: %t\nGot: %t\n", true, contains)
	}
	contains = StringSliceContains("four", slice)
	if contains != false {
		t.Errorf("contains was incorrect.\nExpected: %t\nGot: %t\n", false, contains)
	}
}

func TestRemoveFromStringSlice(t *testing.T) {
	start := []string{"one", "two", "three", "four"}
	got := RemoveFromStringSlice(start, 2)
	expected := []string{"one", "two", "four"}
	if !reflect.DeepEqual(expected, got) {
		t.Errorf("result was incorrect.\nExpected: %v\nGot: %v\n", expected, got)
	}
}

func TestRemoveElementFromStringSlice(t *testing.T) {
	start := []string{"one", "two", "three", "four"}
	got := RemoveElementFromStringSlice(start, "three")
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
	if equal, _ := CompareAsStringSet(a, b); !equal {
		t.Errorf("equal was incorrect.\nExpected: %t\nGot: %t\n", true, equal)
	}
	if equal, _ := CompareAsStringSet(a, c); equal {
		t.Errorf("equal was incorrect.\nExpected: %t\nGot: %t\n", false, equal)
	}
	if equal, _ := CompareAsStringSet(a, d); equal {
		t.Errorf("equal was incorrect.\nExpected: %t\nGot: %t\n", false, equal)
	}
}

func TestIndexOfSlice(t *testing.T) {
	slice := []string{"one", "two", "three"}
	index := IndexOfSlice("two", slice)
	if index != 1 {
		t.Errorf("index was incorrect.\nExpected: %d\nGot: %d\n", 1, index)
	}
	index = IndexOfSlice("four", slice)
	if index != -1 {
		t.Errorf("index was incorrect.\nExpected: %d\nGot: %d\n", -1, index)
	}
}

func TestSliceContains(t *testing.T) {
	slice := []string{"one", "two", "three"}
	contains := SliceContains("two", slice)
	if contains != true {
		t.Errorf("contains was incorrect.\nExpected: %t\nGot: %t\n", true, contains)
	}
	contains = SliceContains("four", slice)
	if contains != false {
		t.Errorf("contains was incorrect.\nExpected: %t\nGot: %t\n", false, contains)
	}
}

func TestRemoveFromSlice(t *testing.T) {
	start := []string{"one", "two", "three", "four"}
	got := (RemoveFromSlice(start, 2)).([]string)
	expected := []string{"one", "two", "four"}
	if !reflect.DeepEqual(expected, got) {
		t.Errorf("result was incorrect.\nExpected: %v\nGot: %v\n", expected, got)
	}
}

func TestCompareAsSet(t *testing.T) {
	a := []string{"one", "two", "three"}
	b := []string{"two", "three", "one"}
	c := []string{"three", "one"}
	d := []string{"four", "one", "three", "two"}
	if equal, msg := CompareAsSet(a, b); !equal {
		t.Errorf("equal was incorrect.\nExpected: %t\nGot: %t\nMsg: %s\n", true, equal, msg)
	}
	if equal, msg := CompareAsSet(a, c); equal {
		t.Errorf("equal was incorrect.\nExpected: %t\nGot: %t\nMsg: %s\n", false, equal, msg)
	}
	if equal, msg := CompareAsSet(a, d); equal {
		t.Errorf("equal was incorrect.\nExpected: %t\nGot: %t\nMsg: %s\n", false, equal, msg)
	}
}
