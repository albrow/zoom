package benchmark

import (
	"errors"
	"github.com/stephenalexbrowne/zoom"
	"github.com/stephenalexbrowne/zoom/redis"
	"log"
	"math/rand"
	"strconv"
	"time"
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
	zoom.Init(&zoom.Configuration{
		Address:  "/tmp/redis.sock",
		Network:  "unix",
		Database: 9,
	})

	conn := zoom.GetConn()
	defer conn.Close()

	n, err := redis.Int(conn.Do("DBSIZE"))
	if err != nil {
		return err
	}

	if n != 0 {
		return errors.New("Database #9 is not empty, test can not continue")
	}

	zoom.ClearCache()

	rand.Seed(time.Now().UTC().UnixNano())

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

// generate a random int from min (inclusively) to max
// (exclusively). I.e. to get either 1 or 0, use randInt(0,2)
func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}

// creates num persons, saves them. Returns a slice of their ids
func savePersons(num int) []string {
	// create a sequence of persons and save each
	// we'll do this concurrently so it's faster.
	ids := make([]string, num)
	idsChan := make(chan string, num)
	for i := 0; i < num; i++ {
		go func() {
			name := "person_" + strconv.Itoa(i)
			p := NewPerson(name, i)
			zoom.Save(p)
			idsChan <- p.Id
		}()
	}

	// wait for all the channels above to send,
	// indicating that they are done
	for i := 0; i < num; i++ {
		ids[i] = <-idsChan
	}

	return ids
}

// creates num persons but does not save them. Returns an slice of persons
func createPersons(num int) []*Person {
	// create a sequence of persons
	// we'll do this concurrently so it's faster.
	persons := make([]*Person, num)
	personsChan := make(chan *Person, num)
	for i := 0; i < num; i++ {
		go func() {
			name := "person_" + strconv.Itoa(i)
			p := NewPerson(name, i)
			personsChan <- p
		}()
	}

	// wait for all the channels above to send,
	// indicating that they are done
	for i := 0; i < num; i++ {
		persons[i] = <-personsChan
	}

	return persons
}
