package zoom

import (
	"reflect"
	"testing"
)

func TestIdSetAdd(t *testing.T) {
	s := newIdSet()
	s.add("1", "2", "3")
	expected := []string{"1", "2", "3"}
	if !reflect.DeepEqual(expected, s.ids) {
		t.Errorf("Elements were not added! Expected %v but got %v", expected, s.ids)
	}
}

func TestIdSetIntersect(t *testing.T) {
	s := newIdSet()
	s.intersect([]string{"1", "2", "3"})
	expected := []string{"1", "2", "3"}
	if !reflect.DeepEqual(expected, s.ids) {
		t.Errorf("Elements were not intersected correctly! Expected %v but got %v", expected, s.ids)
	}
	s.intersect([]string{"2", "2.5", "3", "3.5", "4"})
	expected = []string{"2", "3"}
	if !reflect.DeepEqual(expected, s.ids) {
		t.Errorf("Elements were not intersected correctly! Expected %v but got %v", expected, s.ids)
	}
	s.intersect([]string{"5"})
	expected = []string{}
	if !reflect.DeepEqual(expected, s.ids) {
		t.Errorf("Elements were not intersected correctly! Expected %v but got %v", expected, s.ids)
	}
	s.ids = []string{"3"}
	s.intersect([]string{"1", "2", "4", "5"})
	expected = []string{}
	if !reflect.DeepEqual(expected, s.ids) {
		t.Errorf("Elements were not intersected correctly! Expected %v but got %v", expected, s.ids)
	}
	s.ids = []string{}
	s.intersect([]string{"1"})
	expected = []string{}
	if !reflect.DeepEqual(expected, s.ids) {
		t.Errorf("Elements were not intersected correctly! Expected %v but got %v", expected, s.ids)
	}
}
