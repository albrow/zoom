package test

import (
	"github.com/stephenalexbrowne/zoom"
	"github.com/stephenalexbrowne/zoom/test_support"
	"github.com/stephenalexbrowne/zoom/util"
	"testing"
)

// just get a connection and close it
func BenchmarkConnection(b *testing.B) {

	test_support.SetUp()
	defer test_support.TearDown()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		conn := zoom.GetConn()
		conn.Close()
	}
}

// get a connection, call PING, wait for response, and close it
func BenchmarkPing(b *testing.B) {

	checkReply := func(reply interface{}, err error) {
		if err != nil {
			b.Fatal(err)
		}
		str, ok := reply.(string)
		if !ok {
			b.Fatal("couldn't convert reply to string")
		}
		if str != "PONG" {
			b.Fatal("reply was not PONG: ", str)
		}
	}

	benchmarkCommand(b, nil, checkReply, "PING")
}

// get a connection, call SET, wait for response, and close it
func BenchmarkSet(b *testing.B) {

	checkReply := func(reply interface{}, err error) {
		if err != nil {
			b.Fatal(err)
		}
	}

	benchmarkCommand(b, nil, checkReply, "SET", "foo", "bar")
}

// get a connection, call GET, wait for response, and close it
func BenchmarkGet(b *testing.B) {

	setup := func() {
		conn := zoom.GetConn()
		_, err := conn.Do("SET", "foo", "bar")
		if err != nil {
			b.Fatal(err)
		}
		conn.Close()
	}

	checkReply := func(reply interface{}, err error) {
		if err != nil {
			b.Fatal(err)
		}
		byt, ok := reply.([]byte)
		if !ok {
			b.Fatal("couldn't convert reply to []byte: ", reply)
		}
		str := string(byt)
		if str != "bar" {
			b.Fatal("reply was not bar: ", str)
		}
	}

	benchmarkCommand(b, setup, checkReply, "GET", "foo")
}

// saves the same record repeatedly
// (after the first save, nothing changes)
func BenchmarkSave(b *testing.B) {
	singlePersonSelect := func(i int, persons []*test_support.Person) *test_support.Person {
		return persons[0]
	}
	benchmarkSave(b, 1, singlePersonSelect)
}

// finds the same record over and over
func BenchmarkFindById(b *testing.B) {
	queryFunc := func(id string) zoom.Query { return zoom.FindById("person", id) }
	benchmarkFindQuery(b, 1000, singleIdSelect, queryFunc)
}

// scans the same record over and over
func BenchmarkScanById(b *testing.B) {
	queryFunc := func(id string) zoom.Query {
		p := &test_support.Person{}
		return zoom.ScanById(p, id)
	}
	benchmarkFindQuery(b, 1000, singleIdSelect, queryFunc)
}

// finds a record, but excludes the Name field
func BenchmarkFindByIdExclude(b *testing.B) {
	queryFunc := func(id string) zoom.Query { return zoom.FindById("person", id).Exclude("Name") }
	benchmarkFindQuery(b, 1000, singleIdSelect, queryFunc)
}

// repeatedly calls delete on a record
// (after the first, the record will have already been deleted)
func BenchmarkRepeatDeleteById(b *testing.B) {
	benchmarkDeleteById(b, 1, singleIdSelect)
}

// randomly calls delete on a list of records
func BenchmarkRandomDeleteById(b *testing.B) {
	benchmarkDeleteById(b, 1000, randomIdSelect)
}

// calls FindAll for a dataset of size 10
func BenchmarkFindAll10(b *testing.B) {
	test_support.SetUp()
	defer test_support.TearDown()

	benchmarkFindAllQuery(b, 10, zoom.FindAll("person"))
}

// calls FindAll for a dataset of size 100
func BenchmarkFindAll100(b *testing.B) {
	test_support.SetUp()
	defer test_support.TearDown()

	benchmarkFindAllQuery(b, 100, zoom.FindAll("person"))
}

// calls FindAll for a dataset of size 1,000
func BenchmarkFindAll1000(b *testing.B) {
	test_support.SetUp()
	defer test_support.TearDown()

	benchmarkFindAllQuery(b, 1000, zoom.FindAll("person"))
}

// calls FindAll for a dataset of size 10,000
func BenchmarkFindAll10000(b *testing.B) {
	test_support.SetUp()
	defer test_support.TearDown()

	benchmarkFindAllQuery(b, 10000, zoom.FindAll("person"))
}

// calls ScanAll for a dataset of size 10
func BenchmarkScanAll10(b *testing.B) {
	test_support.SetUp()
	defer test_support.TearDown()

	persons := make([]*test_support.Person, 0)
	benchmarkFindAllQuery(b, 10, zoom.ScanAll(&persons))
}

// calls ScanAll for a dataset of size 100
func BenchmarkScanAll100(b *testing.B) {
	test_support.SetUp()
	defer test_support.TearDown()

	persons := make([]*test_support.Person, 0)
	benchmarkFindAllQuery(b, 100, zoom.ScanAll(&persons))
}

// calls ScanAll for a dataset of size 1,000
func BenchmarkScanAll1000(b *testing.B) {
	test_support.SetUp()
	defer test_support.TearDown()

	persons := make([]*test_support.Person, 0)
	benchmarkFindAllQuery(b, 1000, zoom.ScanAll(&persons))
}

// calls ScanAll for a dataset of size 10,000
func BenchmarkScanAll10000(b *testing.B) {
	test_support.SetUp()
	defer test_support.TearDown()

	persons := make([]*test_support.Person, 0)
	benchmarkFindAllQuery(b, 10000, zoom.ScanAll(&persons))
}

// calls FindAll for a dataset of size 10, sorting by Age
func BenchmarkSortNumeric10(b *testing.B) {
	test_support.SetUp()
	defer test_support.TearDown()

	benchmarkFindAllQuery(b, 10, zoom.FindAll("person").SortBy("Age"))
}

// calls FindAll for a dataset of size 100, sorting by Age
func BenchmarkSortNumeric100(b *testing.B) {
	test_support.SetUp()
	defer test_support.TearDown()

	benchmarkFindAllQuery(b, 100, zoom.FindAll("person").SortBy("Age"))
}

// calls FindAll for a dataset of size 1,000, sorting by Age
func BenchmarkSortNumeric1000(b *testing.B) {
	test_support.SetUp()
	defer test_support.TearDown()

	benchmarkFindAllQuery(b, 1000, zoom.FindAll("person").SortBy("Age"))
}

// calls FindAll for a dataset of size 10,000, sorting by Age
func BenchmarkSortNumeric10000(b *testing.B) {
	test_support.SetUp()
	defer test_support.TearDown()

	benchmarkFindAllQuery(b, 10000, zoom.FindAll("person").SortBy("Age"))
}

// calls FindAll for a dataset of size 10,000, sorting by Age, limiting to 1 result
func BenchmarkSortNumeric10000Limit1(b *testing.B) {
	test_support.SetUp()
	defer test_support.TearDown()

	benchmarkFindAllQuery(b, 10000, zoom.FindAll("person").SortBy("Age").Limit(1))
}

// calls FindAll for a dataset of size 10,000, sorting by Age, limiting to 10 results
func BenchmarkSortNumeric10000Limit10(b *testing.B) {
	test_support.SetUp()
	defer test_support.TearDown()

	benchmarkFindAllQuery(b, 10000, zoom.FindAll("person").SortBy("Age").Limit(10))
}

// calls FindAll for a dataset of size 10,000, sorting by Age, limiting to 100 results
func BenchmarkSortNumeric10000Limit100(b *testing.B) {
	test_support.SetUp()
	defer test_support.TearDown()

	benchmarkFindAllQuery(b, 10000, zoom.FindAll("person").SortBy("Age").Limit(100))
}

// calls FindAll for a dataset of size 10,000, sorting by Age, limiting to 1,000 results
func BenchmarkSortNumeric10000Limit1000(b *testing.B) {
	test_support.SetUp()
	defer test_support.TearDown()

	benchmarkFindAllQuery(b, 10000, zoom.FindAll("person").SortBy("Age").Limit(1000))
}

// calls FindAll for a dataset of size 10, sorting by Name
func BenchmarkSortAlpha10(b *testing.B) {
	test_support.SetUp()
	defer test_support.TearDown()

	benchmarkFindAllQuery(b, 10, zoom.FindAll("person").SortBy("Name"))
}

// calls FindAll for a dataset of size 100, sorting by Name
func BenchmarkSortAlpha100(b *testing.B) {
	test_support.SetUp()
	defer test_support.TearDown()

	benchmarkFindAllQuery(b, 100, zoom.FindAll("person").SortBy("Name"))
}

// calls FindAll for a dataset of size 1000, sorting by Name
func BenchmarkSortAlpha1000(b *testing.B) {
	test_support.SetUp()
	defer test_support.TearDown()

	benchmarkFindAllQuery(b, 1000, zoom.FindAll("person").SortBy("Name"))
}

// calls FindAll for a dataset of size 10000, sorting by Name
func BenchmarkSortAlpha10000(b *testing.B) {
	test_support.SetUp()
	defer test_support.TearDown()

	benchmarkFindAllQuery(b, 10000, zoom.FindAll("person").SortBy("Name"))
}

func benchmarkCommand(b *testing.B, setup func(), checkReply func(interface{}, error), cmd string, args ...interface{}) {
	test_support.SetUp()
	defer test_support.TearDown()

	if setup != nil {
		setup()
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		conn := zoom.GetConn()
		reply, err := conn.Do(cmd, args...)
		b.StopTimer()
		if checkReply != nil {
			checkReply(reply, err)
		}
		b.StartTimer()
		conn.Close()
	}
}

func benchmarkSave(b *testing.B, num int, personSelect func(int, []*test_support.Person) *test_support.Person) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create a sequence of persons to be saved
	persons, err := test_support.NewPersons(num)
	if err != nil {
		b.Error(err)
	}

	// reset the timer
	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		p := personSelect(i, persons)
		b.StartTimer()
		err := zoom.Save(p)
		b.StopTimer()
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
	}
}

func benchmarkFindQuery(b *testing.B, num int, idSelect func(int, []string) string, queryFunc func(string) zoom.Query) {
	test_support.SetUp()
	defer test_support.TearDown()

	persons, err := test_support.CreatePersons(num)
	if err != nil {
		b.Error(err)
	}
	ids := make([]string, len(persons))
	for i, p := range persons {
		ids[i] = p.Id
	}

	// reset the timer
	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		id := idSelect(i, ids)
		q := queryFunc(id)
		b.StartTimer()
		q.Run()
	}
}

func benchmarkDeleteById(b *testing.B, num int, idSelect func(int, []string) string) {
	test_support.SetUp()
	defer test_support.TearDown()

	persons, err := test_support.CreatePersons(num)
	if err != nil {
		b.Error(err)
	}
	ids := make([]string, len(persons))
	for i, p := range persons {
		ids[i] = p.Id
	}

	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		id := idSelect(i, ids)
		b.StartTimer()
		zoom.DeleteById("person", id)
	}
}

func benchmarkDelete(b *testing.B, num int, personSelect func(int, []*test_support.Person) *test_support.Person) {
	test_support.SetUp()
	defer test_support.TearDown()

	persons, err := test_support.CreatePersons(num)
	if err != nil {
		b.Error(err)
	}

	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		p := personSelect(i, persons)
		b.StartTimer()
		zoom.Delete(p)
	}
}

func benchmarkFindAllQuery(b *testing.B, num int, q zoom.Query) {
	_, err := test_support.CreatePersons(num)
	if err != nil {
		b.Error(err)
	}

	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		_, err := q.Run()
		b.StopTimer()
		if err != nil {
			b.Error(err)
		}
		b.StartTimer()
	}
}

func singleIdSelect(i int, ids []string) string {
	return ids[0]
}

func sequentialIdSelect(i int, ids []string) string {
	return ids[i%len(ids)]
}

func randomIdSelect(i int, ids []string) string {
	return ids[util.RandInt(0, len(ids)-1)]
}
