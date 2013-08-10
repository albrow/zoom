package benchmark

import (
	"github.com/stephenalexbrowne/zoom"
	"testing"
)

func BenchmarkSave(b *testing.B) {

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	// try once to make sure there's no errors
	pt := NewPerson("Alice", 25)
	err = zoom.Save(pt)
	if err != nil {
		b.Error(err)
	}

	// reset the timer
	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		p := NewPerson("Bob", 25)
		zoom.Save(p)
	}

}

func BenchmarkFindById(b *testing.B) {

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	// save a Person model
	p := NewPerson("Clarence", 25)
	err = zoom.Save(p)
	if err != nil {
		b.Error(err)
	}
	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		zoom.FindById("person", p.Id)
	}
}

func BenchmarkDeleteById(b *testing.B) {

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	// First, create and delete one person to
	// make sure there's no errors
	p := NewPerson("Dennis", 25)
	err = zoom.Save(p)
	if err != nil {
		b.Error(err)
	}
	err = zoom.DeleteById("person", p.Id)
	if err != nil {
		b.Error(err)
	}

	// save a shit ton of person models
	// we don't know how big b.N will be,
	// so better be safe
	NUM_IDS := 100000
	ids := make([]string, 0, NUM_IDS)
	for i := 0; i < NUM_IDS; i++ {
		p := NewPerson("Fred", 25)
		err := zoom.Save(p)
		if err != nil {
			b.Error(err)
		}
		ids = append(ids, p.Id)
	}
	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		zoom.DeleteById("person", p.Id)
	}
}
