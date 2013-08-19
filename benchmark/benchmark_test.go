package benchmark

import (
	"github.com/stephenalexbrowne/zoom"
	"testing"
)

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

// deletes a list of records on by one in random order
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

	const NUM_PERSONS = 10

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	savePersons(NUM_PERSONS)

	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		zoom.FindAll("person")
	}
}

// calls FindAll for a dataset of size 100
func BenchmarkFindAll100(b *testing.B) {

	const NUM_PERSONS = 100

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	savePersons(NUM_PERSONS)

	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		zoom.FindAll("person")
	}
}

// calls FindAll for a dataset of size 1000
func BenchmarkFindAll1000(b *testing.B) {

	const NUM_PERSONS = 1000

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	savePersons(NUM_PERSONS)

	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		zoom.FindAll("person")
	}
}

// calls FindAll for a dataset of size 10000
func BenchmarkFindAll10000(b *testing.B) {

	const NUM_PERSONS = 10000

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	savePersons(NUM_PERSONS)

	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		zoom.FindAll("person")
	}
}
