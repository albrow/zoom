package zoom

import (
	"errors"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestAlwaysErrorHandler(t *testing.T) {
	expectedErr := errors.New("This is an error")
	handler := newAlwaysErrorHandler(expectedErr)
	gotErr := handler([]byte("This is a reply"))
	if gotErr.Error() != expectedErr.Error() {
		t.Errorf("Expected %v but got %v", expectedErr, gotErr)
	}
}

func TestScanIntHandler(t *testing.T) {
	i := 5
	expectedValue := 3
	handler := NewScanIntHandler(&i)
	if err := handler([]byte("3")); err != nil {
		t.Fatal(err)
	}
	if i != expectedValue {
		t.Errorf("Expected %v but got %v", expectedValue, i)
	}
}

func TestScanBoolHandler(t *testing.T) {
	b := false
	expectedValue := true
	handler := NewScanBoolHandler(&b)
	if err := handler([]byte("true")); err != nil {
		t.Fatal(err)
	}
	if b != expectedValue {
		t.Errorf("Expected %v but got %v", expectedValue, b)
	}
}

func TestScanStringHandler(t *testing.T) {
	s := "foo"
	expectedValue := "bar"
	handler := NewScanStringHandler(&s)
	if err := handler([]byte("bar")); err != nil {
		t.Fatal(err)
	}
	if s != expectedValue {
		t.Errorf("Expected %v but got %v", expectedValue, s)
	}
}

func TestScanFloat64Handler(t *testing.T) {
	f := 4.3
	expectedValue := 42.0
	handler := NewScanFloat64Handler(&f)
	if err := handler([]byte("42.0")); err != nil {
		t.Fatal(err)
	}
	if f != expectedValue {
		t.Errorf("Expected %v but got %v", expectedValue, f)
	}
}

func TestScanStringsHandler(t *testing.T) {
	strings := []string{"foo", "bar", "biz"}
	expectedValue := []string{"biz", "baz", "bar"}
	handler := NewScanStringsHandler(&strings)
	if err := handler([]interface{}{
		[]byte("biz"),
		[]byte("baz"),
		[]byte("bar"),
	}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(strings, expectedValue) {
		t.Errorf("Expected %v but got %v", expectedValue, strings)
	}
}

func TestScanModelHandler(t *testing.T) {
	testingSetUp()
	defer testingTearDown()
	model := testModel{
		Int:    1,
		String: "foo",
		Bool:   true,
	}
	expectedValue := testModel{
		Int:    38,
		String: "bar",
		Bool:   true,
		RandomID: RandomID{
			ID: "thisIsANewID",
		},
	}
	fieldNames := []string{"String", "-", "Int"}
	handler := NewScanModelHandler(fieldNames, &model)
	if err := handler([]interface{}{
		[]byte("bar"),
		[]byte("thisIsANewID"),
		[]byte("38"),
	}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(model, expectedValue) {
		t.Errorf("Expected %v but got %v", expectedValue, model)
	}
}

func TestScanModelsHandler(t *testing.T) {
	testingSetUp()
	defer testingTearDown()
	models := []*testModel{}
	expectedValue := []*testModel{
		{
			Int:    38,
			String: "bar",
			Bool:   false,
			RandomID: RandomID{
				ID: "thisIsANewID",
			},
		},
		{
			Int:    37,
			String: "biz",
			Bool:   false,
			RandomID: RandomID{
				ID: "thisIsAlsoANewID",
			},
		},
	}
	fieldNames := []string{"String", "-", "Int"}
	handler := NewScanModelsHandler(testModels, fieldNames, &models)
	if err := handler([]interface{}{
		[]byte("bar"),
		[]byte("thisIsANewID"),
		[]byte("38"),
		[]byte("biz"),
		[]byte("thisIsAlsoANewID"),
		[]byte("37"),
	}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(models, expectedValue) {
		t.Errorf("\nExpected: %s\nBut got:  %s\n",
			spew.Sprint(expectedValue),
			spew.Sprint(models),
		)
	}
}

func TestScanOneModelHandler(t *testing.T) {
	testingSetUp()
	defer testingTearDown()
	expected := &testModel{
		Int:    38,
		String: "bar",
		Bool:   false,
		RandomID: RandomID{
			ID: "thisIsAnID",
		},
	}
	fieldNames := []string{"String", "-", "Int"}
	got := &testModel{}
	handler := newScanOneModelHandler(testModels.NewQuery().query, testModels.spec, fieldNames, got)
	if err := handler([]interface{}{
		[]byte("bar"),
		[]byte("thisIsAnID"),
		[]byte("38"),
	}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, got) {
		t.Errorf("\nExpected: %s\nBut got:  %s\n",
			spew.Sprint(expected),
			spew.Sprint(got),
		)
	}
	// If reply is nil, the ReplyHandler should return a ModelNotFoundError.
	if err := handler(nil); err == nil {
		t.Error("Expected error but got none")
	} else if _, ok := err.(ModelNotFoundError); !ok {
		t.Errorf("Expected ModelNotFoundError but got: %T: %v", err, err)
	}
}
