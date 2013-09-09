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
	testFindAllWithExpectedIds(t, zoom.FindAll("person"), []string{persons[0].Id, persons[1].Id, persons[2].Id}, false)
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
	testFindAllWithExpectedIds(t, q, []string{persons[0].Id, persons[1].Id, persons[2].Id}, true)
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
	testFindAllWithExpectedIds(t, q, []string{persons[0].Id, persons[1].Id, persons[2].Id}, true)
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
	testFindAllWithExpectedIds(t, q, []string{persons[2].Id, persons[1].Id, persons[0].Id}, true)
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
	testFindAllWithExpectedIds(t, q, []string{persons[2].Id, persons[1].Id, persons[0].Id}, true)
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
	testFindAllWithExpectedIds(t, q, []string{persons[0].Id, persons[1].Id}, true)
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
	testFindAllWithExpectedIds(t, q, []string{persons[1].Id, persons[2].Id}, true)
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
	if _, err := zoom.ScanById(mCopy, m.Id).Exec(); err != nil {
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
	if _, err := zoom.ScanById(mCopy, m.Id).Exec(); err != nil {
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

func findTester(t *testing.T, query *zoom.Query, checker func(*testing.T, zoom.Model)) {
	// execute the query
	result, err := query.Exec()
	if err != nil {
		t.Error(err)
	}

	// run the checker function
	checker(t, result)
}

func testFindWithExpectedPerson(t *testing.T, query *zoom.Query, expectedPerson *support.Person) {
	findTester(t, query, func(t *testing.T, result zoom.Model) {
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

func testScanWithExpectedPersonAndScannable(t *testing.T, query *zoom.Query, expectedPerson *support.Person, scannable *support.Person) {
	testFindWithExpectedPerson(t, query, expectedPerson)
	checkPersonsEqual(t, expectedPerson, scannable)
}

func findAllTester(t *testing.T, query *zoom.MultiQuery, checker func(*testing.T, []zoom.Model)) {
	// execute the query
	results, err := query.Exec()
	if err != nil {
		t.Error(err)
	}

	// run the checker function
	checker(t, results)
}

func testFindAllWithExpectedIds(t *testing.T, query *zoom.MultiQuery, expectedIds []string, orderMatters bool) {
	findAllTester(t, query, func(t *testing.T, results []zoom.Model) {
		// make sure results is the right length
		if len(results) != len(expectedIds) {
			t.Errorf("results was not the right length. Expected: %d. Got: %d.\n", len(results), len(expectedIds))
		}

		// make sure the ids of the results match what we expected
		gotIds := []string{}
		for i, result := range results {
			// each item in results should be able to be casted to *Person
			pResult, ok := result.(*support.Person)
			if !ok {
				t.Errorf("Couldn't cast results[%d] to *Person", i)
			}
			// each item in results should have a valid Id
			if pResult.Id == "" {
				t.Error("Id was not set for Person: ", pResult)
			}
			gotIds = append(gotIds, pResult.Id)
		}

		if orderMatters {
			if !reflect.DeepEqual(expectedIds, gotIds) {
				t.Errorf("Person ids were incorrect.\nExpected: %v\nGot: %v\n", expectedIds, gotIds)
			}
		} else {
			equal, msg := util.CompareAsStringSet(expectedIds, gotIds)
			if !equal {
				t.Errorf("Person ids were incorrect\n%s\nExpected: %v\nGot: %v\n", msg, expectedIds, gotIds)
			}
		}
	})
}

func checkPersonsEqual(t *testing.T, expected, got *support.Person) {
	// make sure the found model is the same as original
	if got.Id != expected.Id {
		t.Errorf("Id was incorrect. Expected: %s. Got: %s.\n", expected.Id, got.Id)
	}
	if got.Name != expected.Name {
		t.Errorf("Name was incorrect. Expected: %s. Got: %s.\n", expected.Name, got.Name)
	}
	if got.Age != expected.Age {
		t.Errorf("Age was incorrect. Expected: %d. Got: %d.\n", expected.Age, got.Age)
	}
}
