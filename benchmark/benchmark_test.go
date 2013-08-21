package benchmark

import (
	"github.com/stephenalexbrowne/zoom"
	"testing"
)

// just get a connection and close it
func BenchmarkConnection(b *testing.B) {

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

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
func BenchmarkRepeatSave(b *testing.B) {
	singlePersonSelect := func(i int, persons []*Person) *Person {
		return persons[0]
	}
	benchmarkSave(b, 1, singlePersonSelect)
}

// sequentially saves a list of records
func BenchmarkSequentialSave(b *testing.B) {
	sequentialPersonSelect := func(i int, persons []*Person) *Person {
		return persons[i%len(persons)]
	}
	benchmarkSave(b, 1000, sequentialPersonSelect)
}

// finds the same record over and over
func BenchmarkRepeatFindById(b *testing.B) {
	benchmarkFindById(b, 1000, singleIdSelect)
}

// sequentially finds a list of records one by one
func BenchmarkSequentialFindById(b *testing.B) {
	benchmarkFindById(b, 1000, sequentialIdSelect)
}

// randomly finds records from a list
func BenchmarkRandomFindById(b *testing.B) {
	benchmarkFindById(b, 1000, randomIdSelect)
}

// finds the same record over and over,
// clearing the cache on each iteration
func BenchmarkRepeatFindByIdNoCache(b *testing.B) {
	benchmarkFindById(b, 1000, singleIdSelectNoCache)
}

// sequentially finds a list of records one by one,
// clearing the cache on each iteration
func BenchmarkSequentialFindByIdNoCache(b *testing.B) {
	benchmarkFindById(b, 1000, sequentialIdSelectNoCache)
}

// randomly finds records from a list,
// clearing the cache on each iteration
func BenchmarkRandomFindByIdNoCache(b *testing.B) {
	benchmarkFindById(b, 1000, randomIdSelectNoCache)
}

// repeatedly calls delete on a record
// (after the first, the record will have already been deleted)
func BenchmarkRepeatDeleteById(b *testing.B) {
	benchmarkDeleteById(b, 1, singleIdSelect)
}

// sequentially deletes a list of records one by one
func BenchmarkSequentialDeleteById(b *testing.B) {
	benchmarkDeleteById(b, 1000, sequentialIdSelect)
}

// randomly calls delete on a list of records
func BenchmarkRandomDeleteById(b *testing.B) {
	benchmarkDeleteById(b, 1000, randomIdSelect)
}

// calls FindAll for a dataset of size 10
func BenchmarkFindAll10(b *testing.B) {
	benchmarkFindAllWithCache(b, 10)
}

// calls FindAll for a dataset of size 100
func BenchmarkFindAll100(b *testing.B) {
	benchmarkFindAllWithCache(b, 100)
}

// calls FindAll for a dataset of size 1000
func BenchmarkFindAll1000(b *testing.B) {
	benchmarkFindAllWithCache(b, 1000)
}

// calls FindAll for a dataset of size 10000
func BenchmarkFindAll10000(b *testing.B) {
	benchmarkFindAllWithCache(b, 10000)
}

// calls FindAll for a dataset of size 10,
// clearing the cache on each iteration
func BenchmarkFindAllNoCache10(b *testing.B) {
	benchmarkFindAllNoCache(b, 10)
}

// calls FindAll for a dataset of size 100,
// clearing the cache on each iteration
func BenchmarkFindAllNoCache100(b *testing.B) {
	benchmarkFindAllNoCache(b, 100)
}

// calls FindAll for a dataset of size 1000,
// clearing the cache on each iteration
func BenchmarkFindAllNoCache1000(b *testing.B) {
	benchmarkFindAllNoCache(b, 1000)
}

// calls FindAll for a dataset of size 10000,
// clearing the cache on each iteration
func BenchmarkFindAllNoCache10000(b *testing.B) {
	benchmarkFindAllNoCache(b, 10000)
}

// create a parent with 10 children and repeatedly call Save on it
func BenchmarkRepeatSaveOneToMany10(b *testing.B) {
	benchmarkRepeatSaveOneToMany(b, 10)
}

// create a parent with 1,000 children and repeatedly call Save on it
func BenchmarkRepeatSaveOneToMany1000(b *testing.B) {
	benchmarkRepeatSaveOneToMany(b, 1000)
}

// create a list of parents with 10 children each. Then call
// FindById for randomly selected parents in the list.
func BenchmarkRandomFindOneToMany10(b *testing.B) {
	benchmarkFindOneToMany(b, 100, 10, randomIdSelect)
}

// create a list of parents with 1,000 children each. Then call
// FindById for randomly selected parents in the list.
func BenchmarkRandomFindOneToMany1000(b *testing.B) {
	benchmarkFindOneToMany(b, 100, 1000, randomIdSelect)
}

// create a list of parents with 10 children each. Then call
// FindById for randomly selected parents in the list.
func BenchmarkRandomFindOneToManyNoCache10(b *testing.B) {
	benchmarkFindOneToMany(b, 100, 10, randomIdSelectNoCache)
}

// create a list of parents with 1,000 children each. Then call
// FindById for randomly selected parents in the list.
func BenchmarkRandomFindOneToManyNoCache1000(b *testing.B) {
	benchmarkFindOneToMany(b, 100, 1000, randomIdSelectNoCache)
}
