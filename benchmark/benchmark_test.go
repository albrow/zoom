package benchmark

import (
	"fmt"
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

// The following benchmarks a sequential Create, Find, Update, and Delete
func BenchmarkCrud(b *testing.B) {

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	// create and save one invoice to make sure there's no errors
	inv := NewInvoice(100, 200, "my memo", 0, true)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := zoom.Save(inv)
		if err != nil {
			panic(err)
		}

		obj, err := zoom.FindById("invoice", inv.Id)
		if err != nil {
			panic(err)
		}

		inv2, ok := obj.(*Invoice)
		if !ok {
			panic(fmt.Sprintf("expected *Invoice, got: %v", obj))
		}

		inv2.Created = 1000
		inv2.Updated = 2000
		inv2.Memo = "my memo 2"
		inv2.PersonId = 3000
		err = zoom.Save(inv2)
		if err != nil {
			panic(err)
		}

		err = zoom.Delete(inv2)
		if err != nil {
			panic(err)
		}

	}

}

func BenchmarkDeleteById(b *testing.B) {

	b.Skip("This one takes much longer. Skipping for now.")

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
