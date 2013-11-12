// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File query_test.go tests the query abstraction (query.go)

package zoom

import (
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
