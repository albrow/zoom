package support

import (
	"github.com/stephenalexbrowne/zoom"
	"strconv"
)

func CreatePersons(num int) ([]*Person, error) {
	results := make([]*Person, num)
	for i := 0; i < num; i++ {
		p := &Person{
			Name: "person_" + strconv.Itoa(i),
			Age:  i,
		}
		if err := zoom.Save(p); err != nil {
			return results, err
		}
		results[i] = p
	}
	return results, nil
}

func CreateArtists(num int) ([]*Artist, error) {
	results := make([]*Artist, num)
	for i := 0; i < num; i++ {
		a := &Artist{
			Name: "artist_" + strconv.Itoa(i),
		}
		if err := zoom.Save(a); err != nil {
			return results, err
		}
		results[i] = a
	}
	return results, nil
}

func CreateColors(num int) ([]*Color, error) {
	results := make([]*Color, num)
	for i := 0; i < num; i++ {
		val := i % 255
		c := &Color{
			R: val,
			G: val,
			B: val,
		}
		if err := zoom.Save(c); err != nil {
			return results, err
		}
		results[i] = c
	}
	return results, nil
}
