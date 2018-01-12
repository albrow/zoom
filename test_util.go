// Copyright 2015 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File test_util.go contains helper functions for writing tests.

package zoom

import (
	"flag"
	"fmt"
	"math/cmplx"
	"math/rand"
	"reflect"
	"sort"
	"sync"
	"testing"

	"github.com/dchest/uniuri"
	"github.com/garyburd/redigo/redis"
)

var (
	address  = flag.String("address", "localhost:6379", "the address of a redis server to connect to")
	network  = flag.String("network", "tcp", "the network to use for the database connection (e.g. 'tcp' or 'unix')")
	database = flag.Int("database", 9, "the redis database number to use for testing")
	testPool *Pool
)

// setUpOnce is used to enforce that the setup process happens exactly once,
// no matter how many times testingSetUp is called
var setUpOnce = sync.Once{}

// testingSetUp prepares the database for testing and registers the testing types.
// The setup-related code only runs once, no matter how many times you call
// testingSetUp
func testingSetUp() {
	setUpOnce.Do(func() {
		options := DefaultPoolOptions
		if address != nil {
			options = options.WithAddress(*address)
		}
		if network != nil {
			options = options.WithNetwork(*network)
		}
		if database != nil {
			options = options.WithDatabase(*database)
		}
		testPool = NewPoolWithOptions(options)
		checkDatabaseEmpty()
		registerTestingTypes()
	})
}

// testModel is a model type that is used for testing
type testModel struct {
	Int    int
	String string
	Bool   bool
	RandomID
}

// createTestModels creates and returns n testModels with
// random field values, but does not save them to the database.
func createTestModels(n int) []*testModel {
	models := make([]*testModel, n)
	for i := 0; i < n; i++ {
		models[i] = &testModel{
			Int:    randomInt(),
			String: randomString(),
			Bool:   randomBool(),
		}
	}
	return models
}

// createAndSaveTestModels creates n testModels with random field
// values, saves them, and returns them.
func createAndSaveTestModels(n int) ([]*testModel, error) {
	models := createTestModels(n)
	t := testPool.NewTransaction()
	for _, model := range models {
		t.Save(testModels, model)
	}
	if err := t.Exec(); err != nil {
		return nil, err
	}
	return models, nil
}

// indexedTestModel is a model type used for testing indexes
// and queries.
type indexedTestModel struct {
	Int    int    `zoom:"index"`
	String string `zoom:"index"`
	Bool   bool   `zoom:"index"`
	RandomID
}

func (m *indexedTestModel) GoString() string {
	if m == nil {
		return fmt.Sprintf("(%T) nil", m)
	}
	return fmt.Sprintf("%v", *m)
}

// createIndexedTestModels creates and returns n testModels with
// random field values, but does not save them to the database.
func createIndexedTestModels(n int) []*indexedTestModel {
	models := make([]*indexedTestModel, n)
	for i := 0; i < n; i++ {
		models[i] = &indexedTestModel{
			Int:    randomInt(),
			String: randomString(),
			Bool:   randomBool(),
		}
	}
	return models
}

// createAndSaveIndexedTestModels creates n indexedTestModels with
// random field values, saves them, and returns them.
func createAndSaveIndexedTestModels(n int) ([]*indexedTestModel, error) {
	models := createIndexedTestModels(n)
	t := testPool.NewTransaction()
	for _, model := range models {
		t.Save(indexedTestModels, model)
	}
	if err := t.Exec(); err != nil {
		return nil, err
	}
	return models, nil
}

type indexedPrimativesModel struct {
	Uint    uint    `zoom:"index"`
	Uint8   uint8   `zoom:"index"`
	Uint16  uint16  `zoom:"index"`
	Uint32  uint32  `zoom:"index"`
	Uint64  uint64  `zoom:"index"`
	Int     int     `zoom:"index"`
	Int8    int8    `zoom:"index"`
	Int16   int16   `zoom:"index"`
	Int32   int32   `zoom:"index"`
	Int64   int64   `zoom:"index"`
	Float32 float32 `zoom:"index"`
	Float64 float64 `zoom:"index"`
	Byte    byte    `zoom:"index"`
	Rune    rune    `zoom:"index"`
	String  string  `zoom:"index"`
	Bool    bool    `zoom:"index"`
	RandomID
}

// createIndexedPrimativesModel instantiates and returns an indexedPrimativesModel with
// random values for all fields.
func createIndexedPrimativesModel() *indexedPrimativesModel {
	return &indexedPrimativesModel{
		Uint:    uint(randomInt()),
		Uint8:   uint8(randomInt()),
		Uint16:  uint16(randomInt()),
		Uint32:  uint32(randomInt()),
		Uint64:  uint64(randomInt()),
		Int:     randomInt(),
		Int8:    int8(randomInt()),
		Int16:   int16(randomInt()),
		Int32:   int32(randomInt()),
		Int64:   int64(randomInt()),
		Float32: float32(randomInt()),
		Float64: float64(randomInt()),
		Byte:    []byte(randomString())[0],
		Rune:    []rune(randomString())[0],
		String:  randomString(),
		Bool:    randomBool(),
	}
}

type indexedPointersModel struct {
	Uint    *uint    `zoom:"index"`
	Uint8   *uint8   `zoom:"index"`
	Uint16  *uint16  `zoom:"index"`
	Uint32  *uint32  `zoom:"index"`
	Uint64  *uint64  `zoom:"index"`
	Int     *int     `zoom:"index"`
	Int8    *int8    `zoom:"index"`
	Int16   *int16   `zoom:"index"`
	Int32   *int32   `zoom:"index"`
	Int64   *int64   `zoom:"index"`
	Float32 *float32 `zoom:"index"`
	Float64 *float64 `zoom:"index"`
	Byte    *byte    `zoom:"index"`
	Rune    *rune    `zoom:"index"`
	String  *string  `zoom:"index"`
	Bool    *bool    `zoom:"index"`
	RandomID
}

// createIndexedPointersModel instantiates and returns an indexedPointersModel with
// random values for all fields.
func createIndexedPointersModel() *indexedPointersModel {
	Uint := uint(randomInt())
	Uint8 := uint8(randomInt())
	Uint16 := uint16(randomInt())
	Uint32 := uint32(randomInt())
	Uint64 := uint64(randomInt())
	Int := randomInt()
	Int8 := int8(randomInt())
	Int16 := int16(randomInt())
	Int32 := int32(randomInt())
	Int64 := int64(randomInt())
	Float32 := float32(randomInt())
	Float64 := float64(randomInt())
	Byte := []byte(randomString())[0]
	Rune := []rune(randomString())[0]
	String := randomString()
	Bool := randomBool()
	return &indexedPointersModel{
		Uint:    &Uint,
		Uint8:   &Uint8,
		Uint16:  &Uint16,
		Uint32:  &Uint32,
		Uint64:  &Uint64,
		Int:     &Int,
		Int8:    &Int8,
		Int16:   &Int16,
		Int32:   &Int32,
		Int64:   &Int64,
		Float32: &Float32,
		Float64: &Float64,
		Byte:    &Byte,
		Rune:    &Rune,
		String:  &String,
		Bool:    &Bool,
	}
}

var (
	testModels              *Collection
	indexedTestModels       *Collection
	indexedPrimativesModels *Collection
	indexedPointersModels   *Collection
)

// registerTestingTypes registers the common types used for testing
func registerTestingTypes() {
	testModelTypes := []struct {
		collection **Collection
		model      Model
		index      bool
	}{
		{
			collection: &testModels,
			model:      &testModel{},
			index:      true,
		},
		{
			collection: &indexedTestModels,
			model:      &indexedTestModel{},
			index:      true,
		},
		{
			collection: &indexedPrimativesModels,
			model:      &indexedPrimativesModel{},
			index:      true,
		},
		{
			collection: &indexedPointersModels,
			model:      &indexedPointersModel{},
			index:      true,
		},
	}
	for _, m := range testModelTypes {
		options := DefaultCollectionOptions.WithIndex(true)
		collection, err := testPool.NewCollectionWithOptions(m.model, options)
		if err != nil {
			panic(err)
		}
		*m.collection = collection
	}
}

// checkDatabaseEmpty panics if the database to be used for testing
// is not empty.
func checkDatabaseEmpty() {
	conn := testPool.NewConn()
	defer func() {
		_ = conn.Close()
	}()
	n, err := redis.Int(conn.Do("DBSIZE"))
	if err != nil {
		panic(err.Error())
	}
	if n != 0 {
		err := fmt.Errorf("database #%d is not empty: testing can not continue", *database)
		panic(err)
	}
}

// testingTearDown flushes the database. It should be run at the end
// of each test that touches the database, typically by using defer.
func testingTearDown() {
	// flush and close the database
	conn := testPool.NewConn()
	defer func() {
		_ = conn.Close()
	}()
	if _, err := conn.Do("flushdb"); err != nil {
		panic(err)
	}
}

// expectSetContains sets an error via t.Errorf if member is not in the set
func expectSetContains(t *testing.T, setName string, member interface{}) {
	conn := testPool.NewConn()
	defer func() {
		_ = conn.Close()
	}()
	contains, err := redis.Bool(conn.Do("SISMEMBER", setName, member))
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	if !contains {
		t.Errorf("Expected set %s to contain %s but it did not.", setName, member)
	}
}

// expectSetDoesNotContain sets an error via t.Errorf if member is in the set
func expectSetDoesNotContain(t *testing.T, setName string, member interface{}) {
	conn := testPool.NewConn()
	defer func() {
		_ = conn.Close()
	}()
	contains, err := redis.Bool(conn.Do("SISMEMBER", setName, member))
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	if contains {
		t.Errorf("Expected set %s to not contain %s but it did.", setName, member)
	}
}

// expectFieldEquals sets an error via t.Errorf if the the field identified by fieldName does
// not equal expected according to the database.
func expectFieldEquals(t *testing.T, key string, fieldName string, marshalerUnmarshaler MarshalerUnmarshaler, expected interface{}) {
	conn := testPool.NewConn()
	defer func() {
		_ = conn.Close()
	}()
	reply, err := conn.Do("HGET", key, fieldName)
	if err != nil {
		t.Errorf("Unexpected error in HGET: %s", err.Error())
	}
	if reply == nil {
		if expected == nil {
			return
		}
		t.Errorf("Field %s was nil. Expected: %v", fieldName, expected)
	}
	srcBytes, ok := reply.([]byte)
	if !ok {
		t.Fatalf("Unexpected error: could not convert %v of type %T to []byte.\n", reply, reply)
	}
	typ := reflect.TypeOf(expected)
	dest := reflect.New(typ).Elem()
	switch {
	case typeIsPrimative(typ):
		err = scanPrimitiveVal(srcBytes, dest)
	case typ.Kind() == reflect.Ptr:
		err = scanPointerVal(srcBytes, dest)
	default:
		err = scanInconvertibleVal(marshalerUnmarshaler, srcBytes, dest)
	}
	if err != nil {
		t.Errorf("Unexpected error scanning value for field %s: %s", fieldName, err)
	}
	got := dest.Interface()
	if !reflect.DeepEqual(expected, got) {
		t.Errorf("Field %s for %s was not saved correctly.\n\tExpected: %v\n\tBut got:  %v", fieldName, key, expected, got)
	}
}

// expectKeyExists sets an error via t.Errorf if key does not exist in the database.
func expectKeyExists(t *testing.T, key string) {
	conn := testPool.NewConn()
	defer func() {
		_ = conn.Close()
	}()
	if exists, err := redis.Bool(conn.Do("EXISTS", key)); err != nil {
		t.Errorf("Unexpected error in EXISTS: %s", err.Error())
	} else if !exists {
		t.Errorf("Expected key %s to exist, but it did not.", key)
	}
}

// expectKeyDoesNotExist sets an error via t.Errorf if key does exist in the database.
func expectKeyDoesNotExist(t *testing.T, key string) {
	conn := testPool.NewConn()
	defer func() {
		_ = conn.Close()
	}()
	if exists, err := redis.Bool(conn.Do("EXISTS", key)); err != nil {
		t.Errorf("Unexpected error in EXISTS: %s", err.Error())
	} else if exists {
		t.Errorf("Expected key %s to not exist, but it did exist.", key)
	}
}

// expectModelExists sets an error via t.Errorf if model does not exist in
// the database. It checks for the main hash as well as the id in the index of all
// ids for a given type.
func expectModelExists(t *testing.T, mt *Collection, model Model) {
	modelKey := mt.ModelKey(model.ModelID())
	expectKeyExists(t, modelKey)
	expectSetContains(t, mt.IndexKey(), model.ModelID())
}

// expectModelDoesNotExist sets an error via t.Errorf if model exists in the database.
// It checks for the main hash as well as the id in the index of all ids for a
// given type.
func expectModelDoesNotExist(t *testing.T, mt *Collection, model Model) {
	modelKey := mt.ModelKey(model.ModelID())
	expectKeyDoesNotExist(t, modelKey)
	expectSetDoesNotContain(t, mt.IndexKey(), model.ModelID())
}

// expectModelsExist sets an error via t.Errorf for each model in models that
// does not exist in the database. It checks for the main hash as well as the id in
// the index of all ids for a given type.
func expectModelsExist(t *testing.T, mt *Collection, models []Model) {
	for _, model := range models {
		modelKey := mt.ModelKey(model.ModelID())
		expectKeyExists(t, modelKey)
		expectSetContains(t, mt.IndexKey(), model.ModelID())
	}
}

// expectModelsDoNotExist sets an error via t.Errorf for each model in models that
// exists in the database. It checks for the main hash as well as the id in the index
// of all ids for a given type.
func expectModelsDoNotExist(t *testing.T, mt *Collection, models []Model) {
	for _, model := range models {
		modelKey := mt.ModelKey(model.ModelID())
		expectKeyDoesNotExist(t, modelKey)
		expectSetDoesNotContain(t, mt.IndexKey(), model.ModelID())
	}
}

// indexExists returns true iff an index for the given type and field exists in the database.
// It returns an error if collection does not have a field called fieldName, the field identified
// by fieldName is not an indexed field, there was a problem connecting to the database, or
// there was some other unexpected problem.
func indexExists(collection *Collection, model Model, fieldName string) (bool, error) {
	fs, found := collection.spec.fieldsByName[fieldName]
	if !found {
		return false, fmt.Errorf("Type %s has no field called %s", collection.spec.typ.String(), fieldName)
	} else if fs.indexKind == noIndex {
		return false, fmt.Errorf("%s.%s is not an indexed field", collection.spec.typ.String(), fieldName)
	}
	typ := fs.typ
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	switch {
	case typeIsNumeric(typ):
		return numericIndexExists(collection, model, fieldName)
	case typeIsString(typ):
		return stringIndexExists(collection, model, fieldName)
	case typeIsBool(typ):
		return booleanIndexExists(collection, model, fieldName)
	default:
		return false, fmt.Errorf("Unknown indexed field type %s", fs.typ)
	}
}

// expectIndexExists sets an error via t.Error if an index on the given type and field
// does not exist in the database. It also reports an error if collection does not have a field
// called fieldName, the field identified by fieldName is not an indexed field, there was a
// problem connecting to the database, or there was some other unexpected problem.
func expectIndexExists(t *testing.T, collection *Collection, model Model, fieldName string) {
	if exists, err := indexExists(collection, model, fieldName); err != nil {
		t.Errorf("Unexpected error in indexExists: %s", err.Error())
	} else if !exists {
		t.Errorf("Expected an index for %s.%s to exist but it did not", collection.spec.typ.String(), fieldName)
	}
}

// expectIndexDoesNotExist sets an error via t.Error if an index on the given type and field
// does exist in the database. It also reports an error if collection does not have a field
// called fieldName, the field identified by fieldName is not an indexed field, there was a
// problem connecting to the database, or there was some other unexpected problem.
func expectIndexDoesNotExist(t *testing.T, collection *Collection, model Model, fieldName string) {
	if exists, err := indexExists(collection, model, fieldName); err != nil {
		t.Errorf("Unexpected error in indexExists: %s", err.Error())
	} else if exists {
		t.Errorf("Expected an index for %s.%s to not exist but it did", collection.spec.typ.String(), fieldName)
	}
}

// numericIndexExists returns true iff a numeric index on the given type and field exists. It
// reads the current field value from model and if it is a pointer, dereferences it until
// it reaches the underlying value.
func numericIndexExists(collection *Collection, model Model, fieldName string) (bool, error) {
	indexKey, err := collection.FieldIndexKey(fieldName)
	if err != nil {
		return false, err
	}
	fieldValue := reflect.ValueOf(model).Elem().FieldByName(fieldName)
	score := numericScore(fieldValue)
	conn := testPool.NewConn()
	defer func() {
		_ = conn.Close()
	}()
	gotIDs, err := redis.Strings(conn.Do("ZRANGEBYSCORE", indexKey, score, score))
	if err != nil {
		return false, fmt.Errorf("Error in ZRANGEBYSCORE: %s", err.Error())
	}
	return stringSliceContains(gotIDs, model.ModelID()), nil
}

// stringIndexExists returns true iff a string index on the given type and field exists. It
// reads the current field value from model and if it is a pointer, dereferences it until
// it reaches the underlying value.
func stringIndexExists(collection *Collection, model Model, fieldName string) (bool, error) {
	indexKey, err := collection.FieldIndexKey(fieldName)
	if err != nil {
		return false, err
	}
	fieldValue := reflect.ValueOf(model).Elem().FieldByName(fieldName)
	for fieldValue.Kind() == reflect.Ptr {
		fieldValue = fieldValue.Elem()
	}
	memberKey := fieldValue.String() + nullString + model.ModelID()
	conn := testPool.NewConn()
	defer func() {
		_ = conn.Close()
	}()
	reply, err := conn.Do("ZRANK", indexKey, memberKey)
	if err != nil {
		return false, fmt.Errorf("Error in ZRANK: %s", err.Error())
	}
	return reply != nil, nil
}

// booleanIndexExists returns true iff a boolean index on the given type and field exists. It
// reads the current field value from model and if it is a pointer, dereferences it until
// it reaches the underlying value.
func booleanIndexExists(collection *Collection, model Model, fieldName string) (bool, error) {
	indexKey, err := collection.FieldIndexKey(fieldName)
	if err != nil {
		return false, err
	}
	fieldValue := reflect.ValueOf(model).Elem().FieldByName(fieldName)
	score := boolScore(fieldValue)
	conn := testPool.NewConn()
	defer func() {
		_ = conn.Close()
	}()
	gotIDs, err := redis.Strings(conn.Do("ZRANGEBYSCORE", indexKey, score, score))
	if err != nil {
		return false, fmt.Errorf("Error in ZRANGEBYSCORE: %s", err.Error())
	}
	return stringSliceContains(gotIDs, model.ModelID()), nil
}

// byID is a utility type for quickly sorting by id
type byID []*indexedTestModel

func (ms byID) Len() int           { return len(ms) }
func (ms byID) Swap(i, j int)      { ms[i], ms[j] = ms[j], ms[i] }
func (ms byID) Less(i, j int) bool { return ms[i].ModelID() < ms[j].ModelID() }

// expectModelsToBeEqual returns an error if the two slices do not contain the exact
// same models.
func expectModelsToBeEqual(expected []*indexedTestModel, got []*indexedTestModel, orderMatters bool) error {
	if len(expected) != len(got) {
		return fmt.Errorf("Lengths did not match.\nExpected: %v\nBut got:  %v", modelIDs(Models(expected)), modelIDs(Models(got)))
	}
	eCopy, gCopy := make([]*indexedTestModel, len(expected)), make([]*indexedTestModel, len(got))
	copy(eCopy, expected)
	copy(gCopy, got)
	if !orderMatters {
		// if order doesn't matter, first sort by id, which is unique.
		// this way we can do a straightforward comparison
		sort.Sort(byID(eCopy))
		sort.Sort(byID(gCopy))
	}
	for i, e := range eCopy {
		g := gCopy[i]
		if !reflect.DeepEqual(e, g) {
			return fmt.Errorf("Inequality detected at iteration %d.Expected: %+v\nGot:  %+v", i, *e, *g)
		}
	}
	return nil
}

// compareAsStringSet compares expecteds and gots as if they were sets, i.e.,
// it checks if they contain the same values, regardless of order. It returns true
// and an empty string if expecteds and gots contain all the same values and false
// and a detailed message if they do not.
func compareAsStringSet(expecteds, gots []string) (bool, string) {
	// make sure everything in expecteds is also in gots
	for _, e := range expecteds {
		index := indexOfStringSlice(gots, e)
		if index == -1 {
			msg := fmt.Sprintf("Missing expected element: %v", e)
			return false, msg
		}
	}

	// make sure everything in gots is also in expecteds
	for _, g := range gots {
		index := indexOfStringSlice(expecteds, g)
		if index == -1 {
			msg := fmt.Sprintf("Found extra element: %v", g)
			return false, msg
		}
	}

	return true, "ok"
}

// randomInt returns a pseudo-random int between the minimum and maximum
// possible values.
func randomInt() int {
	return rand.Int()
}

// randomString returns a random string of length 16
func randomString() string {
	return uniuri.NewLen(16)
}

// randomBool returns a random bool
func randomBool() bool {
	return rand.Int()%2 == 0
}

// randomFloat returns a random float64
func randomFloat() float64 {
	return rand.Float64()
}

// randomComplex returns a random complex128
func randomComplex() complex128 {
	return cmplx.Rect(randomFloat(), randomFloat())
}

// decrementString subtracts 1 to the last codepoint in s and returns the new string
// E.g. if the input string is "abc" the return will be "abb" because the codepoint
// for 'c' is 99, 99-1 = 98, and the codepoint 98 corresponds to 'b'.
func decrementString(s string) string {
	codepoints := []uint8(s)
	codepoints[len(codepoints)-1] = codepoints[len(codepoints)-1] + 1
	return string(codepoints)
}

// incrementString adds 1 to the last codepoint in s and returns the new string
// E.g. if the input string is "abc" the return will be "abd" because the codepoint
// for 'c' is 99, 99+1 = 100, and the codepoint 100 corresponds to 'd'.
func incrementString(s string) string {
	codepoints := []uint8(s)
	codepoints[len(codepoints)-1] = codepoints[len(codepoints)-1] + 1
	return string(codepoints)
}
