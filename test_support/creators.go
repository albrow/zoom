// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File creators.go contains convenient constructors for the types in types.go

package test_support

import (
	"github.com/stephenalexbrowne/zoom"
	"github.com/stephenalexbrowne/zoom/util"
	"strconv"
)

// creates num persons but does not save them
func NewPersons(num int) ([]*Person, error) {
	results := make([]*Person, num)
	for i := 0; i < num; i++ {
		p := &Person{
			Name: "person_" + strconv.Itoa(i+1),
			Age:  i + 1,
		}
		results[i] = p
	}
	return results, nil
}

// creates and saves num persons
func CreatePersons(num int) ([]*Person, error) {
	results, err := NewPersons(num)
	if err != nil {
		return results, err
	}
	if err := zoom.Save(zoom.Models(results)...); err != nil {
		return results, err
	}
	return results, nil
}

func NewArtists(num int) ([]*Artist, error) {
	results := make([]*Artist, num)
	for i := 0; i < num; i++ {
		a := &Artist{
			Name: "artist_" + strconv.Itoa(i+1),
		}
		results[i] = a
	}
	return results, nil
}

func CreateArtists(num int) ([]*Artist, error) {
	results, err := NewArtists(num)
	if err != nil {
		return results, err
	}
	if err := zoom.Save(zoom.Models(results)...); err != nil {
		return results, err
	}
	return results, nil
}

func NewColors(num int) ([]*Color, error) {
	results := make([]*Color, num)
	for i := 0; i < num; i++ {
		val := i%254 + 1
		c := &Color{
			R: val,
			G: val,
			B: val,
		}
		results[i] = c
	}
	return results, nil
}

func CreateColors(num int) ([]*Color, error) {
	results, err := NewColors(num)
	if err != nil {
		return results, err
	}
	if err := zoom.Save(zoom.Models(results)...); err != nil {
		return results, err
	}
	return results, nil
}

func NewPetOwners(num int) ([]*PetOwner, error) {
	results := make([]*PetOwner, num)
	for i := 0; i < num; i++ {
		p := &PetOwner{
			Name: "petOwner_" + strconv.Itoa(i+1),
		}
		results[i] = p
	}
	return results, nil
}

func CreatePetOwners(num int) ([]*PetOwner, error) {
	results, err := NewPetOwners(num)
	if err != nil {
		return results, err
	}
	if err := zoom.Save(zoom.Models(results)...); err != nil {
		return results, err
	}
	return results, nil
}

func NewPets(num int) ([]*Pet, error) {
	results := make([]*Pet, num)
	for i := 0; i < num; i++ {
		p := &Pet{
			Name: "pet_" + strconv.Itoa(i+1),
		}
		results[i] = p
	}
	return results, nil
}

func CreatePets(num int) ([]*Pet, error) {
	results, err := NewPets(num)
	if err != nil {
		return results, err
	}
	if err := zoom.Save(zoom.Models(results)...); err != nil {
		return results, err
	}
	return results, nil
}

func NewFriends(num int) ([]*Friend, error) {
	results := make([]*Friend, num)
	for i := 0; i < num; i++ {
		f := &Friend{
			Name: "friend_" + strconv.Itoa(i),
		}
		results[i] = f
	}
	return results, nil
}

func CreateFriends(num int) ([]*Friend, error) {
	results, err := NewFriends(num)
	if err != nil {
		return results, err
	}
	if err := zoom.Save(zoom.Models(results)...); err != nil {
		return results, err
	}
	return results, nil
}

func NewConnectedFriends(num int) ([]*Friend, error) {
	friends, err := NewFriends(num)
	if err != nil {
		return friends, err
	}

	// randomly connect the friends to one another
	for i1, f1 := range friends {
		numFriends := util.RandInt(0, len(friends)-1)
		for n := 0; n < numFriends; n++ {
			i2 := util.RandInt(0, len(friends)-1)
			f2 := friends[i2]
			for i2 == i1 || friendListContains(f2, f1.Friends) {
				i2 = util.RandInt(0, len(friends)-1)
				f2 = friends[i2]
			}
			f1.Friends = append(f1.Friends, f2)
		}
	}

	return friends, nil
}

func CreateConnectedFriends(num int) ([]*Friend, error) {
	friends, err := CreateFriends(num)
	if err != nil {
		return friends, err
	}

	// randomly connect the friends to one another
	for i1, f1 := range friends {
		numFriends := util.RandInt(0, len(friends)-1)
		for n := 0; n < numFriends; n++ {
			i2 := util.RandInt(0, len(friends)-1)
			f2 := friends[i2]
			for i2 == i1 || friendListContains(f2, f1.Friends) {
				i2 = util.RandInt(0, len(friends)-1)
				f2 = friends[i2]
			}
			f1.Friends = append(f1.Friends, f2)
		}
	}

	// resave all the models
	if err := zoom.Save(zoom.Models(friends)...); err != nil {
		return friends, err
	}

	return friends, nil
}

func friendListContains(f *Friend, list []*Friend) bool {
	for _, e := range list {
		if e == f {
			return true
		}
	}
	return false
}

func NewPrimativeTypes(num int) ([]*PrimativeTypes, error) {
	results := make([]*PrimativeTypes, num)
	for i := 0; i < num; i++ {
		pt := &PrimativeTypes{
			Uint:    1,
			Uint8:   2,
			Uint16:  3,
			Uint32:  4,
			Uint64:  5,
			Int:     6,
			Int8:    7,
			Int16:   8,
			Int32:   9,
			Int64:   10,
			Float32: 11.0,
			Float64: 12.0,
			Byte:    13,
			Rune:    14,
			String:  "15",
		}
		results[i] = pt
	}
	return results, nil
}

func NewPointerPrimativeTypes(num int) ([]*PointerPrimativeTypes, error) {
	results := make([]*PointerPrimativeTypes, num)
	pUint := uint(1)
	pUint8 := uint8(2)
	pUint16 := uint16(3)
	pUint32 := uint32(4)
	pUint64 := uint64(5)
	pInt := int(6)
	pInt8 := int8(7)
	pInt16 := int16(8)
	pInt32 := int32(9)
	pInt64 := int64(10)
	pFloat32 := float32(11.0)
	pFloat64 := float64(12.0)
	pByte := byte(13)
	pRune := rune(14)
	pString := "15"
	for i := 0; i < num; i++ {
		ppt := &PointerPrimativeTypes{
			Uint:    &pUint,
			Uint8:   &pUint8,
			Uint16:  &pUint16,
			Uint32:  &pUint32,
			Uint64:  &pUint64,
			Int:     &pInt,
			Int8:    &pInt8,
			Int16:   &pInt16,
			Int32:   &pInt32,
			Int64:   &pInt64,
			Float32: &pFloat32,
			Float64: &pFloat64,
			Byte:    &pByte,
			Rune:    &pRune,
			String:  &pString,
		}
		results[i] = ppt
	}
	return results, nil
}

func NewInconvertibleTypes(num int) ([]*InconvertibleTypes, error) {
	results := make([]*InconvertibleTypes, num)
	for i := 0; i < num; i++ {
		m := &InconvertibleTypes{
			Complex:     complex128(1 + 2i),
			IntSlice:    []int{3, 4, 5},
			StringSlice: []string{"6", "7", "8"},
			IntArray:    [3]int{9, 10, 11},
			StringArray: [3]string{"12", "13", "14"},
			StringMap:   map[string]string{"15": "fifteen", "16": "sixteen"},
			IntMap:      map[int]int{17: 18, 19: 20},
		}
		results[i] = m
	}
	return results, nil
}

func NewEmbeddedStructs(num int) ([]*EmbeddedStruct, error) {
	results := make([]*EmbeddedStruct, num)
	for i := 0; i < num; i++ {
		m := &EmbeddedStruct{
			Embed: Embed{
				Int:    1,
				String: "foo",
			},
		}
		results[i] = m
	}
	return results, nil
}

func NewPointerEmbeddedStructs(num int) ([]*PointerEmbeddedStruct, error) {
	results := make([]*PointerEmbeddedStruct, num)
	for i := 0; i < num; i++ {
		m := &PointerEmbeddedStruct{
			Embed: &Embed{
				Int:    1,
				String: "foo",
			},
		}
		results[i] = m
	}
	return results, nil
}
