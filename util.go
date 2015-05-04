// Copyright 2015 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File util.go contains miscellaneous utility functions used throughout
// the zoom library.

package zoom

import (
	"fmt"
	"github.com/dchest/uniuri"
	"math/cmplx"
	"math/rand"
	"reflect"
	"strconv"
	"time"
)

var (
	// delString is used as a suffix for string index tricks. This is a string which equals the ASCII
	// DEL character and is the highest possible value (in terms of codepoint, which is also
	// how redis sorts strings) for an ASCII character.
	delString = string([]byte{byte(127)})
	// nullString is used as a suffix for string index tricks. This is a string which equals the ASCII
	// NULL character and is the lowest possible value (in terms of codepoint, which is also
	// how redis sorts strings) for an ASCII character.
	nullString = string([]byte{byte(0)})
)

// Models converts in to []Model. It will panic if the underlying type
// of in is not a slice of some concrete type which implements Model.
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
	if !typeIsRegistered(elemTyp) {
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

// Interfaces converts in to []interface{}. It will panic if the underlying type
// of in is not a slice.
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

// indexOfStringSlice returns the index of s in strings, or
// -1 if a is not found in strings
func indexOfStringSlice(strings []string, s string) int {
	for i, b := range strings {
		if b == s {
			return i
		}
	}
	return -1
}

// stringSliceContains returns true iff strings contains s
func stringSliceContains(strings []string, s string) bool {
	return indexOfStringSlice(strings, s) != -1
}

// removeElementFromStringSlice removes elem from list and returns
// the new slice.
func removeElementFromStringSlice(list []string, elem string) []string {
	for i, e := range list {
		if e == elem {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

// compareAsStringSet compares expecteds and gots as if they were sets, i.e.,
// it checks if they contain the same values, regardless of order. It returns true
// and an empty string if expecteds and gots contain all the same values and false
// and a detailed message if they do not.
func compareAsStringSet(expecteds, gots []string) (bool, string) {
	// make sure everything in expecteds is also in gots
	for _, e := range expecteds {
		index := indexOfStringSlice(gots, e)
		if index == -1 {
			msg := fmt.Sprintf("Missing expected element: %v", e)
			return false, msg
		}
	}

	// make sure everything in gots is also in expecteds
	for _, g := range gots {
		index := indexOfStringSlice(expecteds, g)
		if index == -1 {
			msg := fmt.Sprintf("Found extra element: %v", g)
			return false, msg
		}
	}

	return true, "ok"
}

// typeIsSliceOrArray returns true iff typ is a slice or array
func typeIsSliceOrArray(typ reflect.Type) bool {
	k := typ.Kind()
	return (k == reflect.Slice || k == reflect.Array) && typ.Elem().Kind() != reflect.Uint8
}

// typeIsPointerToStruct returns true iff typ is a pointer to a struct
func typeIsPointerToStruct(typ reflect.Type) bool {
	return typ.Kind() == reflect.Ptr && typ.Elem().Kind() == reflect.Struct
}

// typeIsString returns true iff typ is a string or an array or slice of bytes
// (which is freely castable to a string)
func typeIsString(typ reflect.Type) bool {
	k := typ.Kind()
	return k == reflect.String || ((k == reflect.Slice || k == reflect.Array) && typ.Elem().Kind() == reflect.Uint8)
}

// typeIsNumeric returns true iff typ is one of the numeric primative types
func typeIsNumeric(typ reflect.Type) bool {
	k := typ.Kind()
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// typeIsBool returns true iff typ is a bool
func typeIsBool(typ reflect.Type) bool {
	k := typ.Kind()
	return k == reflect.Bool
}

// typeIsPrimative returns true iff typ is a primative type, i.e. either a
// string, bool, or numeric type.
func typeIsPrimative(typ reflect.Type) bool {
	return typeIsString(typ) || typeIsNumeric(typ) || typeIsBool(typ)
}

// numericScore returns a float64 which is the score for val in a sorted set.
// If val is a pointer, it will keep dereferencing until it reaches the underlying
// value. It panics if val is not a numeric type or a pointer to a numeric type.
func numericScore(val reflect.Value) float64 {
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		integer := val.Int()
		return float64(integer)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uinteger := val.Uint()
		return float64(uinteger)
	case reflect.Float32, reflect.Float64:
		return val.Float()
	default:
		msg := fmt.Sprintf("zoom: attempt to call numericScore on non-numeric type %s", val.Type().String())
		panic(msg)
	}
}

// boolScore returns an int which is the score for val in a sorted set.
// If val is a pointer, it will keep dereferencing until it reaches the underlying
// value. It panics if val is not a boolean or a pointer to a boolean.
func boolScore(val reflect.Value) int {
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Bool {
		msg := fmt.Sprintf("zoom: attempt to call boolScore on non-boolean type %s", val.Type().String())
		panic(msg)
	}
	return convertBoolToInt(val.Bool())
}

// convertBoolToInt converts a bool to an int using the following rule:
// false = 0
// true = 1
func convertBoolToInt(b bool) int {
	if b {
		return 1
	} else {
		return 0
	}
}

// modelIds returns the ids for models
func modelIds(models []Model) []string {
	results := make([]string, len(models))
	for i, m := range models {
		results[i] = m.Id()
	}
	return results
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

// randomInt returns a psuedo-random int between the minimum and maximum
// possible values.
func randomInt() int {
	return rand.Int()
}

// randomString returns a random string of length 16
func randomString() string {
	return uniuri.NewLen(16)
}

// randomBool returns a random bool
func randomBool() bool {
	return rand.Int()%2 == 0
}

// randomFloat returns a random float64
func randomFloat() float64 {
	return rand.Float64()
}

// randomComplex returns a random complex128
func randomComplex() complex128 {
	return cmplx.Rect(randomFloat(), randomFloat())
}

// decrementString subtracts 1 to the last codepoint in s and returns the new string
// E.g. if the input string is "abc" the return will be "abb" because the codepoint
// for 'c' is 99, 99-1 = 98, and the codepoint 98 corresponds to 'b'.
func decrementString(s string) string {
	codepoints := []uint8(s)
	codepoints[len(codepoints)-1] = codepoints[len(codepoints)-1] + 1
	return string(codepoints)
}

// incrementString adds 1 to the last codepoint in s and returns the new string
// E.g. if the input string is "abc" the return will be "abd" because the codepoint
// for 'c' is 99, 99+1 = 100, and the codepoint 100 corresponds to 'd'.
func incrementString(s string) string {
	codepoints := []uint8(s)
	codepoints[len(codepoints)-1] = codepoints[len(codepoints)-1] + 1
	return string(codepoints)
}
