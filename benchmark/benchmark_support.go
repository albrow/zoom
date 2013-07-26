package benchmark

import (
	"github.com/dchest/uniuri"
	"github.com/stephenalexbrowne/go-redis"
	"github.com/stephenalexbrowne/zoom"
	"strconv"
	"time"
)

// Person uses zoom for persistence
type Person struct {
	Name      string
	Age       int
	SiblingId string `refersTo:"person" as:"sibling"`
	*zoom.Model
}

var config = redis.Configuration{
	Database:   15,
	PoolSize:   99999,
	UseSockets: true,
	Address:    "/tmp/redis.sock",
}

var db *redis.Database = redis.Connect(config)

// A convenient constructor for our Person struct
func NewPerson(name string, age int) *Person {
	p := &Person{
		Name: name,
		Age:  age,
	}
	p.Model = zoom.NewModelFor(p)
	return p
}

// DirectPerson does not use zoom and interfaces directly with redis driver
type DirectPerson struct {
	Id        string
	Name      string
	Age       int
	SiblingId string
}

// A convenient constructor for our Person struct
func NewDirectPerson(name string, age int) *DirectPerson {
	p := &DirectPerson{
		Name: name,
		Age:  age,
	}
	return p
}

func (p *DirectPerson) Save() error {
	// main record
	id := generateRandomId()
	key := "person:" + id
	result := db.Command("hmset", key, "Name", p.Name, "Age", p.Age, "SiblingId", p.SiblingId)
	if result.Error() != nil {
		return result.Error()
	}
	p.Id = id
	return nil
}

func findDirectPersonById(id string) (*DirectPerson, error) {
	key := "person:" + id
	result := db.Command("hgetall", key)
	if result.Error() != nil {
		return nil, result.Error()
	}
	keyValues := result.KeyValues()
	p, err := convertKeyValuesToPerson(keyValues)
	if err != nil {
		return nil, err
	}
	p.Id = id
	return p, nil
}

func (p *DirectPerson) Delete() error {
	key := "person:" + p.Id
	result := db.Command("del", key)
	if result.Error() != nil {
		return result.Error()
	}
	return nil
}

func deleteDirectPersonById(id string) error {
	p, err := findDirectPersonById(id)
	if err != nil {
		return err
	}
	return p.Delete()
}

func (p *DirectPerson) FetchSibling() (*DirectPerson, error) {
	key := "person:" + p.SiblingId
	result := db.Command("hgetall", key)
	if result.Error() != nil {
		return nil, result.Error()
	}
	keyValues := result.KeyValues()
	p2, err := convertKeyValuesToPerson(keyValues)
	if err != nil {
		return nil, err
	}
	p2.Id = p.SiblingId
	return p2, nil
}

func convertKeyValuesToPerson(keyValues []*redis.KeyValue) (*DirectPerson, error) {
	p := &DirectPerson{}
	for _, keyValue := range keyValues {
		key := keyValue.Key
		value := keyValue.Value
		switch key {
		case "Name":
			p.Name = value.String()
		case "Age":
			ageInt, err := value.Int()
			if err != nil {
				return nil, err
			}
			p.Age = ageInt
		case "SiblingId":
			p.SiblingId = value.String()
		}
	}
	return p, nil
}

// Database helper functions
// setUp() and tearDown()
func setUp() {
	config := zoom.DbConfig{
		Database:   15,
		PoolSize:   99999,
		UseSockets: true,
		Address:    "/tmp/redis.sock",
	}
	zoom.InitDb(config)
	zoom.Register(&Person{}, "person")
}

func tearDown() {
	go zoom.CloseDb()
}

// generates a random string that is more or less
// garunteed to be unique. Used as Ids for records
// where an Id is not otherwise provided.
func generateRandomId() string {
	timeInt := time.Now().Unix()
	timeString := strconv.FormatInt(timeInt, 36)
	randomString := uniuri.NewLen(16)
	return randomString + timeString
}
