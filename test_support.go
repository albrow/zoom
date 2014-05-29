// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

package zoom

import (
	"flag"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"math/rand"
	"sort"
	"strconv"
	"testing"
)

type basicModel struct {
	Attr string
	DefaultData
}

type modelWithList struct {
	List []string `redisType:"list"`
	DefaultData
}

type modelWithSet struct {
	Set []string `redisType:"set"`
	DefaultData
}

type oneToOneModelDifferentType struct {
	Attr string
	One  *basicModel
	DefaultData
}

type oneToOneModelSameType struct {
	Attr string
	One  *oneToOneModelSameType
	DefaultData
}

type oneToManyModelDifferentType struct {
	Attr string
	Many []*basicModel
	DefaultData
}

type manyToManyModelDifferentTypeOne struct {
	Attr string
	Many []*manyToManyModelDifferentTypeTwo
	DefaultData
}

type manyToManyModelDifferentTypeTwo struct {
	Attr string
	Many []*manyToManyModelDifferentTypeOne
	DefaultData
}

type manyToManyModelSameType struct {
	Attr string
	Many []*manyToManyModelSameType
	DefaultData
}

type primativeTypesModel struct {
	Uint    uint
	Uint8   uint8
	Uint16  uint16
	Uint32  uint32
	Uint64  uint64
	Int     int
	Int8    int8
	Int16   int16
	Int32   int32
	Int64   int64
	Float32 float32
	Float64 float64
	Byte    byte
	Rune    rune
	String  string
	Bool    bool
	DefaultData
}

type pointersToPrimativeTypesModel struct {
	Uint    *uint
	Uint8   *uint8
	Uint16  *uint16
	Uint32  *uint32
	Uint64  *uint64
	Int     *int
	Int8    *int8
	Int16   *int16
	Int32   *int32
	Int64   *int64
	Float32 *float32
	Float64 *float64
	Byte    *byte
	Rune    *rune
	String  *string
	Bool    *bool
	DefaultData
}

type inconvertibleTypesModel struct {
	Complex     complex128
	IntSlice    []int
	StringSlice []string
	IntArray    [3]int
	StringArray [3]string
	StringMap   map[string]string
	IntMap      map[int]int
	DefaultData
}

type embeddedStructModel struct {
	embed
	DefaultData
}

type embeddedPointerToStructModel struct {
	*embed
	DefaultData
}

type embed struct {
	Int    int
	String string
}

type indexedPrimativesModel struct {
	Uint    uint    `zoom:"index" json:"-"`
	Uint8   uint8   `zoom:"index" json:"-"`
	Uint16  uint16  `zoom:"index" json:"-"`
	Uint32  uint32  `zoom:"index" json:"-"`
	Uint64  uint64  `zoom:"index" json:"-"`
	Int     int     `zoom:"index"`
	Int8    int8    `zoom:"index" json:"-"`
	Int16   int16   `zoom:"index" json:"-"`
	Int32   int32   `zoom:"index" json:"-"`
	Int64   int64   `zoom:"index" json:"-"`
	Float32 float32 `zoom:"index" json:"-"`
	Float64 float64 `zoom:"index" json:"-"`
	Byte    byte    `zoom:"index" json:"-"`
	Rune    rune    `zoom:"index" json:"-"`
	String  string  `zoom:"index"`
	Bool    bool    `zoom:"index"`
	DefaultData
}

type indexedPointersModel struct {
	Uint    *uint    `zoom:"index"`
	Uint8   *uint8   `zoom:"index"`
	Uint16  *uint16  `zoom:"index"`
	Uint32  *uint32  `zoom:"index"`
	Uint64  *uint64  `zoom:"index"`
	Int     *int     `zoom:"index"`
	Int8    *int8    `zoom:"index"`
	Int16   *int16   `zoom:"index"`
	Int32   *int32   `zoom:"index"`
	Int64   *int64   `zoom:"index"`
	Float32 *float32 `zoom:"index"`
	Float64 *float64 `zoom:"index"`
	Byte    *byte    `zoom:"index"`
	Rune    *rune    `zoom:"index"`
	String  *string  `zoom:"index"`
	Bool    *bool    `zoom:"index"`
	DefaultData
}

var address *string = flag.String("address", "localhost:6379", "the address of a redis server to connect to")
var network *string = flag.String("network", "tcp", "the network to use for the database connection (e.g. 'tcp' or 'unix')")
var database *int = flag.Int("database", 9, "the redis database number to use for testing")

var testingTypes []Model = []Model{
	&basicModel{},
	&modelWithList{},
	&modelWithSet{},
	&oneToOneModelDifferentType{},
	&oneToOneModelSameType{},
	&oneToManyModelDifferentType{},
	&manyToManyModelDifferentTypeOne{},
	&manyToManyModelDifferentTypeTwo{},
	&manyToManyModelSameType{},
	&primativeTypesModel{},
	&pointersToPrimativeTypesModel{},
	&inconvertibleTypesModel{},
	&embeddedStructModel{},
	&embeddedPointerToStructModel{},
	&indexedPrimativesModel{},
	&indexedPointersModel{},
}

func testingSetUp() {
	conn := testingConnect()
	defer conn.Close()

	registerTestingTypes()

	// make sure database is empty
	n, err := redis.Int(conn.Do("DBSIZE"))
	if err != nil {
		panic(err.Error())
	}
	if n != 0 {
		err := fmt.Errorf("Database #%d is not empty, test can not continue", *database)
		panic(err)
	}

	// use a seed for rand to make test runs consistent
	rand.Seed(42)
}

// initialize zoom and test the connection
func testingConnect() redis.Conn {
	testingDial()
	conn := GetConn()
	if err := testConn(conn); err != nil {
		panic(err)
	}
	return conn
}

func testingDial() {
	Init(&Configuration{
		Address:  *address,
		Network:  *network,
		Database: *database,
	})
}

func testConn(conn redis.Conn) error {
	if _, err := conn.Do("PING"); err != nil {
		return err
	}
	return nil
}

func registerTestingTypes() {
	for _, m := range testingTypes {
		if err := Register(m); err != nil {
			panic(err)
		}
	}
}

func testingTearDown() {
	unregisterTestingTypes()

	// flush and close the database
	conn := GetConn()
	_, err := conn.Do("flushdb")
	if err != nil {
		panic(err)
	}
	conn.Close()
	Close()
}

func unregisterTestingTypes() {
	for _, m := range testingTypes {
		Unregister(m)
	}
}

func newBasicModels(num int) ([]*basicModel, error) {
	results := make([]*basicModel, num)
	for i := 0; i < num; i++ {
		m := &basicModel{
			Attr: "test_" + strconv.Itoa(i+1),
		}
		results[i] = m
	}
	return results, nil
}

func newPrimativeTypesModels(num int) ([]*primativeTypesModel, error) {
	results := make([]*primativeTypesModel, num)
	for i := 0; i < num; i++ {
		pt := &primativeTypesModel{
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
			Bool:    true,
		}
		results[i] = pt
	}
	return results, nil
}

func newPointersToPrimativeTypesModels(num int) ([]*pointersToPrimativeTypesModel, error) {
	results := make([]*pointersToPrimativeTypesModel, num)
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
	pBool := true
	for i := 0; i < num; i++ {
		ppt := &pointersToPrimativeTypesModel{
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
			Bool:    &pBool,
		}
		results[i] = ppt
	}
	return results, nil
}

func newInconvertibleTypesModels(num int) ([]*inconvertibleTypesModel, error) {
	results := make([]*inconvertibleTypesModel, num)
	for i := 0; i < num; i++ {
		m := &inconvertibleTypesModel{
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

func newIndexedPrimativesModels(num int) ([]*indexedPrimativesModel, error) {
	results := make([]*indexedPrimativesModel, num)
	for i := 0; i < num; i++ {
		m := &indexedPrimativesModel{
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
			Bool:    true,
		}
		results[i] = m
	}
	return results, nil
}

func newIndexedPointersModels(num int) ([]*indexedPointersModel, error) {
	results := make([]*indexedPointersModel, num)
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
	pBool := true
	for i := 0; i < num; i++ {
		ppt := &indexedPointersModel{
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
			Bool:    &pBool,
		}
		results[i] = ppt
	}
	return results, nil
}

// utility type for quickly sorting by id
type ById []*indexedPrimativesModel

func (ms ById) Len() int           { return len(ms) }
func (ms ById) Swap(i, j int)      { ms[i], ms[j] = ms[j], ms[i] }
func (ms ById) Less(i, j int) bool { return ms[i].Id < ms[j].Id }

// returns true if the models are equal, false if they are not.
// also calls.Error with a specific error message if they don't match
func compareModelSlices(t *testing.T, expected []*indexedPrimativesModel, got []*indexedPrimativesModel, orderMatters bool) bool {
	if len(expected) != len(got) {
		t.Errorf("Lengths did not match. Expected: %d but got: %d\n", len(expected), len(got))
		_, msg := compareAsStringSet(modelIds(Models(expected)), modelIds(Models(got)))
		t.Error(msg)
		return false
	}
	eCopy, gCopy := make([]*indexedPrimativesModel, len(expected)), make([]*indexedPrimativesModel, len(got))
	copy(eCopy, expected)
	copy(gCopy, got)
	if !orderMatters {
		// if order doesn't matter, first sort by Id, which is unique.
		// this way we can do a straightforward comparison
		sort.Sort(ById(eCopy))
		sort.Sort(ById(gCopy))
	}
	for i, e := range eCopy {
		g := gCopy[i]
		if eql, msgs := compareModels(e, g); !eql {
			t.Errorf("Inequality detected at iteration %d:", i)
			for _, msg := range msgs {
				t.Errorf("\t%s", msg)
			}
			return false
		}
	}
	return true
}

func compareModels(expected, got *indexedPrimativesModel) (bool, []string) {
	eql := true
	msgs := []string{}

	// Id field
	if expected.Id != got.Id {
		eql = false
		msg := fmt.Sprintf("Id was incorrect. Expected: %s but got: %s\n", expected.Id, got.Id)
		msgs = append(msgs, msg)
	}

	// Uint fields
	if expected.Uint != got.Uint {
		eql = false
		msg := fmt.Sprintf("Uint was incorrect. Expected: %v but got: %v\n", expected.Uint, got.Uint)
		msgs = append(msgs, msg)
	}
	if expected.Uint8 != got.Uint8 {
		eql = false
		msg := fmt.Sprintf("Uint8 was incorrect. Expected: %v but got: %v\n", expected.Uint8, got.Uint8)
		msgs = append(msgs, msg)
	}
	if expected.Uint16 != got.Uint16 {
		eql = false
		msg := fmt.Sprintf("Uint16 was incorrect. Expected: %v but got: %v\n", expected.Uint16, got.Uint16)
		msgs = append(msgs, msg)
	}
	if expected.Uint32 != got.Uint32 {
		eql = false
		msg := fmt.Sprintf("Uint32 was incorrect. Expected: %v but got: %v\n", expected.Uint32, got.Uint32)
		msgs = append(msgs, msg)
	}
	if expected.Uint64 != got.Uint64 {
		eql = false
		msg := fmt.Sprintf("Uint64 was incorrect. Expected: %v but got: %v\n", expected.Uint64, got.Uint64)
		msgs = append(msgs, msg)
	}
	if expected.Id != got.Id {
		eql = false
		msg := fmt.Sprintf("Id was incorrect. Expected: %v but got: %v\n", expected.Id, got.Id)
		msgs = append(msgs, msg)
	}

	// Int fields
	if expected.Int != got.Int {
		eql = false
		msg := fmt.Sprintf("Int was incorrect. Expected: %v but got: %v\n", expected.Int, got.Int)
		msgs = append(msgs, msg)
	}
	if expected.Int8 != got.Int8 {
		eql = false
		msg := fmt.Sprintf("Int8 was incorrect. Expected: %v but got: %v\n", expected.Int8, got.Int8)
		msgs = append(msgs, msg)
	}
	if expected.Int16 != got.Int16 {
		eql = false
		msg := fmt.Sprintf("Int16 was incorrect. Expected: %v but got: %v\n", expected.Int16, got.Int16)
		msgs = append(msgs, msg)
	}
	if expected.Int32 != got.Int32 {
		eql = false
		msg := fmt.Sprintf("Int32 was incorrect. Expected: %v but got: %v\n", expected.Int32, got.Int32)
		msgs = append(msgs, msg)
	}
	if expected.Int64 != got.Int64 {
		eql = false
		msg := fmt.Sprintf("Int64 was incorrect. Expected: %v but got: %v\n", expected.Int64, got.Int64)
		msgs = append(msgs, msg)
	}

	// Other fields
	if expected.Float32 != got.Float32 {
		eql = false
		msg := fmt.Sprintf("Float32 was incorrect. Expected: %f but got: %f\n", expected.Float32, got.Float32)
		msgs = append(msgs, msg)
	}
	if expected.Float64 != got.Float64 {
		eql = false
		msg := fmt.Sprintf("Float64 was incorrect. Expected: %f but got: %f\n", expected.Float64, got.Float64)
		msgs = append(msgs, msg)
	}
	if expected.Byte != got.Byte {
		eql = false
		msg := fmt.Sprintf("Byte was incorrect. Expected: %v but got: %v\n", expected.Byte, got.Byte)
		msgs = append(msgs, msg)
	}
	if expected.Rune != got.Rune {
		eql = false
		msg := fmt.Sprintf("Rune was incorrect. Expected: %v but got: %v\n", expected.Rune, got.Rune)
		msgs = append(msgs, msg)
	}
	if expected.String != got.String {
		eql = false
		msg := fmt.Sprintf("String was incorrect. Expected: %s but got: %s\n", expected.String, got.String)
		msgs = append(msgs, msg)
	}
	if expected.Bool != got.Bool {
		eql = false
		msg := fmt.Sprintf("Bool was incorrect. Expected: %t but got: %t\n", expected.Bool, got.Bool)
		msgs = append(msgs, msg)
	}

	return eql, msgs
}
