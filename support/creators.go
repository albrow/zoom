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
