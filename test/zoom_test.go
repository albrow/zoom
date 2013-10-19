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

	// use FindById to get a copy of the person
	result, err := zoom.FindById("person", p.Id)
	if err != nil {
		t.Error(err)
	}
	pCopy, ok := result.(*test_support.Person)
	if !ok {
		t.Errorf("Could not convert type %T to *test_support.Person", result)
	}

	// make sure the found model is the same as original
	checkPersonsEqual(t, p, pCopy)
}

func TestScanById(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save a new model
	persons, _ := test_support.CreatePersons(1)
	p := persons[0]

	// create a new person and use ScanById
	pCopy := new(test_support.Person)
	if err := zoom.ScanById(p.Id, pCopy); err != nil {
		t.Error(err)
	}

	// make sure the found model is the same as original
	checkPersonsEqual(t, p, pCopy)
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

func TestScanByIdWithList(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save a modelWithList model
	m := &test_support.ModelWithList{
		List: []string{"one", "two", "three"},
	}
	zoom.Save(m)

	// retrieve using ScanById
	mCopy := &test_support.ModelWithList{}
	if err := zoom.ScanById(m.Id, mCopy); err != nil {
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

func TestScanByIdWithSet(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save a modelWithSet model
	m := &test_support.ModelWithSet{
		Set: []string{"one", "two", "three", "three"},
	}
	zoom.Save(m)

	// retrieve using ScanById
	mCopy := &test_support.ModelWithSet{}
	if err := zoom.ScanById(m.Id, mCopy); err != nil {
		t.Error(err)
	}

	// make sure the set is what we expect
	set := []string{"one", "two", "three"}
	equal, msg := util.CompareAsStringSet(set, mCopy.Set)
	if !equal {
		t.Error(msg)
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
