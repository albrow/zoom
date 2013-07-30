package test

// File contains support code for tests.
// e.g. type declarations, constructors,
// and other methods.
type Person struct {
	Id   string
	Name string
	Age  int
}

func (p *Person) SetId(id string) {
	p.Id = id
}

func (p *Person) GetId() string {
	return p.Id
}

// A convenient constructor for our Person struct
func NewPerson(name string, age int) *Person {
	p := &Person{
		Name: name,
		Age:  age,
	}
	return p
}

type AllTypes struct {
	Id      string
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
}

func (in *AllTypes) SetId(id string) {
	in.Id = id
}

func (in *AllTypes) GetId() string {
	return in.Id
}
