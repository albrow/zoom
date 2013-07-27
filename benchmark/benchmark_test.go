package benchmark

import (
	"github.com/stephenalexbrowne/zoom"
	"testing"
)

func BenchmarkSave(b *testing.B) {

	setUp()

	// try once to make sure there's no errors
	pt := NewPerson("Bob", 25)
	err := zoom.Save(pt)
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

// func BenchmarkFindById(b *testing.B) {
// 	// save a Person model
// 	p := NewPerson("Bob", 25)
// 	err := p.Save()
// 	if err != nil {
// 		b.Error(err)
// 	}
// 	b.ResetTimer()

// 	// run the actual test
// 	for i := 0; i < b.N; i++ {
// 		zoom.FindById("person", p.Id)
// 	}

// }

// func BenchmarkOneToOneRelation(b *testing.B) {
// 	// save some peeps. Make them siblings.
// 	p1 := NewPerson("Alice", 27)
// 	err := p1.Save()
// 	if err != nil {
// 		b.Error(err)
// 	}
// 	p2 := NewPerson("Bob", 25)
// 	p2.SiblingId = p1.Id
// 	err = p2.Save()
// 	if err != nil {
// 		b.Error(err)
// 	}
// 	b.ResetTimer()

// 	// run the actual test
// 	for i := 0; i < b.N; i++ {
// 		p2.Fetch("sibling")
// 	}
// }
