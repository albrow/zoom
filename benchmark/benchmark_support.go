package benchmark

import (
	"errors"
	"github.com/stephenalexbrowne/zoom"
	"github.com/stephenalexbrowne/zoom/redis"
	"log"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

const CHANNEL_BUFFER_SIZE = 10000
const MAX_GOROUTINES = 10000

var routinesChan chan int = make(chan int, MAX_GOROUTINES)

// The Person struct
type Person struct {
	Name string
	Age  int
	*zoom.Model
}

// A convenient constructor for the Person struct
func NewPerson(name string, age int) *Person {
	return &Person{
		Name:  name,
		Age:   age,
		Model: new(zoom.Model),
	}
}

// The Parent struct, used for testing many-to-many performance
type Parent struct {
	Name     string
	Children []*Person
	*zoom.Model
}

// A convenient constructor for the Parent struct
func NewParent(name string) *Parent {
	return &Parent{
		Name:     name,
		Children: make([]*Person, 0),
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
	zoom.Register(&Parent{}, "parent")
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
	for i := 0; i < num; i++ {
		name := "person_" + strconv.Itoa(i)
		p := NewPerson(name, i)
		zoom.Save(p)
		ids[i] = p.Id
	}

	return ids
}

// creates num persons but does not save them. Returns an slice of persons
func createPersons(num int) []*Person {
	// create a sequence of persons
	// we'll do this concurrently so it's faster.
	persons := make([]*Person, num)
	for i := 0; i < num; i++ {
		name := "person_" + strconv.Itoa(i)
		p := NewPerson(name, i)
		persons[i] = p
	}

	return persons
}

// creates and saves numParents parents with numChildren children each.
// Returns a slice of parent ids
func saveParentsWithChildren(numParents, numChildren int) []string {
	// create a sequence of parents and save each
	// we'll do this concurrently so it's faster.
	ids := make([]string, numParents)
	for i := 0; i < numParents; i++ {
		name := "parent_" + strconv.Itoa(i)
		p := NewParent(name)
		p.Children = append(p.Children, createPersons(numChildren)...)
		zoom.Save(p)
		ids[i] = p.Id
	}

	return ids
}

func benchmarkSetTransaction(b *testing.B, num int) {
	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	conn := zoom.GetConn()

	if err := conn.Send("MULTI"); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		key := "foo_" + strconv.Itoa(i)
		val := "bar_" + strconv.Itoa(i)
		b.StartTimer()
		err := conn.Send("SET", key, val)
		b.StopTimer()
		if err != nil {
			b.Fatal(err)
		}
		if i%num == 0 {
			b.StartTimer()
			// call exec and start a new transaction
			_, err := conn.Do("EXEC")
			b.StopTimer()
			if err != nil {
				b.Fatal(err)
			}
			b.StartTimer()
			conn.Send("MULTI")
		}
	}

	// make sure the pipeline is flushed before the next test
	conn.Do("EXEC")
	conn.Close()
}

func benchmarkGetTransaction(b *testing.B, num int) {
	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	tConn := zoom.GetConn() // used for transactional GETs
	mConn := zoom.GetConn() // used for non-transactional SETs in between

	if err := tConn.Send("MULTI"); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		key := "foo_" + strconv.Itoa(i)
		val := "bar_" + strconv.Itoa(i)
		_, err := mConn.Do("SET", key, val)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		err = tConn.Send("GET", key)
		b.StopTimer()
		if err != nil {
			b.Fatal(err)
		}
		if i%num == 0 {
			b.StartTimer()
			// call exec and start a new transaction
			_, err := tConn.Do("EXEC")
			b.StopTimer()
			if err != nil {
				b.Fatal(err)
			}
			b.StartTimer()
			tConn.Send("MULTI")
		}
	}

	// make sure the pipeline is flushed before the next test
	tConn.Do("EXEC")
	tConn.Close()
}

func benchmarkFindAll(b *testing.B, num int) {
	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	savePersons(num)

	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		zoom.FindAll("person")
	}
}

func benchmarkFindAllNoCache(b *testing.B, num int) {
	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	savePersons(num)

	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		zoom.FindAll("person")
		b.StopTimer()
		zoom.ClearCache()
		b.StartTimer()
	}
}

func benchmarkRepeatSaveOneToMany(b *testing.B, numChildren int) {
	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	p := NewParent("parent_0")
	p.Children = append(p.Children, createPersons(numChildren)...)

	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		err := zoom.Save(p)
		b.StopTimer()
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
	}
}

func benchmarkRepeatFindOneToMany(b *testing.B, numChildren int) {
	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	ids := saveParentsWithChildren(1, numChildren)

	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		id := ids[0]
		b.StartTimer()
		_, err := zoom.FindById("parent", id)
		b.StopTimer()
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
	}
}

func benchmarkSequentialFindOneToMany(b *testing.B, numChildren int) {

	b.StopTimer()

	const NUM_PARENTS = 100

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	ids := saveParentsWithChildren(NUM_PARENTS, numChildren)

	b.StartTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		index := i % NUM_PARENTS
		id := ids[index]
		b.StartTimer()
		_, err := zoom.FindById("parent", id)
		b.StopTimer()
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
	}
}

func benchmarkRandomFindOneToMany(b *testing.B, numChildren int) {

	b.StopTimer()

	const NUM_PARENTS = 100

	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	ids := saveParentsWithChildren(NUM_PARENTS, numChildren)

	b.StartTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		index := randInt(0, NUM_PARENTS)
		id := ids[index]
		b.StartTimer()
		_, err := zoom.FindById("parent", id)
		b.StopTimer()
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
	}
}
