package test

import (
	"github.com/stephenalexbrowne/zoom"
	"github.com/stephenalexbrowne/zoom/redis"
	"github.com/stephenalexbrowne/zoom/support"
	"testing"
)

func TestSaveOneToOne(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save a new color
	c := &support.Color{R: 25, G: 152, B: 166}
	zoom.Save(c)

	// create and save a new artist, assigning favoriteColor to above
	a := &support.Artist{Name: "Alex", FavoriteColor: c}
	if err := zoom.Save(a); err != nil {
		t.Error(err)
	}

	// get a connection
	conn := zoom.GetConn()
	defer conn.Close()

	// invoke redis driver to check if the value was set appropriately
	colorKey := "artist:" + a.Id + ":FavoriteColor"
	id, err := redis.String(conn.Do("GET", colorKey))
	if err != nil {
		t.Error(err)
	}
	if id != c.Id {
		t.Errorf("color id for artist was not set correctly.\nExpected: %s\nGot: %s\n", c.Id, id)
	}
}

func TestFindOneToOne(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save a new color
	c := &support.Color{R: 25, G: 152, B: 166}
	zoom.Save(c)

	// create and save a new artist, assigning favoriteColor to above
	a := &support.Artist{Name: "Alex", FavoriteColor: c}
	if err := zoom.Save(a); err != nil {
		t.Error(err)
	}

	// find the saved person
	aCopy := &support.Artist{}
	if _, err := zoom.ScanById(aCopy, a.Id).Exec(); err != nil {
		t.Error(err)
	}

	// make sure favorite color is the same
	if aCopy.FavoriteColor == nil {
		t.Error("relation was not persisted. aCopy.FavoriteColor was nil")
	}
	if a.FavoriteColor.Id != aCopy.FavoriteColor.Id {
		t.Errorf("Id of favorite color was incorrect.\nExpected: %s\nGot: %s\n", a.FavoriteColor.Id, aCopy.FavoriteColor.Id)
	}
}
