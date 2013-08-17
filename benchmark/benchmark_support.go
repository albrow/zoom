package benchmark

import (
	"errors"
	"github.com/stephenalexbrowne/zoom"
	"github.com/stephenalexbrowne/zoom/redis"
	"log"
)

// The Person struct
type Person struct {
	Name string
	Age  int
	*zoom.Model
}

// A convenient constructor for the Person struct
func NewPerson(name string, age int) *Person {
	p := &Person{
		Name:  name,
		Age:   age,
		Model: new(zoom.Model),
	}
	return p
}

// The Invoice struct
type Invoice struct {
	Created  int64
	Updated  int64
	Memo     string
	PersonId int64
	IsPaid   bool
	*zoom.Model
}

// A convenient constructor for the Invoice struct
func NewInvoice(created, updated int64, memo string, personId int64, isPaid bool) *Invoice {
	return &Invoice{
		Created:  created,
		Updated:  updated,
		Memo:     memo,
		PersonId: personId,
		IsPaid:   isPaid,
		Model:    new(zoom.Model),
	}
}

// Database helper functions
// setUp() and tearDown()
func setUp() error {
	zoom.Init(&zoom.Configuration{Database: 9})

	conn := zoom.GetConn()
	defer conn.Close()

	n, err := redis.Int(conn.Do("DBSIZE"))
	if err != nil {
		return err
	}

	if n != 0 {
		return errors.New("Database #9 is not empty, test can not continue")
	}

	zoom.Register(&Person{}, "person")
	zoom.Register(&Invoice{}, "invoice")
	return nil
}

func tearDown() {
	conn := zoom.GetConn()
	_, err := conn.Do("flushdb")
	if err != nil {
		log.Fatal(err)
	}
	conn.Close()
	zoom.Close()
}
