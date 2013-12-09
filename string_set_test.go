package zoom

import (
	"reflect"
	"testing"
)

func TestNewStringSet(t *testing.T) {
	s := newStringSet("one", "two", "three")
	m := map[string]bool(s)
	expected := []string{"one", "two", "three"}
	for _, e := range expected {
		if _, found := m[e]; !found {
			t.Errorf("set did not contain correct values. Missing %s", e)
		}
	}
}

func TestNewStringSetFromSlice(t *testing.T) {
	s := newStringSetFromSlice([]string{"one", "two", "three"})
	m := map[string]bool(s)
	expected := []string{"one", "two", "three"}
	for _, e := range expected {
		if _, found := m[e]; !found {
			t.Errorf("set did not contain correct values. Missing %s", e)
		}
	}
}

func TestAdd(t *testing.T) {
	s := newStringSet()
	s.add("one")
	s.add("two", "three")
	m := map[string]bool(s)
	expected := []string{"one", "two", "three"}
	for _, e := range expected {
		if _, found := m[e]; !found {
			t.Errorf("set did not contain correct values. Missing %s", e)
		}
	}
}

func TestContains(t *testing.T) {
	s := newStringSet("one", "two", "three")
	if !s.contains("one") {
		t.Error("incorrect. s.contains('one') should return true")
	}
	if s.contains("four") {
		t.Error("incorrect. s.contains('four') should return false")
	}
}

func TestSize(t *testing.T) {
	s0 := newStringSet()
	s1 := newStringSet("one", "two", "three")
	sets := []stringSet{s0, s1}
	expecteds := []int{0, 3}
	for i, s := range sets {
		expected := expecteds[i]
		if s.size() != expected {
			t.Errorf("incorrect size. Expected %d but got %d", expected, s.size())
		}
	}
}

func TestEquals(t *testing.T) {
	s0 := newStringSet()
	s1 := newStringSet("one", "two", "three")
	s2 := newStringSet("one", "three", "two")
	if s0.equals(s1) {
		t.Error("incorrect. expected false but got true")
	}
	if !s1.equals(s2) {
		t.Error("incorrect. expected true but got false")
	}
	if !s2.equals(s1) {
		t.Error("incorrect. expected true but got false")
	}
}

func TestSortBySize(t *testing.T) {
	s0 := newStringSet()
	s1 := newStringSet("one", "two", "three")
	s2 := newStringSet("one")
	s3 := newStringSet("one", "two")
	sets := []stringSet{s0, s1, s2, s3}
	sortStringSetsBySize(sets)
	expecteds := []stringSet{s0, s2, s3, s1}
	for i, expected := range expecteds {
		got := sets[i]
		if !got.equals(expected) {
			t.Errorf("sets were not sorted correctly. set %d should have been %v but was %v", i, expected, got)
		}
	}
}

func TestIntersect(t *testing.T) {
	s0 := newStringSet()
	s1 := newStringSet("one", "two", "three", "four", "five")
	s2 := newStringSet("one", "three", "four", "five")
	s3 := newStringSet("one", "three", "five", "six")

	s0x := s0.intersect(newStringSet())
	if !s0x.equals(s0) {
		t.Errorf("intersection of 0 and [] was incorrect. Expected [] but got %v", s0x)
	}
	s01 := s0.intersect(s1)
	if !s01.equals(s0) {
		t.Errorf("intersection of 0 and 1 was incorrect. Expected [] but got %v", s01)
	}
	s0123 := s0.intersect(s1, s2, s3)
	if !s01.equals(s0) {
		t.Errorf("intersection of 0, 1, 2, and 3 was incorrect. Expected [] but got %v", s0123)
	}
	s12 := s1.intersect(s2)
	e12 := newStringSet("one", "three", "four", "five")
	if !s12.equals(e12) {
		t.Errorf("intersection of 1 and 2 was incorrect. Expected %v but got %v", e12, s12)
	}
	s123 := s1.intersect(s2, s3)
	e123 := newStringSet("one", "three", "five")
	if !s123.equals(e123) {
		t.Errorf("intersection of 1, 2, and 3 was incorrect. Expected %v but got %v", e123, s123)
	}
}

func TestSlice(t *testing.T) {
	s0 := newStringSet()
	e0 := []string{}
	if !reflect.DeepEqual(s0.slice(), e0) {
		t.Errorf("incorrect results for slice. Expected [] but got %v", s0.slice())
	}
	s1 := newStringSet("one", "two", "three")
	e1 := []string{"one", "two", "three"}
	if !reflect.DeepEqual(s1.slice(), e1) {
		t.Errorf("incorrect results for slice. Expected %v but got %v", e1, s1.slice())
	}
}
