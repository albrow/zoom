// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

package zoom

import (
	"github.com/garyburd/redigo/redis"
	"testing"
)

func TestSaveOneToOneDifferentType(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	bm := &basicModel{Attr: "test"}
	Save(bm)

	m := &oneToOneModelDifferentType{Attr: "test", One: bm}
	if err := Save(m); err != nil {
		t.Error(err)
	}

	// get a connection
	conn := GetConn()
	defer conn.Close()

	// invoke redis driver to check if the value was set appropriately
	oneKey := "oneToOneModelDifferentType:" + m.Id + ":One"
	id, err := redis.String(conn.Do("GET", oneKey))
	if err != nil {
		t.Error(err)
	}
	if id != bm.Id {
		t.Errorf("basic model id for the model was not set correctly.\nExpected: %s\nGot: %s\n", bm.Id, id)
	}
}

func TestFindOneToOneDifferentType(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	bm := &basicModel{Attr: "test"}
	m := &oneToOneModelDifferentType{Attr: "test", One: bm}
	MSave([]Model{bm, m})

	mCopy := new(oneToOneModelDifferentType)
	if err := ScanById(m.Id, mCopy); err != nil {
		t.Error(err)
	}

	if equal, err := looseEquals(m, mCopy); err != nil {
		t.Error(err)
	} else if !equal {
		t.Errorf("one to one model not saved correctly.Expected: %+v\nGot: %+v\n", m, mCopy)
	}
	if equal, err := looseEquals(m.One, mCopy.One); err != nil {
		t.Error(err)
	} else if !equal {
		t.Errorf("one to one model not saved correctly.Expected: %+v\nGot: %+v\n", m.One, mCopy.One)
	}
}

func TestSaveOneToOneSameType(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	one := &oneToOneModelSameType{Attr: "test_1"}
	Save(one)

	two := &oneToOneModelSameType{Attr: "test_2", One: one}
	if err := Save(two); err != nil {
		t.Error(err)
	}

	// get a connection
	conn := GetConn()
	defer conn.Close()

	// invoke redis driver to check if the value was set appropriately
	oneKey := "oneToOneModelSameType:" + two.Id + ":One"
	id, err := redis.String(conn.Do("GET", oneKey))
	if err != nil {
		t.Error(err)
	}
	if id != one.Id {
		t.Errorf("basic model id for the model was not set correctly.\nExpected: %s\nGot: %s\n", one.Id, id)
	}
}

func TestFindOneToOneSameType(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	one := &oneToOneModelSameType{Attr: "test_1"}
	two := &oneToOneModelSameType{Attr: "test_2", One: one}
	MSave([]Model{one, two})

	twoCopy := new(oneToOneModelSameType)
	if err := ScanById(two.Id, twoCopy); err != nil {
		t.Error(err)
	}

	if equal, err := looseEquals(two, twoCopy); err != nil {
		t.Error(err)
	} else if !equal {
		t.Errorf("one to one model not saved correctly.Expected: %+v\nGot: %+v\n", two, twoCopy)
	}
	if equal, err := looseEquals(two.One, twoCopy.One); err != nil {
		t.Error(err)
	} else if !equal {
		t.Errorf("one to one model not saved correctly.Expected: %+v\nGot: %+v\n", two.One, twoCopy.One)
	}
}

func setUpOneToManyDifferentyType() ([]*basicModel, *oneToManyModelDifferentType, error) {
	bms, err := newBasicModels(3)
	if err != nil {
		return nil, nil, err
	}
	if err := MSave(Models(bms)); err != nil {
		return nil, nil, err
	}

	m := &oneToManyModelDifferentType{
		Attr: "test",
		Many: bms,
	}
	if err := Save(m); err != nil {
		return nil, nil, err
	}

	return bms, m, nil
}

func TestSaveOneToManyDifferentType(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// Test with the one-to-many attr set as nil
	m1 := &oneToManyModelDifferentType{
		Attr: "test",
		Many: nil,
	}
	if err := Save(m1); err != nil {
		t.Error(err)
	}

	// Test with a non-nil one-to-many attr
	bms, m2, err := setUpOneToManyDifferentyType()
	if err != nil {
		t.Error(err)
	}

	// get a connection
	conn := GetConn()
	defer conn.Close()

	// invoke redis driver to check if the value was set appropriately
	manyKey := "oneToManyModelDifferentType:" + m2.Id + ":Many"
	gotIds, err := redis.Strings(conn.Do("SMEMBERS", manyKey))
	if err != nil {
		t.Error(err)
	}

	// compare expected ids to got ids
	expectedIds := make([]string, 0)
	for _, bm := range bms {
		if bm.Id == "" {
			t.Errorf("basic model id was empty for %+v\n", bm)
		}
		expectedIds = append(expectedIds, bm.Id)
	}
	equal, msg := compareAsStringSet(expectedIds, gotIds)
	if !equal {
		t.Errorf("saved one to many ids were not correct.\n%s\n", msg)
	}
}

func TestFindOneToManyDifferentType(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// Test with the one-to-many attr set as nil
	m1 := &oneToManyModelDifferentType{
		Attr: "test",
		Many: nil,
	}
	if err := Save(m1); err != nil {
		t.Error(err)
	}
	m1Copy := new(oneToManyModelDifferentType)
	if err := ScanById(m1.Id, m1Copy); err != nil {
		t.Error(err)
	}

	// Test with a non-nil one-to-many attr
	bms, m2, err := setUpOneToManyDifferentyType()
	if err != nil {
		t.Error(err)
	}

	// get a copy of the model from the database
	m2Copy := new(oneToManyModelDifferentType)
	if err := ScanById(m2.Id, m2Copy); err != nil {
		t.Error(err)
	}

	// compare expected ids to got ids
	expectedIds := make([]string, 0)
	for _, bm := range bms {
		if bm.Id == "" {
			t.Errorf("basic model id was empty for %+v\n", bm)
		}
		expectedIds = append(expectedIds, bm.Id)
	}
	gotIds := make([]string, 0)
	for _, bm := range m2Copy.Many {
		if bm.Id == "" {
			t.Errorf("basic model id was empty for %+v\n", bm)
		}
		gotIds = append(gotIds, bm.Id)
	}
	equal, msg := compareAsStringSet(expectedIds, gotIds)
	if !equal {
		t.Errorf("saved one to many ids were not correct.\n%s\n", msg)
	}
}

func setUpManyToManyDifferentType() ([]*manyToManyModelDifferentTypeOne, []*manyToManyModelDifferentTypeTwo, error) {
	oneA := &manyToManyModelDifferentTypeOne{Attr: "1A"}
	oneB := &manyToManyModelDifferentTypeOne{Attr: "1B"}
	oneC := &manyToManyModelDifferentTypeOne{Attr: "1C"}
	twoA := &manyToManyModelDifferentTypeTwo{Attr: "2A"}
	twoB := &manyToManyModelDifferentTypeTwo{Attr: "2B"}
	twoC := &manyToManyModelDifferentTypeTwo{Attr: "2C"}
	if err := MSave([]Model{oneA, oneB, oneC, twoA, twoB, twoC}); err != nil {
		return nil, nil, err
	}

	// oneA has twoA
	oneA.Many = []*manyToManyModelDifferentTypeTwo{
		twoA,
	}
	// oneB has twoA and twoC
	oneB.Many = []*manyToManyModelDifferentTypeTwo{
		twoA,
		twoC,
	}
	// oneC has nothing
	oneC.Many = make([]*manyToManyModelDifferentTypeTwo, 0)
	// twoA has oneA
	twoA.Many = []*manyToManyModelDifferentTypeOne{
		oneA,
	}
	// twoB has oneC
	twoB.Many = []*manyToManyModelDifferentTypeOne{
		oneC,
	}
	// twoC has oneA and oneB
	twoC.Many = []*manyToManyModelDifferentTypeOne{
		oneA,
		oneB,
	}
	if err := MSave([]Model{oneA, oneB, oneC, twoA, twoB, twoC}); err != nil {
		return nil, nil, err
	}

	return []*manyToManyModelDifferentTypeOne{
			oneA, oneB, oneC,
		}, []*manyToManyModelDifferentTypeTwo{
			twoA, twoB, twoC,
		}, nil
}

func TestSaveManyToManyDifferentType(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ones, twos, err := setUpManyToManyDifferentType()
	if err != nil {
		t.Error(err)
	}

	conn := GetConn()
	defer conn.Close()

	for i, one := range ones {
		manyKey := "manyToManyModelDifferentTypeOne:" + one.Id + ":Many"
		gotIds, err := redis.Strings(conn.Do("SMEMBERS", manyKey))
		if err != nil {
			t.Error(err)
		}
		expectedIds := make([]string, 0)
		for _, two := range one.Many {
			if two.Id == "" {
				t.Errorf("two had nil id: %+v\n", two)
			}
			expectedIds = append(expectedIds, two.Id)
		}
		if equal, msg := compareAsSet(expectedIds, gotIds); !equal {
			t.Errorf("many to many ids were incorrect for one on iteration %d\nExpected: %v\nGot: %v\n%s\n", i, expectedIds, gotIds, msg)
		}
	}
	for i, two := range twos {
		manyKey := "manyToManyModelDifferentTypeTwo:" + two.Id + ":Many"
		gotIds, err := redis.Strings(conn.Do("SMEMBERS", manyKey))
		if err != nil {
			t.Error(err)
		}
		expectedIds := make([]string, 0)
		for _, one := range two.Many {
			if one.Id == "" {
				t.Errorf("one had nil id: %+v\n", one)
			}
			expectedIds = append(expectedIds, one.Id)
		}
		if equal, msg := compareAsSet(expectedIds, gotIds); !equal {
			t.Errorf("many to many ids were incorrect for two on iteration %d\nExpected: %v\nGot: %v\n%s\n", i, expectedIds, gotIds, msg)
		}
	}
}

func TestFindManyToManyDifferentType(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ones, twos, err := setUpManyToManyDifferentType()
	if err != nil {
		t.Error(err)
	}

	for i, one := range ones {
		oneCopy := new(manyToManyModelDifferentTypeOne)
		if err := ScanById(one.Id, oneCopy); err != nil {
			t.Error(err)
		}
		gotIds := make([]string, 0)
		for _, two := range oneCopy.Many {
			if two.Id == "" {
				t.Errorf("two had nil id: %+v\n", two)
			}
			gotIds = append(gotIds, two.Id)
		}
		expectedIds := make([]string, 0)
		for _, two := range one.Many {
			if two.Id == "" {
				t.Errorf("two had nil id: %+v\n", two)
			}
			expectedIds = append(expectedIds, two.Id)
		}
		if equal, msg := compareAsSet(expectedIds, gotIds); !equal {
			t.Errorf("many to many ids were incorrect for one on iteration %d\nExpected: %v\nGot: %v\n%s\n", i, expectedIds, gotIds, msg)
		}
	}
	for i, two := range twos {
		twoCopy := new(manyToManyModelDifferentTypeTwo)
		if err := ScanById(two.Id, twoCopy); err != nil {
			t.Error(err)
		}
		gotIds := make([]string, 0)
		for _, one := range twoCopy.Many {
			if one.Id == "" {
				t.Errorf("one had nil id: %+v\n", one)
			}
			gotIds = append(gotIds, one.Id)
		}
		expectedIds := make([]string, 0)
		for _, one := range two.Many {
			if one.Id == "" {
				t.Errorf("one had nil id: %+v\n", one)
			}
			expectedIds = append(expectedIds, one.Id)
		}
		if equal, msg := compareAsSet(expectedIds, gotIds); !equal {
			t.Errorf("many to many ids were incorrect for two on iteration %d\nExpected: %v\nGot: %v\n%s\n", i, expectedIds, gotIds, msg)
		}
	}
}

func setUpManyToManySameType() ([]*manyToManyModelSameType, error) {
	m1 := &manyToManyModelSameType{Attr: "1A"}
	m2 := &manyToManyModelSameType{Attr: "1B"}
	m3 := &manyToManyModelSameType{Attr: "1C"}
	m4 := &manyToManyModelSameType{Attr: "2A"}
	if err := MSave([]Model{m1, m2, m3, m4}); err != nil {
		return nil, err
	}

	// m1 has m4
	m1.Many = []*manyToManyModelSameType{
		m4,
	}
	// m2 has m1, m3, and m4
	m2.Many = []*manyToManyModelSameType{
		m1,
		m3,
		m4,
	}
	// m3 has nothing
	m3.Many = make([]*manyToManyModelSameType, 0)
	// m4 has m1
	m4.Many = []*manyToManyModelSameType{
		m1,
	}
	if err := MSave([]Model{m1, m2, m3, m4}); err != nil {
		return nil, err
	}

	return []*manyToManyModelSameType{
		m1, m2, m3, m4,
	}, nil
}

func TestSaveManyToManySameType(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms, err := setUpManyToManySameType()
	if err != nil {
		t.Error(err)
	}

	conn := GetConn()
	defer conn.Close()

	for i, m1 := range ms {
		manyKey := "manyToManyModelSameType:" + m1.Id + ":Many"
		gotIds, err := redis.Strings(conn.Do("SMEMBERS", manyKey))
		if err != nil {
			t.Error(err)
		}
		expectedIds := make([]string, 0)
		for _, m2 := range m1.Many {
			if m2.Id == "" {
				t.Errorf("m2 had nil id: %+v\n", m2)
			}
			expectedIds = append(expectedIds, m2.Id)
		}
		if equal, msg := compareAsSet(expectedIds, gotIds); !equal {
			t.Errorf("many to many ids were incorrect for m%d\nExpected: %v\nGot: %v\n%s\n", i+1, expectedIds, gotIds, msg)
		}
	}
}

func TestFindManyToManySameType(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	ms, err := setUpManyToManySameType()
	if err != nil {
		t.Error(err)
	}

	for i, m1 := range ms {
		m1Copy := new(manyToManyModelSameType)
		if err := ScanById(m1.Id, m1Copy); err != nil {
			t.Error(err)
		}
		gotIds := make([]string, 0)
		for _, m2 := range m1Copy.Many {
			if m2.Id == "" {
				t.Errorf("m2 had nil id: %+v\n", m2)
			}
			gotIds = append(gotIds, m2.Id)
		}
		expectedIds := make([]string, 0)
		for _, m2 := range m1.Many {
			if m2.Id == "" {
				t.Errorf("m2 had nil id: %+v\n", m2)
			}
			expectedIds = append(expectedIds, m2.Id)
		}
		if equal, msg := compareAsSet(expectedIds, gotIds); !equal {
			t.Errorf("many to many ids were incorrect for m%d\nExpected: %v\nGot: %v\n%s\n", i+1, expectedIds, gotIds, msg)
		}
	}
}
