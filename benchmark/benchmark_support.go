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
	zoom.DefaultData
}

// The Parent struct, used for testing many-to-many performance
type Parent struct {
	Name     string
	Children []*Person
	zoom.DefaultData
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
		p := &Person{Name: name, Age: i}
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
		p := &Person{Name: name, Age: i}
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
		p := &Parent{Name: name}
		p.Children = append(p.Children, createPersons(numChildren)...)
		zoom.Save(p)
		ids[i] = p.Id
	}

	return ids
}

func benchmarkCommand(b *testing.B, setup func(), checkReply func(interface{}, error), cmd string, args ...interface{}) {
	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	if setup != nil {
		setup()
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		conn := zoom.GetConn()
		reply, err := conn.Do(cmd, args...)
		b.StopTimer()
		if checkReply != nil {
			checkReply(reply, err)
		}
		b.StartTimer()
		conn.Close()
	}
}

func benchmarkSave(b *testing.B, num int, personSelect func(int, []*Person) *Person) {
	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	// create a sequence of persons to be saved
	persons := createPersons(num)

	// reset the timer
	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		p := personSelect(i, persons)
		b.StartTimer()
		err := zoom.Save(p)
		b.StopTimer()
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
	}
}

func benchmarkFindById(b *testing.B, num int, idSelect func(int, []string) string) {
	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	ids := savePersons(num)

	// reset the timer
	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		id := idSelect(i, ids)
		b.StartTimer()
		zoom.FindById("person", id)
	}
}

func benchmarkDeleteById(b *testing.B, num int, idSelect func(int, []string) string) {
	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	ids := savePersons(num)

	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		id := idSelect(i, ids)
		b.StartTimer()
		zoom.DeleteById("person", id)
	}
}

func benchmarkFindAll(b *testing.B, num int, onEach func()) {
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
		if onEach != nil {
			onEach()
		}
		b.StartTimer()
	}
}

func benchmarkFindAllWithCache(b *testing.B, num int) {
	benchmarkFindAll(b, num, nil)
}

func benchmarkFindAllNoCache(b *testing.B, num int) {
	benchmarkFindAll(b, num, func() { zoom.ClearCache() })
}

func benchmarkRepeatSaveOneToMany(b *testing.B, numChildren int) {
	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	p := &Parent{Name: "parent_0"}
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

func benchmarkFindOneToMany(b *testing.B, numParents, numChildren int, selectId func(int, []string) string) {
	err := setUp()
	if err != nil {
		b.Fatal(err)
	} else {
		defer tearDown()
	}

	ids := saveParentsWithChildren(numParents, numChildren)

	b.ResetTimer()

	// run the actual test
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		id := selectId(i, ids)
		b.StartTimer()
		_, err := zoom.FindById("parent", id)
		b.StopTimer()
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
	}
}

func singleIdSelect(i int, ids []string) string {
	return ids[0]
}

func sequentialIdSelect(i int, ids []string) string {
	return ids[i%len(ids)]
}

func randomIdSelect(i int, ids []string) string {
	return ids[randInt(0, len(ids))]
}

func singleIdSelectNoCache(i int, ids []string) string {
	zoom.ClearCache()
	return ids[0]
}

func sequentialIdSelectNoCache(i int, ids []string) string {
	zoom.ClearCache()
	return ids[i%len(ids)]
}

func randomIdSelectNoCache(i int, ids []string) string {
	zoom.ClearCache()
	return ids[randInt(0, len(ids))]
}
