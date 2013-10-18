// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

package test

import (
	"github.com/garyburd/redigo/redis"
	"github.com/stephenalexbrowne/zoom"
	"github.com/stephenalexbrowne/zoom/test_support"
	"github.com/stephenalexbrowne/zoom/util"
	"reflect"
	"strconv"
	"testing"
)

func TestSave(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save a new person
	p := &test_support.Person{Name: "Bob", Age: 25}
	err := zoom.Save(p)
	if err != nil {
		t.Error(err)
	}

	// get a connection
	conn := zoom.GetConn()
	defer conn.Close()

	checkPersonSaved(t, p, conn)
}

func TestVariadicSave(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create some new persons, but don't save them yet
	persons, err := test_support.NewPersons(3)
	if err != nil {
		t.Error(err)
	}

	if err := zoom.Save(zoom.Models(persons)...); err != nil {
		t.Error(err)
	}

	// get a connection
	conn := zoom.GetConn()
	defer conn.Close()

	// for each person...
	for _, p := range persons {
		checkPersonSaved(t, p, conn)
	}
}

func TestFindById(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save a new model
	persons, _ := test_support.CreatePersons(1)
	p := persons[0]

	// execute a test query and compare against the expected person
	testFindWithExpectedPerson(t, zoom.FindById("person", p.Id), p)
}

func TestScanById(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save a new model
	persons, _ := test_support.CreatePersons(1)
	p := persons[0]

	pCopy := &test_support.Person{}

	// execute a test query and compare against the expected person
	testScanWithExpectedPersonAndScannable(t, zoom.ScanById(p.Id, pCopy), p, pCopy)
}

func TestDelete(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save a new model
	p := &test_support.Person{Name: "Bob", Age: 25}
	zoom.Save(p)

	// delete it
	if err := zoom.Delete(p); err != nil {
		t.Error(err)
	}

	// get a connection
	conn := zoom.GetConn()
	defer conn.Close()

	// make sure it's gone
	checkPersonDeleted(t, p.Id, conn)
}

func TestVariadicDelete(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save some new models
	persons, err := test_support.CreatePersons(3)
	if err != nil {
		t.Error(err)
	}

	// delete it
	if err := zoom.Delete(zoom.Models(persons)...); err != nil {
		t.Error(err)
	}

	// get a connection
	conn := zoom.GetConn()
	defer conn.Close()

	// for each person...
	for _, p := range persons {
		// make sure it's gone
		checkPersonDeleted(t, p.Id, conn)
	}
}

func TestDeleteById(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save a new model
	p := &test_support.Person{Name: "Bob", Age: 25}
	zoom.Save(p)

	// delete it
	zoom.DeleteById("person", p.Id)

	// get a connection
	conn := zoom.GetConn()
	defer conn.Close()

	// make sure it's gone
	checkPersonDeleted(t, p.Id, conn)
}

func TestVariadicDeleteById(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save some new models
	persons, err := test_support.CreatePersons(3)
	if err != nil {
		t.Error(err)
	}

	// delete it
	zoom.DeleteById("person", persons[0].Id, persons[1].Id, persons[2].Id)

	// get a connection
	conn := zoom.GetConn()
	defer conn.Close()

	// for each person...
	for _, p := range persons {
		// make sure it's gone
		checkPersonDeleted(t, p.Id, conn)
	}
}

func TestFindAll(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save some persons
	persons, err := test_support.CreatePersons(3)
	if err != nil {
		t.Error(err)
	}

	// execute a test query and check the results
	testFindAllWithExpectedPersons(t, zoom.FindAll("person"), persons, false)
}

func TestScanAll(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save some persons
	persons, err := test_support.CreatePersons(3)
	if err != nil {
		t.Error(err)
	}

	// create a scannable
	personsCopy := make([]*test_support.Person, 0)

	// execute a test query and check the results
	testFindAllWithExpectedPersonsAndScannable(t, zoom.ScanAll(&personsCopy), persons, false, &personsCopy)
}

func TestFindAllSortAlpha(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save some persons
	persons, err := test_support.CreatePersons(3)
	if err != nil {
		t.Error(err)
	}

	// execute a test query and check the results
	q := zoom.FindAll("person").SortBy("Name")
	testFindAllWithExpectedPersons(t, q, persons, true)
}

func TestFindAllSortNumeric(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save some persons
	persons, err := test_support.CreatePersons(3)
	if err != nil {
		t.Error(err)
	}

	// execute a test query and check the results
	q := zoom.FindAll("person").SortBy("Age")
	testFindAllWithExpectedPersons(t, q, persons, true)
}

func TestFindAllSortAlphaDesc(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save some persons
	persons, err := test_support.CreatePersons(3)
	if err != nil {
		t.Error(err)
	}

	// execute a test query and check the results
	q := zoom.FindAll("person").SortBy("Name").Order("DESC")
	expected := []*test_support.Person{persons[2], persons[1], persons[0]}
	testFindAllWithExpectedPersons(t, q, expected, true)
}

func TestFindAllSortNumericDesc(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save some persons
	persons, err := test_support.CreatePersons(3)
	if err != nil {
		t.Error(err)
	}

	// execute a test query and check the results
	q := zoom.FindAll("person").SortBy("Age").Order("DESC")
	expected := []*test_support.Person{persons[2], persons[1], persons[0]}
	testFindAllWithExpectedPersons(t, q, expected, true)
}

func TestFindAllLimit(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save some persons
	persons, err := test_support.CreatePersons(3)
	if err != nil {
		t.Error(err)
	}

	// execute a test query and check the results
	q := zoom.FindAll("person").SortBy("Name").Limit(2)
	expected := []*test_support.Person{persons[0], persons[1]}
	testFindAllWithExpectedPersons(t, q, expected, true)
}

func TestFindAllOffset(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save some persons
	persons, err := test_support.CreatePersons(3)
	if err != nil {
		t.Error(err)
	}

	// execute a test query and check the results
	q := zoom.FindAll("person").SortBy("Name").Limit(2).Offset(1)
	expected := []*test_support.Person{persons[1], persons[2]}
	testFindAllWithExpectedPersons(t, q, expected, true)
}

func TestSaveWithList(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create a modelWithList model
	m := &test_support.ModelWithList{
		List: []string{"one", "two", "three"},
	}

	// save it
	if err := zoom.Save(m); err != nil {
		t.Error(err)
	}

	// get a connection
	conn := zoom.GetConn()
	defer conn.Close()

	// make sure the list was saved properly
	listKey := "modelWithList:" + m.Id + ":List"
	listCopy, err := redis.Strings(conn.Do("LRANGE", listKey, 0, -1))
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(m.List, listCopy) {
		t.Errorf("List was not the same.\nExpected: %v\nGot: %v\n", m.List, listCopy)
	}
}

func TestFindByIdWithList(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save a modelWithList model
	m := &test_support.ModelWithList{
		List: []string{"one", "two", "three"},
	}
	zoom.Save(m)

	// retrieve using FindById
	mCopy := &test_support.ModelWithList{}
	if _, err := zoom.ScanById(m.Id, mCopy).Run(); err != nil {
		t.Error(err)
	}

	// make sure the list is the same
	if !reflect.DeepEqual(m.List, mCopy.List) {
		t.Errorf("List was not the same.\nExpected: %v\nGot: %v\n", m.List, mCopy.List)
	}
}

func TestSaveWithSet(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create a modelWithSet model
	m := &test_support.ModelWithSet{
		Set: []string{"one", "two", "three", "three"},
	}

	// save it
	if err := zoom.Save(m); err != nil {
		t.Error(err)
	}

	// get a connection
	conn := zoom.GetConn()
	defer conn.Close()

	// make sure the set was saved properly
	setKey := "modelWithSet:" + m.Id + ":Set"
	setCopy, err := redis.Strings(conn.Do("SMEMBERS", setKey))
	if err != nil {
		t.Error(err)
	}
	set := []string{"one", "two", "three"}
	equal, msg := util.CompareAsStringSet(set, setCopy)
	if !equal {
		t.Error(msg)
	}
}

func TestFindByIdWithSet(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save a modelWithSet model
	m := &test_support.ModelWithSet{
		Set: []string{"one", "two", "three", "three"},
	}
	zoom.Save(m)

	// retrieve using FindById
	mCopy := &test_support.ModelWithSet{}
	if _, err := zoom.ScanById(m.Id, mCopy).Run(); err != nil {
		t.Error(err)
	}

	// make sure the set is what we expect
	set := []string{"one", "two", "three"}
	equal, msg := util.CompareAsStringSet(set, mCopy.Set)
	if !equal {
		t.Error(msg)
	}
}

func TestFindByIdExclude(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save a new model
	persons, _ := test_support.CreatePersons(1)
	p := persons[0]

	// execute a test query and compare against the expected person
	noName := &test_support.Person{Age: p.Age}
	noName.Id = p.Id
	testFindWithExpectedPerson(t, zoom.FindById("person", p.Id).Exclude("Name"), noName)
}

func TestFindByIdInclude(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save a new model
	persons, _ := test_support.CreatePersons(1)
	p := persons[0]

	// execute a test query and compare against the expected person
	justName := &test_support.Person{Name: p.Name}
	justName.Id = p.Id
	testFindWithExpectedPerson(t, zoom.FindById("person", p.Id).Include("Name"), justName)
}

func TestFindByIdWithListExclude(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save a modelWithList model
	m := &test_support.ModelWithList{
		List: []string{"one", "two", "three"},
	}
	zoom.Save(m)

	// retrieve using FindById
	mCopy := &test_support.ModelWithList{}
	if _, err := zoom.ScanById(m.Id, mCopy).Exclude("List").Run(); err != nil {
		t.Error(err)
	}

	// make sure the list is empty
	if len(mCopy.List) != 0 {
		t.Errorf("list was not empty. was: %v\n", mCopy.List)
	}
}

func TestFindByIdWithSetExclude(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save a modelWithSet model
	m := &test_support.ModelWithSet{
		Set: []string{"one", "two", "three", "three"},
	}
	zoom.Save(m)

	// retrieve using FindById
	mCopy := &test_support.ModelWithSet{}
	if _, err := zoom.ScanById(m.Id, mCopy).Exclude("Set").Run(); err != nil {
		t.Error(err)
	}

	// make sure the set is empty
	if len(mCopy.Set) != 0 {
		t.Errorf("set was not empty. was: %v\n", mCopy.Set)
	}
}

func findTester(t *testing.T, query zoom.Query, checker func(*testing.T, interface{})) {
	// execute the query
	result, err := query.Run()
	if err != nil {
		t.Error(err)
	}

	// run the checker function
	checker(t, result)
}

func testFindWithExpectedPerson(t *testing.T, query zoom.Query, expectedPerson *test_support.Person) {
	findTester(t, query, func(t *testing.T, result interface{}) {
		if result == nil {
			t.Error("result of query was nil")
		}
		pCopy, ok := result.(*test_support.Person)
		if !ok {
			t.Error("could not type assert result to *Person")
		}

		// make sure the found model is the same as original
		checkPersonsEqual(t, expectedPerson, pCopy)
	})
}

func testScanWithExpectedPersonAndScannable(t *testing.T, query zoom.Query, expectedPerson *test_support.Person, scannable *test_support.Person) {
	testFindWithExpectedPerson(t, query, expectedPerson)
	checkPersonsEqual(t, expectedPerson, scannable)
}

func findAllTester(t *testing.T, query zoom.Query, checker func(*testing.T, interface{})) {
	// execute the query
	results, err := query.Run()
	if err != nil {
		t.Error(err)
	}

	// run the checker function
	checker(t, results)
}

func testFindAllWithExpectedPersons(t *testing.T, query zoom.Query, expectedPersons []*test_support.Person, orderMatters bool) {
	findAllTester(t, query, func(t *testing.T, results interface{}) {
		gotPersons, ok := results.([]*test_support.Person)
		if !ok {
			t.Error("could not convert results to []*Person")
		}

		// make sure gotPersons is the right length
		if len(gotPersons) != len(expectedPersons) {
			t.Errorf("gotPersons was not the right length. Expected: %d. Got: %d.\n", len(gotPersons), len(expectedPersons))
		}

		if orderMatters {
			if !reflect.DeepEqual(expectedPersons, gotPersons) {
				t.Errorf("Persons were incorrect.\nExpected: %v\nGot: %v\n", expectedPersons, gotPersons)
			}
		} else {
			equal, msg := util.CompareAsSet(expectedPersons, gotPersons)
			if !equal {
				t.Errorf("Persons were incorrect\n%s\nExpected: %v\nGot: %v\n", msg, expectedPersons, gotPersons)
			}
		}
	})
}

func testFindAllWithExpectedPersonsAndScannable(t *testing.T, query zoom.Query, expectedPersons []*test_support.Person, orderMatters bool, scannables *[]*test_support.Person) {
	testFindAllWithExpectedPersons(t, query, expectedPersons, orderMatters)
	if orderMatters {
		if !reflect.DeepEqual(expectedPersons, *scannables) {
			t.Errorf("expected persons did not match scannables.\nExpected: %v\nGot: %v\n", expectedPersons, *scannables)
		}
	} else {
		if equal, msg := util.CompareAsSet(expectedPersons, *scannables); !equal {
			t.Errorf("expected persons did not match scannables.\nMsg: %s\nExpected: %v\nGot: %v\n", msg, expectedPersons, *scannables)
		}
	}
}

func checkPersonsEqual(t *testing.T, expected, got *test_support.Person) {
	if !reflect.DeepEqual(expected, got) {
		t.Errorf("person was not equal.\nExpected: %+v\nGot: %+v\n", expected, got)
	}
}

func checkPersonSaved(t *testing.T, p *test_support.Person, conn redis.Conn) {
	// make sure it was assigned an id
	if p.Id == "" {
		t.Error("model was not assigned an id")
	}

	// make sure the key exists
	key := "person:" + p.Id
	exists, err := zoom.KeyExists(key, conn)
	if err != nil {
		t.Error(err)
	}
	if exists == false {
		t.Error("model was not saved in redis")
	}

	// make sure the values are correct for the person
	results, err := redis.Strings(conn.Do("HMGET", key, "Name", "Age"))
	if err != nil {
		t.Error(err)
	}
	if results[0] != p.Name {
		t.Errorf("Name of saved model was incorrect. Expected: %s. Got: %s.\n", p.Name, results[0])
	}
	if results[1] != strconv.Itoa(p.Age) {
		t.Errorf("Age of saved model was incorrect. Expected: %d. Got: %s.\n", p.Age, results[1])
	}

	// make sure it was added to the index
	indexed, err := zoom.SetContains("person:all", p.Id, conn)
	if err != nil {
		t.Error(err)
	}
	if indexed == false {
		t.Error("model was not added to person:all")
	}
}

func checkPersonDeleted(t *testing.T, id string, conn redis.Conn) {
	key := "person:" + id
	exists, err := zoom.KeyExists(key, conn)
	if err != nil {
		t.Error(err)
	}
	if exists {
		t.Error("model key still exists")
	}

	// Make sure it was removed from index
	indexed, err := zoom.SetContains("person:all", id, conn)
	if err != nil {
		t.Error(err)
	}
	if indexed {
		t.Error("model id is still in person:all")
	}
}
