package test

import (
	"github.com/stephenalexbrowne/zoom"
	"github.com/stephenalexbrowne/zoom/redis"
	"github.com/stephenalexbrowne/zoom/support"
	"github.com/stephenalexbrowne/zoom/util"
	"reflect"
	"testing"
)

func TestSave(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save a new person
	p := &support.Person{Name: "Bob", Age: 25}
	err := zoom.Save(p)
	if err != nil {
		t.Error(err)
	}

	// make sure it was assigned an id
	if p.Id == "" {
		t.Error("model was not assigned an id")
	}

	// get a connection
	conn := zoom.GetConn()
	defer conn.Close()

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
	if results[0] != "Bob" {
		t.Errorf("Name of saved model was incorrect. Expected: %s. Got: %s.\n", "Bob", results[0])
	}
	if results[1] != "25" {
		t.Errorf("Age of saved model was incorrect. Expected: %s. Got: %s.\n", "25", results[1])
	}

	// make sure it was added to the index
	indexed, err := zoom.SetContains("person:index", p.Id, conn)
	if err != nil {
		t.Error(err)
	}
	if indexed == false {
		t.Error("model was not added to person:index")
	}
}

func TestFindById(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save a new model
	persons, _ := support.CreatePersons(1)
	p := persons[0]

	// execute a test query and compare against the expected person
	testFindWithExpectedPerson(t, zoom.FindById("person", p.Id), p)
}

func TestScanById(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save a new model
	persons, _ := support.CreatePersons(1)
	p := persons[0]

	pCopy := &support.Person{}

	// execute a test query and compare against the expected person
	testScanWithExpectedPersonAndScannable(t, zoom.ScanById(pCopy, p.Id), p, pCopy)
}

func TestDelete(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save a new model
	p := &support.Person{Name: "Bob", Age: 25}
	zoom.Save(p)

	// delete it
	zoom.Delete(p)

	// get a connection
	conn := zoom.GetConn()
	defer conn.Close()

	// Make sure it's gone
	key := "person:" + p.Id
	exists, err := zoom.KeyExists(key, conn)
	if err != nil {
		t.Error(err)
	}
	if exists {
		t.Error("model key still exists")
	}

	// Make sure it was removed from index
	indexed, err := zoom.SetContains("person:index", p.Id, conn)
	if err != nil {
		t.Error(err)
	}
	if indexed {
		t.Error("model id is still in person:index")
	}
}

func TestDeleteById(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save a new model
	p := &support.Person{Name: "Bob", Age: 25}
	zoom.Save(p)

	// delete it
	zoom.DeleteById("person", p.Id)

	// get a connection
	conn := zoom.GetConn()
	defer conn.Close()

	// Make sure it's gone
	key := "person:" + p.Id
	exists, err := zoom.KeyExists(key, conn)
	if err != nil {
		t.Error(err)
	}
	if exists {
		t.Error("model key still exists")
	}

	// Make sure it was removed from index
	indexed, err := zoom.SetContains("person:index", p.Id, conn)
	if err != nil {
		t.Error(err)
	}
	if indexed {
		t.Error("model id is still in person:index")
	}
}

func TestFindAll(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save some persons
	persons, err := support.CreatePersons(3)
	if err != nil {
		t.Error(err)
	}

	// execute a test query and check the results
	testFindAllWithExpectedPersons(t, zoom.FindAll("person"), persons, false)
}

func TestScanAll(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save some persons
	persons, err := support.CreatePersons(3)
	if err != nil {
		t.Error(err)
	}

	// create a scannable
	personsCopy := make([]*support.Person, 0)

	// execute a test query and check the results
	testFindAllWithExpectedPersonsAndScannable(t, zoom.ScanAll(&personsCopy), persons, false, &personsCopy)
}

func TestFindAllSortAlpha(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save some persons
	persons, err := support.CreatePersons(3)
	if err != nil {
		t.Error(err)
	}

	// execute a test query and check the results
	q := zoom.FindAll("person").SortBy("Name")
	testFindAllWithExpectedPersons(t, q, persons, true)
}

func TestFindAllSortNumeric(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save some persons
	persons, err := support.CreatePersons(3)
	if err != nil {
		t.Error(err)
	}

	// execute a test query and check the results
	q := zoom.FindAll("person").SortBy("Age")
	testFindAllWithExpectedPersons(t, q, persons, true)
}

func TestFindAllSortAlphaDesc(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save some persons
	persons, err := support.CreatePersons(3)
	if err != nil {
		t.Error(err)
	}

	// execute a test query and check the results
	q := zoom.FindAll("person").SortBy("Name").Order("DESC")
	expected := []*support.Person{persons[2], persons[1], persons[0]}
	testFindAllWithExpectedPersons(t, q, expected, true)
}

func TestFindAllSortNumericDesc(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save some persons
	persons, err := support.CreatePersons(3)
	if err != nil {
		t.Error(err)
	}

	// execute a test query and check the results
	q := zoom.FindAll("person").SortBy("Age").Order("DESC")
	expected := []*support.Person{persons[2], persons[1], persons[0]}
	testFindAllWithExpectedPersons(t, q, expected, true)
}

func TestFindAllLimit(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save some persons
	persons, err := support.CreatePersons(3)
	if err != nil {
		t.Error(err)
	}

	// execute a test query and check the results
	q := zoom.FindAll("person").SortBy("Name").Limit(2)
	expected := []*support.Person{persons[0], persons[1]}
	testFindAllWithExpectedPersons(t, q, expected, true)
}

func TestFindAllOffset(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save some persons
	persons, err := support.CreatePersons(3)
	if err != nil {
		t.Error(err)
	}

	// execute a test query and check the results
	q := zoom.FindAll("person").SortBy("Name").Limit(2).Offset(1)
	expected := []*support.Person{persons[1], persons[2]}
	testFindAllWithExpectedPersons(t, q, expected, true)
}

func TestSaveWithList(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create a modelWithList model
	m := &support.ModelWithList{
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
	support.SetUp()
	defer support.TearDown()

	// create and save a modelWithList model
	m := &support.ModelWithList{
		List: []string{"one", "two", "three"},
	}
	zoom.Save(m)

	// retrieve using FindById
	mCopy := &support.ModelWithList{}
	if _, err := zoom.ScanById(mCopy, m.Id).Run(); err != nil {
		t.Error(err)
	}

	// make sure the list is the same
	if !reflect.DeepEqual(m.List, mCopy.List) {
		t.Errorf("List was not the same.\nExpected: %v\nGot: %v\n", m.List, mCopy.List)
	}
}

func TestSaveWithSet(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create a modelWithSet model
	m := &support.ModelWithSet{
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
	support.SetUp()
	defer support.TearDown()

	// create and save a modelWithSet model
	m := &support.ModelWithSet{
		Set: []string{"one", "two", "three", "three"},
	}
	zoom.Save(m)

	// retrieve using FindById
	mCopy := &support.ModelWithSet{}
	if _, err := zoom.ScanById(mCopy, m.Id).Run(); err != nil {
		t.Error(err)
	}

	// make sure the set is what we expect
	set := []string{"one", "two", "three"}
	equal, msg := util.CompareAsStringSet(set, mCopy.Set)
	if !equal {
		t.Error(msg)
	}
}

func TestFindByIdInclude(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save a new model
	persons, _ := support.CreatePersons(1)
	p := persons[0]

	// execute a test query and compare against the expected person
	noName := &support.Person{Age: p.Age}
	noName.Id = p.Id
	testFindWithExpectedPerson(t, zoom.FindById("person", p.Id).Exclude("Name"), noName)
}

func TestFindByIdExclude(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save a new model
	persons, _ := support.CreatePersons(1)
	p := persons[0]

	// execute a test query and compare against the expected person
	justName := &support.Person{Name: p.Name}
	justName.Id = p.Id
	testFindWithExpectedPerson(t, zoom.FindById("person", p.Id).Include("Name"), justName)
}

func TestFindByIdWithListExclude(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save a modelWithList model
	m := &support.ModelWithList{
		List: []string{"one", "two", "three"},
	}
	zoom.Save(m)

	// retrieve using FindById
	mCopy := &support.ModelWithList{}
	if _, err := zoom.ScanById(mCopy, m.Id).Exclude("List").Run(); err != nil {
		t.Error(err)
	}

	// make sure the list is empty
	if len(mCopy.List) != 0 {
		t.Errorf("list was not empty. was: %v\n", mCopy.List)
	}
}

func TestFindByIdWithSetExclude(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save a modelWithSet model
	m := &support.ModelWithSet{
		Set: []string{"one", "two", "three", "three"},
	}
	zoom.Save(m)

	// retrieve using FindById
	mCopy := &support.ModelWithSet{}
	if _, err := zoom.ScanById(mCopy, m.Id).Exclude("Set").Run(); err != nil {
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

func testFindWithExpectedPerson(t *testing.T, query zoom.Query, expectedPerson *support.Person) {
	findTester(t, query, func(t *testing.T, result interface{}) {
		if result == nil {
			t.Error("result of query was nil")
		}
		pCopy, ok := result.(*support.Person)
		if !ok {
			t.Error("could not type assert result to *Person")
		}

		// make sure the found model is the same as original
		checkPersonsEqual(t, expectedPerson, pCopy)
	})
}

func testScanWithExpectedPersonAndScannable(t *testing.T, query zoom.Query, expectedPerson *support.Person, scannable *support.Person) {
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

func testFindAllWithExpectedPersons(t *testing.T, query zoom.Query, expectedPersons []*support.Person, orderMatters bool) {
	findAllTester(t, query, func(t *testing.T, results interface{}) {
		gotPersons, ok := results.([]*support.Person)
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

func testFindAllWithExpectedPersonsAndScannable(t *testing.T, query zoom.Query, expectedPersons []*support.Person, orderMatters bool, scannables *[]*support.Person) {
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

func checkPersonsEqual(t *testing.T, expected, got *support.Person) {
	if !reflect.DeepEqual(expected, got) {
		t.Error("person was not equal.\nExpected: %+v\nGot: %+v\n", expected, got)
	}
}
