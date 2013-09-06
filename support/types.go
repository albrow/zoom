// file contains type declarations that are used in various places, especially in test package

package support

import (
	"github.com/stephenalexbrowne/zoom"
)

// The Person struct
type Person struct {
	Name string
	Age  int
	zoom.DefaultData
}

// The AllTypes struct
// A struct containing all supported types
type AllTypes struct {
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
	zoom.DefaultData
}

type ModelWithList struct {
	List []string `redisType:"list"`
	zoom.DefaultData
}

type ModelWithSet struct {
	Set []string `redisType:"set"`
	zoom.DefaultData
}
