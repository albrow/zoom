// Copyright 2014 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File util.go contains miscellaneous utility functions used throughout
// the zoom library.

package zoom

import (
	"encoding/json"
	"fmt"
	"github.com/dchest/uniuri"
	"math/rand"
	"reflect"
	"strconv"
	"time"
)

// Models converts an interface to a slice of Model. It is typically
// used to convert a return value of a Query. Will panic if the type
// is invalid.
func Models(in interface{}) []Model {
	typ := reflect.TypeOf(in)
	if !typeIsSliceOrArray(typ) {
		msg := fmt.Sprintf("zoom: panic in Models() - attempt to convert invalid type %T to []Model.\nArgument must be slice or array.", in)
		panic(msg)
	}
	elemTyp := typ.Elem()
	if !typeIsPointerToStruct(elemTyp) {
		msg := fmt.Sprintf("zoom: panic in Models() - attempt to convert invalid type %T to []Model.\nSlice or array must have elements of type pointer to struct.", in)
		panic(msg)
	}
	_, found := modelTypeToSpec[elemTyp]
	if !found {
		msg := fmt.Sprintf("zoom: panic in Models() - attempt to convert invalid type %T to []Model.\nType %s is not registered.", in, elemTyp)
		panic(msg)
	}
	val := reflect.ValueOf(in)
	length := val.Len()
	results := make([]Model, length)
	for i := 0; i < length; i++ {
		elemVal := val.Index(i)
		model, ok := elemVal.Interface().(Model)
		if !ok {
			msg := fmt.Sprintf("zoom: panic in Models() - cannot convert type %T to Model", elemVal.Interface())
			panic(msg)
		}
		results[i] = model
	}
	return results
}

// Interfaces converts in to []interface{}. It will panic if the type
// of in is not a slice or array.
func Interfaces(in interface{}) []interface{} {
	val := reflect.ValueOf(in)
	length := val.Len()
	results := make([]interface{}, length)
	for i := 0; i < length; i++ {
		elemVal := val.Index(i)
		results[i] = elemVal.Interface()
	}
	return results
}

func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func indexOfStringSlice(a string, list []string) int {
	for i, b := range list {
		if b == a {
			return i
		}
	}
	return -1
}

func indexOfSlice(a interface{}, list interface{}) int {
	lVal := reflect.ValueOf(list)
	size := lVal.Len()
	for i := 0; i < size; i++ {
		elem := lVal.Index(i)
		if reflect.DeepEqual(a, elem.Interface()) {
			return i
		}
	}
	return -1
}

func stringSliceContains(a string, list []string) bool {
	return indexOfStringSlice(a, list) != -1
}

func sliceContains(a interface{}, list interface{}) bool {
	return indexOfSlice(a, list) != -1
}

func removeFromStringSlice(list []string, i int) []string {
	return append(list[:i], list[i+1:]...)
}

func removeElementFromStringSlice(list []string, elem string) []string {
	for i, e := range list {
		if e == elem {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

func compareAsStringSet(expecteds, gots []string) (bool, string) {
	// make sure everything in expecteds is also in gots
	for _, e := range expecteds {
		index := indexOfStringSlice(e, gots)
		if index == -1 {
			msg := fmt.Sprintf("Missing expected element: %v", e)
			return false, msg
		}
	}

	// make sure everything in gots is also in expecteds
	for _, g := range gots {
		index := indexOfStringSlice(g, expecteds)
		if index == -1 {
			msg := fmt.Sprintf("Found extra element: %v", g)
			return false, msg
		}
	}

	return true, "ok"
}

func compareAsSet(expecteds, gots interface{}) (bool, string) {
	eVal := reflect.ValueOf(expecteds)
	gVal := reflect.ValueOf(gots)

	if !eVal.IsValid() {
		return false, "expecteds was nil"
	} else if !gVal.IsValid() {
		return false, "gots was nil"
	}

	// make sure everything in expecteds is also in gots
	for i := 0; i < eVal.Len(); i++ {
		expected := eVal.Index(i).Interface()
		index := indexOfSlice(expected, gots)
		if index == -1 {
			msg := fmt.Sprintf("Missing expected element: %v", expected)
			return false, msg
		}
	}

	// make sure everything in gots is also in expecteds
	for i := 0; i < gVal.Len(); i++ {
		got := gVal.Index(i).Interface()
		index := indexOfSlice(got, expecteds)
		if index == -1 {
			msg := fmt.Sprintf("Found unexpected element: %v", got)
			return false, msg
		}
	}
	return true, "ok"
}

func typeIsSliceOrArray(typ reflect.Type) bool {
	k := typ.Kind()
	return (k == reflect.Slice || k == reflect.Array) && typ.Elem().Kind() != reflect.Uint8
}

func typeIsPointerToStruct(typ reflect.Type) bool {
	return typ.Kind() == reflect.Ptr && typ.Elem().Kind() == reflect.Struct
}

func typeIsString(typ reflect.Type) bool {
	k := typ.Kind()
	return k == reflect.String || ((k == reflect.Slice || k == reflect.Array) && typ.Elem().Kind() == reflect.Uint8)
}

func typeIsNumeric(typ reflect.Type) bool {
	k := typ.Kind()
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func typeIsBool(typ reflect.Type) bool {
	k := typ.Kind()
	return k == reflect.Bool
}

func typeIsPrimative(typ reflect.Type) bool {
	return typeIsString(typ) || typeIsNumeric(typ) || typeIsBool(typ)
}

// looseEquals returns true if the two things are equal.
// equality is based on underlying value, so if the pointer addresses
// are different it doesn't matter. We use gob encoding for simplicity,
// assuming that if the gob representation of two things is the same,
// those two things can be considered equal. Differs from reflect.DeepEqual
// because of the indifference concerning pointer addresses.
func looseEquals(one, two interface{}) (bool, error) {
	oneBytes, err := json.Marshal(one)
	if err != nil {
		return false, err
	}
	twoBytes, err := json.Marshal(two)
	if err != nil {
		return false, err
	}

	return (string(oneBytes) == string(twoBytes)), nil
}

func convertNumericToFloat64(val reflect.Value) float64 {
	switch val.Type().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		integer := val.Int()
		return float64(integer)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uinteger := val.Uint()
		return float64(uinteger)
	case reflect.Float32, reflect.Float64:
		return val.Float()
	default:
		msg := fmt.Sprintf("zoom: attempt to call convertNumericToFloat64 on non-numeric type %s", val.Type().String())
		panic(msg)
	}
}

func modelIds(ms []Model) []string {
	results := make([]string, len(ms))
	for i, m := range ms {
		results[i] = m.GetId()
	}
	return results
}

// converts a bool to an int using the following rule:
// false = 0
// true = 1
func boolToInt(b bool) int {
	if b {
		return 1
	} else {
		return 0
	}
}

// generateRandomId generates a random string that is more or less
// garunteed to be unique. Used as Ids for records where an Id is
// not otherwise provided.
func generateRandomId() string {
	timeInt := time.Now().Unix()
	timeString := strconv.FormatInt(timeInt, 36)
	randomString := uniuri.NewLen(16)
	return randomString + timeString
}

func randomInt() int {
	return rand.Int()
}

func randomString() string {
	return uniuri.NewLen(16)
}

func randomBool() bool {
	return rand.Int()%2 == 0
}
