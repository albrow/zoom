// file contains miscellaneous utility functions used throughout

package util

import (
	"fmt"
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

func RemoveFromStringSlice(list []string, i int) []string {
	return append(list[:i], list[i+1:]...)
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

func TypeIsSliceOrArray(typ reflect.Type) bool {
	return (typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array) && typ.Elem().Kind() != reflect.Uint8
}
