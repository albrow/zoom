package benchmark

import (
	"github.com/stephenalexbrowne/zoom"
	"testing"
)

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
