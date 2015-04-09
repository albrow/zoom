// Copyright 2014 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File query.go contains code related to the query abstraction.

package zoom

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Query represents a query which will retrieve some models from
// the database. A Query may consist of one or more query modifiers
// and can be run in several different ways with different query
// finishers.
type Query struct {
	modelSpec *modelSpec
	tx        *Transaction
	includes  []string
	excludes  []string
	order     order
	limit     uint
	offset    uint
	filters   []filter
	err       error
}

func (q *Query) String() string {
	result := fmt.Sprintf("%s.NewQuery()", q.modelSpec.name)
	for _, filter := range q.filters {
		result += fmt.Sprintf(".%s", filter)
	}
	if q.order.fieldName != "" {
		result += fmt.Sprintf(".%s", q.order)
	}
	if q.offset != 0 {
		result += fmt.Sprintf(".Offset(%d)", q.offset)
	}
	if q.limit != 0 {
		result += fmt.Sprintf(".Limit(%d)", q.limit)
	}
	return result
}

type order struct {
	fieldName string
	redisName string
	kind      orderKind
}

func (o order) String() string {
	if o.kind == ascendingOrder {
		return fmt.Sprintf("Order(%s)", o.fieldName)
	} else {
		return fmt.Sprintf("Order(-%s)", o.fieldName)
	}
}

type orderKind int

const (
	ascendingOrder orderKind = iota
	descendingOrder
)

func (ok orderKind) String() string {
	switch ok {
	case ascendingOrder:
		return "ascending"
	case descendingOrder:
		return "descending"
	}
	return ""
}

type filter struct {
	fieldName string
	redisName string
	kind      filterKind
	value     reflect.Value
	indexKind indexKind
}

func (f filter) String() string {
	return fmt.Sprintf(`Filter("%s %s", %v)`, f.fieldName, f.kind, f.value.Interface())
}

type filterKind int

const (
	equalFilter filterKind = iota
	notEqualFilter
	greaterFilter
	lessFilter
	greaterOrEqualFilter
	lessOrEqualFilter
)

func (fk filterKind) String() string {
	switch fk {
	case equalFilter:
		return "="
	case notEqualFilter:
		return "!="
	case greaterFilter:
		return ">"
	case lessFilter:
		return "<"
	case greaterOrEqualFilter:
		return ">="
	case lessOrEqualFilter:
		return "<="
	}
	return ""
}

var filterSymbols = map[string]filterKind{
	"=":  equalFilter,
	"!=": notEqualFilter,
	">":  greaterFilter,
	"<":  lessFilter,
	">=": greaterOrEqualFilter,
	"<=": lessOrEqualFilter,
}

// delString is a string consisting of only the the ASCII DEL character. It is
// used as a prefix for queries on string indexes.
var delString string = string([]byte{byte(127)})

// NewQuery is used to construct a query. The query returned can be chained
// together with one or more query modifiers, and then executed using the Run
// Scan, Count, or Ids methods. If no query modifiers are used, running the query
// will return all models of the given type in uspecified order.
func (modelType *ModelType) NewQuery() *Query {
	return &Query{
		modelSpec: modelType.spec,
	}
}

func (q *Query) setError(e error) {
	if q.err == nil {
		q.err = e
	}
}

// Order specifies a field by which to sort the records and the order in which
// records should be sorted. fieldName should be a field in the struct type
// specified by the modelName argument in the query constructor. By default, the
// records are sorted by ascending order. To sort by descending order, put a
// negative sign before the field name. Zoom can only sort by fields which have
// been indexed, i.e. those which have the `zoom:"index"` struct tag. However,
// in the future this may change. Only one order may be specified per query.
// However in the future, secondary orders may be allowed, and will take effect
// when two or more models have the same value for the primary order field.
// Order will set an error on the query if the fieldName is invalid, if another
// order has already been applied to the query, or if the fieldName specified
// does not correspond to an indexed field. The error, same as any other error
// that occurs during the lifetime of the query, is not returned until the Query
// is executed. When the query is executed the first error that occured during
// the lifetime of the query object (if any) will be returned.
func (q *Query) Order(fieldName string) *Query {
	if q.order.fieldName != "" {
		// TODO: allow secondary sort orders?
		q.setError(errors.New("zoom: error in Query.Order: previous order already specified. Only one order per query is allowed."))
	}

	// Check for the presence of the "-" prefix
	var orderKind orderKind
	if strings.HasPrefix(fieldName, "-") {
		orderKind = descendingOrder
		// remove the "-" prefix
		fieldName = fieldName[1:]
	} else {
		orderKind = ascendingOrder
	}

	// Get the redisName for the given fieldName
	fs, found := q.modelSpec.fieldsByName[fieldName]
	if !found {
		err := fmt.Errorf("zoom: error in Query.Order: could not find field %s in type %s", fieldName, q.modelSpec.typ.String())
		q.setError(err)
	}
	q.order = order{
		fieldName: fs.name,
		redisName: fs.redisName,
		kind:      orderKind,
	}
	return q
}

// Limit specifies an upper limit on the number of records to return. If amount
// is 0, no limit will be applied. The default value is 0.
func (q *Query) Limit(amount uint) *Query {
	q.limit = amount
	return q
}

// Offset specifies a starting index (inclusive) from which to start counting
// records that will be returned. The default value is 0.
func (q *Query) Offset(amount uint) *Query {
	q.offset = amount
	return q
}

// Include specifies one or more field names which will be read from the
// database and scanned into the resulting models when the query is run. Field
// names which are not specified in Include will not be read or scanned. You can
// only use one of Include or Exclude, not both on the same query. Include will
// set an error if you try to use it with Exclude on the same query. The error,
// same as any other error that occurs during the lifetime of the query, is not
// returned until the Query is executed. When the query is executed the first
// error that occured during the lifetime of the query object (if any) will be
// returned.
func (q *Query) Include(fields ...string) *Query {
	if len(q.excludes) > 0 {
		q.setError(errors.New("zoom: cannot use both Include and Exclude modifiers on a query"))
		return q
	}
	q.includes = append(q.includes, fields...)
	return q
}

// Exclude specifies one or more field names which will *not* be read from the
// database and scanned. Any other fields *will* be read and scanned into the
// resulting models when the query is run. You can only use one of Include or
// Exclude, not both on the same query. Exclude will set an error if you try to
// use it with Include on the same query. The error, same as any other error
// that occurs during the lifetime of the query, is not returned until the Query
// is executed. When the query is executed the first error that occured during
// the lifetime of the query object (if any) will be returned.
func (q *Query) Exclude(fields ...string) *Query {
	if len(q.includes) > 0 {
		q.setError(errors.New("zoom: cannot use both Include and Exclude modifiers on a query"))
		return q
	}
	q.excludes = append(q.excludes, fields...)
	return q
}

// getIncludes parses the includes and excludes properties to return a list of
// fieldNames which should be included in all find operations. a return value of
// nil means that all fields should be considered.
func (q *Query) getIncludes() []string {
	if len(q.includes) != 0 {
		return q.includes
	} else if len(q.excludes) != 0 {
		results := q.modelSpec.fieldNames()
		for _, name := range q.excludes {
			results = removeElementFromStringSlice(results, name)
		}
		return results
	}
	return nil
}

// Filter applies a filter to the query, which will cause the query to only
// return models with attributes matching the expression. filterString should be
// an expression which includes a fieldName, a space, and an operator in that
// order. Operators must be one of "=", "!=", ">", "<", ">=", or "<=". You can
// only use Filter on fields which are indexed, i.e. those which have the
// `zoom:"index"` struct tag. If multiple filters are applied to the same query,
// the query will only return models which have matches for ALL of the filters.
// I.e. applying multiple filters is logially equivalent to combining them with
// a AND or INTERSECT operator. Filter will set an error on the query if the
// arguments are improperly formated, if the field you are attempting to filter
// is not indexed, or if the type of value does not match the type of the field.
// The error, same as any other error that occurs during the lifetime of the
// query, is not returned until the Query is executed. When the query is
// executed the first error that occured during the lifetime of the query object
// (if any) will be returned.
func (q *Query) Filter(filterString string, value interface{}) *Query {
	fieldName, operator, err := splitFilterString(filterString)
	if err != nil {
		q.setError(err)
		return q
	}
	if fieldName == "Id" {
		// Special case for Id
		return q.filterById(operator, value)
	}
	filter := filter{
		fieldName: fieldName,
	}
	// Get the filterKind based on the operator
	if filterKind, found := filterSymbols[operator]; !found {
		q.setError(errors.New("zoom: invalid operator in fieldStr. should be one of =, !=, >, <, >=, or <=."))
		return q
	} else {
		filter.kind = filterKind
	}
	// Get the redisName and indexKind based on the fieldName
	fs, found := q.modelSpec.fieldsByName[fieldName]
	if !found {
		err := fmt.Errorf("zoom: error in Query.Order: could not find field %s in type %s", fieldName, q.modelSpec.typ.String())
		q.setError(err)
		return q
	}
	filter.redisName = fs.redisName
	if fs.indexKind == noIndex {
		err := fmt.Errorf("zoom: filters are only allowed on indexed fields. %s.%s is not indexed. You can index it by adding the `zoom:\"index\"` struct tag.", q.modelSpec.typ.String(), fieldName)
		q.setError(err)
		return q
	}
	filter.indexKind = fs.indexKind
	// Get type of the field and make sure it matches type of value arg
	// Here we iterate through pointer inderections. This is so you can
	// just pass in a primative instead of a pointer to a primative for
	// filtering on fields which have pointer values.
	valueType := reflect.TypeOf(value)
	valueVal := reflect.ValueOf(value)
	for valueType.Kind() == reflect.Ptr {
		valueType = valueType.Elem()
		valueVal = valueVal.Elem()
		if !valueVal.IsValid() {
			q.setError(errors.New("zoom: invalid value arg for Filter. Is it a nil pointer?"))
			return q
		}
	}
	// Also dereference the field type to reach the underlying type.
	fieldType := fs.typ
	for fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}
	if valueType != fs.typ {
		err := fmt.Errorf("zoom: invalid value arg for Filter. Type of value (%T) does not match type of field (%s).", value, fieldType.String())
		q.setError(err)
		return q
	}
	filter.value = valueVal
	q.filters = append(q.filters, filter)
	return q
}

func splitFilterString(filterString string) (fieldName string, operator string, err error) {
	tokens := strings.Split(filterString, " ")
	if len(tokens) != 2 {
		return "", "", errors.New("zoom: too many spaces in fieldStr argument. should be a field name, a space, and an operator.")
	}
	return tokens[0], tokens[1], nil
}

func (q *Query) filterById(operator string, value interface{}) *Query {
	if operator != "=" {
		q.setError(errors.New("zoom: only the = operator can be used with Filter on Id field."))
		return q
	}
	idVal := reflect.ValueOf(value)
	if idVal.Kind() != reflect.String {
		err := fmt.Errorf("zoom: for a Filter on Id field, value must be a string type. Got type %T", value)
		q.setError(err)
		return q
	}
	f := filter{
		fieldName: "Id",
		redisName: "Id",
		kind:      equalFilter,
		value:     idVal,
	}
	q.filters = append(q.filters, f)
	return q
}

// Run executes the query and scans the results into models. The type of models
// should be a pointer to a slice of pointers to a registered Model. Run will
// return the first error that occured during the lifetime of the query object
// (if any). It will also return an error if models is the wrong type.
func (q *Query) Run(models interface{}) error {
	return nil
}

// RunOne is exactly like Run but finds only the first model that fits the
// query criteria and scans the values into model. If no model fits the criteria,
// an error will be returned.
func (q *Query) RunOne(model Model) error {
	return nil
}

// Count counts the number of models that would be returned by the query without
// actually retreiving the models themselves. Count will also return the first
// error that occured during the lifetime of the query object (if any).
// Otherwise, the second return value will be nil.
func (q *Query) Count() (int, error) {
	return 0, nil
}

// Ids returns only the ids of the models without actually retreiving the
// models themselves. Ids will return the first error that occured
// during the lifetime of the query object (if any).
func (q *Query) Ids() ([]string, error) {
	return nil, nil
}

// converts limit and offset to start and stop values for cases where redis
// requires them. NOTE start cannot be negative, but stop can be
func (q *Query) getStartStop() (int, int) {
	start := int(q.offset)
	stop := -1
	if q.limit != 0 {
		stop = int(start) + int(q.limit) - 1
	}
	return start, stop
}

func getMinMaxForNumericFilter(f filter) (min interface{}, max interface{}) {
	switch f.kind {
	case equalFilter:
		min, max = f.value.Interface(), f.value.Interface()
	case lessFilter:
		min = "-inf"
		// use "(" for exclusive
		max = fmt.Sprintf("(%v", f.value.Interface())
	case greaterFilter:
		// use "(" for exclusive
		min = fmt.Sprintf("(%v", f.value.Interface())
		max = "+inf"
	case lessOrEqualFilter:
		min = "-inf"
		max = f.value.Interface()
	case greaterOrEqualFilter:
		min = f.value.Interface()
		max = "+inf"
	}
	return min, max
}

// string returns a string representation of the filterKind
func (fk filterKind) string() string {
	switch fk {
	case equalFilter:
		return "="
	case notEqualFilter:
		return "!="
	case greaterFilter:
		return ">"
	case lessFilter:
		return "<"
	case greaterOrEqualFilter:
		return ">="
	case lessOrEqualFilter:
		return "<="
	}
	return ""
}

// string returns a string representation of the filter
func (f filter) string() string {
	return fmt.Sprintf("(filter %s %s %v)", f.fieldName, f.kind.string(), f.value.Interface())
}

// string returns a string representation of the order
func (o order) string() string {
	if o.fieldName == "" {
		return ""
	}
	switch o.kind {
	case ascendingOrder:
		return fmt.Sprintf("(order %s)", o.fieldName)
	case descendingOrder:
		return fmt.Sprintf("(order -%s)", o.fieldName)
	}
	return ""
}
