// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File string_set.go is a simple string stringSet implementation based
// on an unerlying map[string]bool. Couldn't find a stringSet implementation
// that meets my needs, so I threw this together quickly. The main
// thing needed was an efficient mIntersect and the ability to sort
// by size.

package zoom

import (
	"sort"
)

type stringSet map[string]bool
type setsBySize []stringSet

func (s setsBySize) Len() int           { return len(s) }
func (s setsBySize) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s setsBySize) Less(i, j int) bool { return s[i].size() < s[j].size() }

func newStringSet(elems ...string) stringSet {
	results := make(stringSet)
	for _, elem := range elems {
		results.add(elem)
	}
	return results
}

func newStringSetFromSlice(slice []string) stringSet {
	s := newStringSet()
	for _, el := range slice {
		s.add(el)
	}
	return s
}

func (s stringSet) add(elems ...string) {
	for _, e := range elems {
		s[e] = true
	}
}

func (s stringSet) contains(elem string) bool {
	_, found := s[elem]
	return found
}

func (s stringSet) size() int {
	return len(s)
}

func (s stringSet) equals(other stringSet) bool {
	if s.size() != other.size() {
		return false
	}
	for elem, _ := range s {
		if !other.contains(elem) {
			return false
		}
	}
	return true
}

func sortStringSetsBySize(sets []stringSet) {
	sort.Sort(setsBySize(sets))
}

func (s stringSet) intersect(others ...stringSet) stringSet {
	switch len(others) {
	case 0:
		return s
	case 1:
		results := newStringSet()
		for elem, _ := range s {
			if others[0].contains(elem) {
				results.add(elem)
			}
		}
		return results
	default:
		sortStringSetsBySize(others)
		results := s.intersect(others[0])
		for i := 1; i < len(others); i++ {
			results = results.intersect(others[i])
		}
		return results
	}
}

func (s stringSet) slice() []string {
	results := make([]string, 0)
	for elem, _ := range s {
		results = append(results, elem)
	}
	return results
}
