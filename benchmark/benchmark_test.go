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

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		conn := zoom.GetConn()
		reply, err := conn.Do("PING")
		b.StopTimer()
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
		b.StartTimer()
		conn.Close()
	}
}

// get a connection, call SET, wait for response, and close it
func BenchmarkSet(b *testing.B) {

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		conn := zoom.GetConn()
		_, err := conn.Do("SET", "foo", "bar")
		b.StopTimer()
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		conn.Close()
	}
}

// perform transactions with 10 SET commands each. Every 10th iteration,
// the transaction is executed with a call to EXEC
func BenchmarkSet10(b *testing.B) {
	benchmarkSetTransaction(b, 10)
}

// perform transactions with 100 SET commands each. Every 100th iteration,
// the transaction is executed with a call to EXEC
func BenchmarkSet100(b *testing.B) {
	benchmarkSetTransaction(b, 100)
}

// perform transactions with 1,000 SET commands each. Every 1,000th iteration,
// the transaction is executed with a call to EXEC
func BenchmarkSet1000(b *testing.B) {
	benchmarkSetTransaction(b, 1000)
}

// perform transactions with 10,000 SET commands each. Every 10,000th iteration,
// the transaction is executed with a call to EXEC
func BenchmarkSet10000(b *testing.B) {
	benchmarkSetTransaction(b, 10000)
}

// get a connection, call GET, wait for response, and close it
func BenchmarkGet(b *testing.B) {

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	conn := zoom.GetConn()
	_, err = conn.Do("SET", "foo", "bar")
	if err != nil {
		b.Fatal(err)
	}
	conn.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		conn := zoom.GetConn()
		reply, err := conn.Do("GET", "foo")
		b.StopTimer()
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
		b.StartTimer()
		conn.Close()
	}
}

// perform transactions with 10 GET commands each. Every 10th iteration,
// the transaction is executed with a call to EXEC
func BenchmarkGet10(b *testing.B) {
	benchmarkGetTransaction(b, 10)
}

// perform transactions with 100 GET commands each. Every 100th iteration,
// the transaction is executed with a call to EXEC
func BenchmarkGet100(b *testing.B) {
	benchmarkGetTransaction(b, 100)
}

// perform transactions with 1,000 GET commands each. Every 1,000th iteration,
// the transaction is executed with a call to EXEC
func BenchmarkGet1000(b *testing.B) {
	benchmarkGetTransaction(b, 1000)
}

// perform transactions with 10,000 GET commands each. Every 10,000th iteration,
// the transaction is executed with a call to EXEC
func BenchmarkGet10000(b *testing.B) {
	benchmarkGetTransaction(b, 10000)
}

// saves the same record repeatedly
// (after the first save, nothing changes)
func BenchmarkRepeatSave(b *testing.B) {

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	// try once to make sure there's no errors
	persons := createPersons(1)

	// reset the timer
	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		p := persons[0]
		b.StartTimer()
		zoom.Save(p)
	}

}

// sequentially saves a list of records
func BenchmarkSequentialSave(b *testing.B) {

	const NUM_PERSONS = 10000

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	// try once to make sure there's no errors
	pt := NewPerson("Amy", 25)
	err = zoom.Save(pt)
	if err != nil {
		b.Error(err)
	}

	// create a sequence of persons to be saved
	persons := createPersons(NUM_PERSONS)

	// reset the timer
	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		index := i % NUM_PERSONS
		p := persons[index]
		b.StartTimer()
		zoom.Save(p)
	}
}

// finds the same record over and over
func BenchmarkRepeatFindById(b *testing.B) {

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	ids := savePersons(1)

	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		id := ids[0]
		b.StartTimer()
		zoom.FindById("person", id)
	}
}

// sequentially finds a list of records one by one
func BenchmarkSequentialFindById(b *testing.B) {

	const NUM_PERSONS = 10000

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	ids := savePersons(NUM_PERSONS)

	// reset the timer
	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		index := i % NUM_PERSONS
		id := ids[index]
		b.StartTimer()
		zoom.FindById("person", id)
	}
}

// finds all the records in a list on by one in random order
func BenchmarkRandomFindById(b *testing.B) {

	const NUM_PERSONS = 10000

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	ids := savePersons(NUM_PERSONS)

	// reset the timer
	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		index := randInt(0, NUM_PERSONS)
		id := ids[index]
		b.StartTimer()
		zoom.FindById("person", id)
	}
}

// for the ..NoCache bencchmarks, we clear the cache before each Find
func BenchmarkRepeatFindByIdNoCache(b *testing.B) {

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	ids := savePersons(1)

	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		id := ids[0]
		zoom.ClearCache()
		b.StartTimer()
		zoom.FindById("person", id)
	}
}

// for the ..NoCache bencchmarks, we clear the cache before each Find
func BenchmarkSequentialFindByIdNoCache(b *testing.B) {

	const NUM_PERSONS = 10000

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	ids := savePersons(NUM_PERSONS)

	// reset the timer
	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		index := i % NUM_PERSONS
		id := ids[index]
		zoom.ClearCache()
		b.StartTimer()
		zoom.FindById("person", id)
	}
}

// for the ..NoCache bencchmarks, we clear the cache before each Find
func BenchmarkRandomFindByIdNoCache(b *testing.B) {

	const NUM_PERSONS = 10000

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	ids := savePersons(NUM_PERSONS)

	// reset the timer
	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		index := randInt(0, NUM_PERSONS)
		id := ids[index]
		zoom.ClearCache()
		b.StartTimer()
		zoom.FindById("person", id)
	}
}

// repeatedly calls delete on a record
// (after the first, the record will have already been deleted)
func BenchmarkRepeatDeleteById(b *testing.B) {

	const NUM_PERSONS = 1

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	ids := savePersons(NUM_PERSONS)

	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		id := ids[0]
		b.StartTimer()
		zoom.DeleteById("person", id)
	}
}

// sequentially deletes a list of records one by one
func BenchmarkSequentialDeleteById(b *testing.B) {

	const NUM_PERSONS = 10000

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	ids := savePersons(NUM_PERSONS)

	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		index := i % NUM_PERSONS
		id := ids[index]
		b.StartTimer()
		zoom.DeleteById("person", id)
	}
}

// deletes a list of records one by one in random order
func BenchmarkRandomDeleteById(b *testing.B) {

	const NUM_PERSONS = 10000

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	ids := savePersons(NUM_PERSONS)

	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		index := randInt(0, NUM_PERSONS)
		id := ids[index]
		b.StartTimer()
		zoom.DeleteById("person", id)
	}
}

// calls FindAll for a dataset of size 10
func BenchmarkFindAll10(b *testing.B) {
	benchmarkFindAll(b, 10)
}

// calls FindAll for a dataset of size 100
func BenchmarkFindAll100(b *testing.B) {
	benchmarkFindAll(b, 100)
}

// calls FindAll for a dataset of size 1000
func BenchmarkFindAll1000(b *testing.B) {
	benchmarkFindAll(b, 1000)
}

// calls FindAll for a dataset of size 10000
func BenchmarkFindAll10000(b *testing.B) {
	benchmarkFindAll(b, 10000)
}

// create a parent with 10 children and repeatedly call Save on it
func BenchmarkRepeatSaveOneToMany10(b *testing.B) {
	benchmarkRepeatSaveOneToMany(b, 10)
}

// create a parent with 100 children and repeatedly call Save on it
func BenchmarkRepeatSaveOneToMany100(b *testing.B) {
	benchmarkRepeatSaveOneToMany(b, 100)
}

// create a parent with 1,000 children and repeatedly call Save on it
func BenchmarkRepeatSaveOneToMany1000(b *testing.B) {
	benchmarkRepeatSaveOneToMany(b, 1000)
}

// create a parent with 10,000 children and repeatedly call Save on it
func BenchmarkRepeatSaveOneToMany10000(b *testing.B) {
	benchmarkRepeatSaveOneToMany(b, 10000)
}

// create a parent with 10 children and repeatedly call find on it
func BenchmarkRepeatFindOneToMany10(b *testing.B) {
	benchmarkRepeatFindOneToMany(b, 10)
}

// create a parent with 100 children and repeatedly call find on it
func BenchmarkRepeatFindOneToMany100(b *testing.B) {
	benchmarkRepeatFindOneToMany(b, 100)
}

// create a parent with 1,000 children and repeatedly call find on it
func BenchmarkRepeatFindOneToMany1000(b *testing.B) {
	benchmarkRepeatFindOneToMany(b, 1000)
}

// create a parent with 10,000 children and repeatedly call find on it
func BenchmarkRepeatFindOneToMany10000(b *testing.B) {
	benchmarkRepeatFindOneToMany(b, 10000)
}

// create a list of parents with 10 children each. Then iterate through
// the list sequentially calling FindById on each parent.
func BenchmarkSequentialFindOneToMany10(b *testing.B) {
	benchmarkSequentialFindOneToMany(b, 10)
}

// create a list of parents with 100 children each. Then iterate through
// the list sequentially calling FindById on each parent.
func BenchmarkSequentialFindOneToMany100(b *testing.B) {
	benchmarkSequentialFindOneToMany(b, 100)
}

// create a list of parents with 1,000 children each. Then iterate through
// the list sequentially calling FindById on each parent.
func BenchmarkSequentialFindOneToMany1000(b *testing.B) {
	benchmarkSequentialFindOneToMany(b, 1000)
}

// create a list of parents with 10 children each. Then iterate through
// the list in random order, calling FindById on each parent.
func BenchmarkRandomFindOneToMany10(b *testing.B) {
	benchmarkRandomFindOneToMany(b, 10)
}

// create a list of parents with 100 children each. Then iterate through
// the list in random order, calling FindById on each parent.
func BenchmarkRandomFindOneToMany100(b *testing.B) {
	benchmarkRandomFindOneToMany(b, 100)
}

// create a list of parents with 1,000 children each. Then iterate through
// the list in random order, calling FindById on each parent.
func BenchmarkRandomFindOneToMany1000(b *testing.B) {
	benchmarkRandomFindOneToMany(b, 1000)
}
