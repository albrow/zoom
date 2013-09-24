// file contains miscellaneous utility functions used throughout

package util

import (
	"fmt"
	"math/rand"
	"reflect"
)

func IndexOfStringSlice(a string, list []string) int {
	for i, b := range list {
		if b == a {
			return i
		}
	}
	return -1
}

func IndexOfSlice(a interface{}, list interface{}) int {
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

func StringSliceContains(a string, list []string) bool {
	return IndexOfStringSlice(a, list) != -1
}

func SliceContains(a interface{}, list interface{}) bool {
	return IndexOfSlice(a, list) != -1
}

func RemoveFromStringSlice(list []string, i int) []string {
	return append(list[:i], list[i+1:]...)
}

func RemoveElementFromStringSlice(list []string, elem string) []string {
	for i, e := range list {
		if e == elem {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

func CompareAsStringSet(expecteds, gots []string) (bool, string) {
	for _, got := range gots {
		index := IndexOfStringSlice(got, expecteds)
		if index == -1 {
			msg := fmt.Sprintf("Found unexpected element: %v", got)
			return false, msg
		}
		// remove from expecteds. makes sure we have one of each
		expecteds = RemoveFromStringSlice(expecteds, index)
	}
	// now expecteds should be empty. If it's not, there's a problem
	if len(expecteds) != 0 {
		msg := fmt.Sprintf("The following expected elements were not found: %v\n", expecteds)
		return false, msg
	}
	return true, "ok"
}

func CompareAsSet(expecteds, gots interface{}) (bool, string) {
	eVal := reflect.ValueOf(expecteds)
	gVal := reflect.ValueOf(gots)

	// make sure everything in expecteds is also in gots
	for i := 0; i < eVal.Len(); i++ {
		expected := eVal.Index(i).Interface()
		index := IndexOfSlice(expected, gots)
		if index == -1 {
			msg := fmt.Sprintf("Missing expected element: %v", expected)
			return false, msg
		}
	}

	// make sure everything in gots is also in expecteds
	for i := 0; i < gVal.Len(); i++ {
		got := gVal.Index(i).Interface()
		index := IndexOfSlice(got, expecteds)
		if index == -1 {
			msg := fmt.Sprintf("Found unexpected element: %v", got)
			return false, msg
		}
	}
	return true, "ok"
}

func TypeIsSliceOrArray(typ reflect.Type) bool {
	k := typ.Kind()
	return (k == reflect.Slice || k == reflect.Array) && typ.Elem().Kind() != reflect.Uint8
}

func TypeIsPointerToStruct(typ reflect.Type) bool {
	return typ.Kind() == reflect.Ptr && typ.Elem().Kind() == reflect.Struct
}

func TypeIsString(typ reflect.Type) bool {
	if typ.Kind() == reflect.Ptr {
		return elemIsString(typ.Elem())
	} else {
		return elemIsString(typ)
	}
}

func elemIsString(typ reflect.Type) bool {
	k := typ.Kind()
	return k == reflect.String || ((k == reflect.Slice || k == reflect.Array) && typ.Elem().Kind() == reflect.Uint8)
}

func TypeIsNumeric(typ reflect.Type) bool {
	if typ.Kind() == reflect.Ptr {
		return elemIsNumeric(typ.Elem())
	} else {
		return elemIsNumeric(typ)
	}
}

func elemIsNumeric(typ reflect.Type) bool {
	k := typ.Kind()
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func TypeIsBool(typ reflect.Type) bool {
	k := typ.Kind()
	return k == reflect.Bool || (k == reflect.Ptr && typ.Elem().Kind() == reflect.Bool)
}

// generate a random int from min to max (inclusively).
// I.e. to get either 1 or 0, use randInt(0,1)
func RandInt(min int, max int) int {
	if !(max-min >= 1) {
		panic("invalid args. max must be at least one more than min")
	}
	return min + rand.Intn(max-min+1)
}
