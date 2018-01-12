// Copyright 2015 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File util.go contains miscellaneous utility functions used throughout
// the zoom library.

package zoom

import (
	"fmt"
	"hash/crc32"
	"math/big"
	"net"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/dchest/uniuri"
	"github.com/tv42/base58"
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
	// hardwareID is a unique id for the current machine. Right now it uses the crc32 checksum of the MAC address.
	hardwareID = ""
)

func init() {
	// Set chars to the 58 non-ambiguous characters use by base58 encoding
	uniuri.StdChars = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")
}

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

// typeIsNumeric returns true iff typ is one of the numeric primitive types
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

// typeIsPrimative returns true iff typ is a primitive type, i.e. either a
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
	}
	return 0
}

// modelIDs returns the ids for models
func modelIDs(models []Model) []string {
	results := make([]string, len(models))
	for i, m := range models {
		results[i] = m.ModelID()
	}
	return results
}

// generateRandomID generates a pseudo-random string that is highly likely to be unique.
// The string is base58 encoded and consists of 4 components:
//   1. The current UTC unix time with second precision
//   2. An atomic counter which is always 4 characters long and cycles
//      through the range of 0 to 11,316,495
//   3. A unique hardware identifier based on the MAC address of the
//      current machine
//   4. A pseudo-randomly generated sequence of 6 characters
func generateRandomID() string {
	return getTimeString() + getAtomicCounter() + getHardwareID() + uniuri.NewLen(6)
}

// getTimeString returns the current UTC unix time with second precision encoded
// with base58 encoding.
func getTimeString() string {
	timeInt := time.Now().UTC().Unix()
	timeBytes := base58.EncodeBig(nil, big.NewInt(timeInt))
	return string(timeBytes)
}

// getHardwareID returns a unique identifier for the current machine. It does this
// by iterating through the network interfaces of the machine and picking the first
// one that has a non-empty hardware (MAC) address. Then it takes the crc32 checksum
// of the MAC address and encodes it in base58 encoding. getHardwareID caches results,
// so subsequent calls will return the previously calculated result. If no MAC address
// could be found, the function will use "0" as the MAC address. This is not ideal, but
// generateRandomID uses other means to try and avoid collisions.
func getHardwareID() string {
	if hardwareID != "" {
		return hardwareID
	}
	address := ""
	inters, err := net.Interfaces()
	if err == nil {
		for _, inter := range inters {
			if inter.HardwareAddr.String() != "" {
				address = inter.HardwareAddr.String()
				break
			}
		}
	}
	if address == "" {
		address = "0"
	}
	check32 := crc32.ChecksumIEEE([]byte(address))
	id58 := base58.EncodeBig(nil, big.NewInt(int64(check32)))
	hardwareID = string(id58)
	return hardwareID
}

var counter int32

// getAtomicCounter returns the base58 encoding of a counter which cycles through
// the values in the range 0 to 11,316,495. This is the range that can be represented
// with 4 base58 characters. The returned result will be padded with zeros such that
// it is always 4 characters long.
func getAtomicCounter() string {
	atomic.AddInt32(&counter, 1)
	if counter > 58*58*58*58-1 {
		// Reset the counter if we're beyond what we
		// can represent with 4 base58 characters
		atomic.StoreInt32(&counter, 0)
	}
	counterBytes := base58.EncodeBig(nil, big.NewInt(int64(counter)))
	counterStr := string(counterBytes)
	switch len(counterStr) {
	case 0:
		return "0000"
	case 1:
		return "000" + counterStr
	case 2:
		return "00" + counterStr
	case 3:
		return "0" + counterStr
	default:
		return counterStr[0:4]
	}
}
