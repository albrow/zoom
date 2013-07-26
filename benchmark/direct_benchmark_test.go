package benchmark

import (
	"testing"
)

func BenchmarkDirectSave(b *testing.B) {

	setUp()

	// try once to make sure there's no errors
	pt := NewDirectPerson("Bob", 25)
	err := pt.Save()
	if err != nil {
		b.Error(err)
	}

	// reset the timer
	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		p := NewDirectPerson("Bob", 25)
		p.Save()
	}

}

func BenchmarkDirectFindById(b *testing.B) {
	// save a Person model
	p := NewDirectPerson("Bob", 25)
	err := p.Save()
	if err != nil {
		b.Error(err)
	}
	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		findDirectPersonById(p.Id)
	}

}

func BenchmarkDirectOneToOneRelation(b *testing.B) {
	// save some peeps. Make them siblings.
	p1 := NewDirectPerson("Alice", 27)
	err := p1.Save()
	if err != nil {
		b.Error(err)
	}
	p2 := NewDirectPerson("Bob", 25)
	p2.SiblingId = p1.Id
	err = p2.Save()
	if err != nil {
		b.Error(err)
	}
	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		p2.FetchSibling()
	}
}
