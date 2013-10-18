// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

package test

import (
	"github.com/garyburd/redigo/redis"
	"github.com/stephenalexbrowne/zoom"
	"github.com/stephenalexbrowne/zoom/test_support"
	"github.com/stephenalexbrowne/zoom/util"
	"testing"
)

func TestSaveOneToOne(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save a new color
	c := &test_support.Color{R: 25, G: 152, B: 166}
	zoom.Save(c)

	// create and save a new artist, assigning favoriteColor to above
	a := &test_support.Artist{Name: "Alex", FavoriteColor: c}
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
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save a new color
	c := &test_support.Color{R: 25, G: 152, B: 166}
	zoom.Save(c)

	// create and save a new artist, assigning favoriteColor to above
	a := &test_support.Artist{Name: "Alex", FavoriteColor: c}
	zoom.Save(a)

	// find the saved person
	aCopy := &test_support.Artist{}
	if _, err := zoom.ScanById(a.Id, aCopy).Run(); err != nil {
		t.Error(err)
	}

	// make sure favorite color is the same
	equal, err := util.Equals(a, aCopy)
	if err != nil {
		t.Error(err)
	}
	if !equal {
		t.Errorf("artist was not saved/retrieved correctly.\nExpected: %+v\nGot: %+v\n", a, aCopy)
	}
}

func TestSaveOneToMany(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save a new petOwner
	owners, err := test_support.CreatePetOwners(1)
	if err != nil {
		t.Error(err)
	}
	o := owners[0]

	// create and save some pets
	pets, err := test_support.CreatePets(3)
	if err != nil {
		t.Error(err)
	}

	// assign the pets to the owner
	o.Pets = pets
	if err := zoom.Save(o); err != nil {
		t.Error(err)
	}

	// get a connection
	conn := zoom.GetConn()
	defer conn.Close()

	// invoke redis driver to check if the value was set appropriately
	petsKey := "petOwner:" + o.Id + ":Pets"
	gotIds, err := redis.Strings(conn.Do("SMEMBERS", petsKey))
	if err != nil {
		t.Error(err)
	}

	// compare expected ids to got ids
	expectedIds := make([]string, 0)
	for _, pet := range o.Pets {
		if pet.Id == "" {
			t.Errorf("pet id was empty for %+v\n", pet)
		}
		expectedIds = append(expectedIds, pet.Id)
	}
	equal, msg := util.CompareAsStringSet(expectedIds, gotIds)
	if !equal {
		t.Errorf("pet ids were not correct.\n%s\n", msg)
	}
}

func TestFindOneToMany(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save a new petOwner
	owners, _ := test_support.CreatePetOwners(1)
	o := owners[0]

	// create and save some pets
	pets, _ := test_support.CreatePets(3)

	// assign the pets to the owner
	o.Pets = pets
	zoom.Save(o)

	// get a copy of the owner from the database
	oCopy := &test_support.PetOwner{}
	if _, err := zoom.ScanById(o.Id, oCopy).Run(); err != nil {
		t.Error(err)
	}

	// compare expected ids to got ids
	expectedIds := make([]string, 0)
	for _, pet := range o.Pets {
		if pet.Id == "" {
			t.Errorf("pet id was empty for %+v\n", pet)
		}
		expectedIds = append(expectedIds, pet.Id)
	}
	gotIds := make([]string, 0)
	for _, pet := range oCopy.Pets {
		if pet.Id == "" {
			t.Errorf("pet id was empty for %+v\n", pet)
		}
		gotIds = append(gotIds, pet.Id)
	}
	equal, msg := util.CompareAsStringSet(expectedIds, gotIds)
	if !equal {
		t.Errorf("pet ids were not correct.\n%s\n", msg)
	}
}

func TestSaveManyToMany(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save some friends
	friends, err := test_support.CreateConnectedFriends(5)
	if err != nil {
		t.Error(err)
	}

	// get a connection
	conn := zoom.GetConn()
	defer conn.Close()

	for i, f := range friends {
		// invoke redis driver to check if the value was set appropriately
		friendsKey := "friend:" + f.Id + ":Friends"
		gotIds, err := redis.Strings(conn.Do("SMEMBERS", friendsKey))
		if err != nil {
			t.Error(err)
		}

		// compare expected ids to got ids
		expectedIds := make([]string, 0)
		for _, f2 := range f.Friends {
			if f2.Id == "" {
				t.Errorf("friend id was empty for %+v\n", f2)
			}
			expectedIds = append(expectedIds, f2.Id)
		}
		equal, msg := util.CompareAsStringSet(expectedIds, gotIds)
		if !equal {
			t.Errorf("friend ids for friend[%d] were not correct.\n%s\n", i, msg)
		}
	}

}

func TestFindManyToMany(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save some friends
	friends, err := test_support.CreateConnectedFriends(5)
	if err != nil {
		t.Error(err)
	}

	for i, f := range friends {

		// get a copy of the friend from the database
		fCopy := &test_support.Friend{}
		if _, err := zoom.ScanById(f.Id, fCopy).Run(); err != nil {
			t.Error(err)
		}

		// compare expected ids to got ids
		expectedIds := make([]string, 0)
		for _, f2 := range f.Friends {
			if f2.Id == "" {
				t.Errorf("iteration %d: friend:%s - id was empty for %+v\n", i, f.Id, f2)
			}
			expectedIds = append(expectedIds, f2.Id)
		}
		gotIds := make([]string, 0)
		for _, f2 := range fCopy.Friends {
			if f2.Id == "" {
				t.Errorf("fCopy on iteration %d: friend:%s - id was empty for %+v\n", i, fCopy.Id, f2)
			}
			gotIds = append(gotIds, f2.Id)
		}
		equal, msg := util.CompareAsStringSet(expectedIds, gotIds)
		if !equal {
			t.Errorf("on iteration %d.\nfriend:%s friend ids were not correct.\nExpected: %v\nGot: %v\n%s\n", i, fCopy.Id, expectedIds, gotIds, msg)
		}
	}
}

func TestFindOneToOneExclude(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save a new color
	c := &test_support.Color{R: 25, G: 152, B: 166}
	zoom.Save(c)

	// create and save a new artist, assigning favoriteColor to above
	a := &test_support.Artist{Name: "Alex", FavoriteColor: c}
	zoom.Save(a)

	// find the saved person
	aCopy := &test_support.Artist{}
	if _, err := zoom.ScanById(a.Id, aCopy).Exclude("FavoriteColor").Run(); err != nil {
		t.Error(err)
	}

	// make sure favorite color is nil
	if aCopy.FavoriteColor != nil {
		t.Errorf("excluded relation was not empty. aCopy.FavoriteColor was: ", aCopy.FavoriteColor)
	}

	// make sure Name was still set
	if aCopy.Name != "Alex" {
		t.Errorf("artist Name was incorrect.\nExpected: %s\nWas: %s\n", "Alex", aCopy.Name)
	}
}

func TestFindOneToManyExclude(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	// create and save a new petOwner
	owners, _ := test_support.CreatePetOwners(1)
	o := owners[0]

	// create and save some pets
	pets, _ := test_support.CreatePets(3)

	// assign the pets to the owner
	o.Pets = pets
	zoom.Save(o)

	// get a copy of the owner from the database
	oCopy := &test_support.PetOwner{}
	if _, err := zoom.ScanById(o.Id, oCopy).Exclude("Pets").Run(); err != nil {
		t.Error(err)
	}

	// make sure pets is nil
	if oCopy.Pets != nil {
		t.Errorf("excluded relation was not empty. oCopy.Pets was: ", oCopy.Pets)
	}

	// make sure name was still set
	if oCopy.Name != o.Name {
		t.Errorf("artist Name was incorrect.\nExpected: %s\nWas: %s\n", o.Name, oCopy.Name)
	}
}
