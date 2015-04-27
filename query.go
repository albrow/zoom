// Copyright 2014 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File query.go contains code related to the query abstraction.

package zoom

import (
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
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
	if q.hasOrder() {
		result += fmt.Sprintf(".%s", q.order)
	}
	if q.hasOffset() {
		result += fmt.Sprintf(".Offset(%d)", q.offset)
	}
	if q.hasLimit() {
		result += fmt.Sprintf(".Limit(%d)", q.limit)
	}
	if q.hasIncludes() {
		result += fmt.Sprintf(`.Include("%s")`, strings.Join(q.includes, `", "`))
	} else if q.hasExcludes() {
		result += fmt.Sprintf(`.Exclude("%s")`, strings.Join(q.excludes, `", "`))
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
		return fmt.Sprintf(`Order("%s")`, o.fieldName)
	} else {
		return fmt.Sprintf(`Order("-%s")`, o.fieldName)
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
	fieldSpec *fieldSpec
	op        filterOp
	value     reflect.Value
}

func (f filter) String() string {
	if f.value.Kind() == reflect.String {
		return fmt.Sprintf(`Filter("%s %s", "%s")`, f.fieldSpec.name, f.op, f.value.String())
	} else {
		return fmt.Sprintf(`Filter("%s %s", %v)`, f.fieldSpec.name, f.op, f.value.Interface())
	}
}

type filterOp int

const (
	equalOp filterOp = iota
	notEqualOp
	greaterOp
	lessOp
	greaterOrEqualOp
	lessOrEqualOp
)

func (fk filterOp) String() string {
	switch fk {
	case equalOp:
		return "="
	case notEqualOp:
		return "!="
	case greaterOp:
		return ">"
	case lessOp:
		return "<"
	case greaterOrEqualOp:
		return ">="
	case lessOrEqualOp:
		return "<="
	}
	return ""
}

var filterOps = map[string]filterOp{
	"=":  equalOp,
	"!=": notEqualOp,
	">":  greaterOp,
	"<":  lessOp,
	">=": greaterOrEqualOp,
	"<=": lessOrEqualOp,
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
	if !q.hasError() {
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
	if q.hasOrder() {
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
	if q.hasExcludes() {
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
	if q.hasIncludes() {
		q.setError(errors.New("zoom: cannot use both Include and Exclude modifiers on a query"))
		return q
	}
	q.excludes = append(q.excludes, fields...)
	return q
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
	// Parse the filter operator
	filterOp, found := filterOps[operator]
	if !found {
		q.setError(errors.New("zoom: invalid Filter operator in fieldStr. should be one of =, !=, >, <, >=, or <=."))
		return q
	}
	// Get the fieldSpec for the given fieldName
	fieldSpec, found := q.modelSpec.fieldsByName[fieldName]
	if !found {
		err := fmt.Errorf("zoom: error in Query.Order: could not find field %s in type %s", fieldName, q.modelSpec.typ.String())
		q.setError(err)
		return q
	}
	// Make sure the field is an indexed field
	if fieldSpec.indexKind == noIndex {
		err := fmt.Errorf("zoom: filters are only allowed on indexed fields. %s.%s is not indexed. You can index it by adding the `zoom:\"index\"` struct tag.", q.modelSpec.typ.String(), fieldName)
		q.setError(err)
		return q
	}
	filter := filter{
		fieldSpec: fieldSpec,
		op:        filterOp,
	}
	// Make sure the given value is the correct type
	if err := filter.checkValType(value); err != nil {
		q.setError(err)
		return q
	}
	filter.value = reflect.ValueOf(value)
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

// checkValType returns an error if the type of value does not correspond to
// filter.fieldSpec.
func (filter filter) checkValType(value interface{}) error {
	// Here we iterate through pointer inderections. This is so you can
	// just pass in a primative instead of a pointer to a primative for
	// filtering on fields which have pointer values.
	valueType := reflect.TypeOf(value)
	valueVal := reflect.ValueOf(value)
	for valueType.Kind() == reflect.Ptr {
		valueType = valueType.Elem()
		valueVal = valueVal.Elem()
		if !valueVal.IsValid() {
			return errors.New("zoom: invalid value arg for Filter. Is it a nil pointer?")
		}
	}
	// Also dereference the field type to reach the underlying type.
	fieldType := filter.fieldSpec.typ
	for fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}
	if valueType != fieldType {
		return fmt.Errorf("zoom: invalid value arg for Filter. Type of value (%T) does not match type of field (%s).", value, fieldType.String())
	}
	return nil
}

// Run executes the query and scans the results into models. The type of models
// should be a pointer to a slice of pointers to a registered Model. Run will
// return the first error that occured during the lifetime of the query object
// (if any). It will also return an error if models is the wrong type.
func (q *Query) Run(models interface{}) error {
	// TODO: type-checking
	q.tx = NewTransaction()
	idsKey, tmpKeys, err := q.generateIdsSet()
	if err != nil {
		return err
	}
	limit := int(q.limit)
	if limit == 0 {
		// In our query syntax, a limit of 0 means unlimited
		// But in redis, -1 means unlimited
		limit = -1
	}
	sortArgs := q.modelSpec.sortArgs(idsKey, q.redisFieldNames(), limit, q.offset, q.order.kind)
	q.tx.Command("SORT", sortArgs, newScanModelsHandler(q.modelSpec, append(q.fieldNames(), "-"), models))
	if len(tmpKeys) > 0 {
		q.tx.Command("DEL", (redis.Args{}).Add(tmpKeys...), nil)
	}
	if err := q.tx.Exec(); err != nil {
		return err
	}
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
func (q *Query) Count() (uint, error) {
	if !q.hasFilters() {
		// Just return the number of ids in the all index set
		conn := Conn()
		defer conn.Close()
		count64, err := redis.Uint64(conn.Do("SCARD", q.modelSpec.allIndexKey()))
		if err != nil {
			return 0, nil
		}
		count := uint(count64)
		// Apply math to take into account limit and offset
		switch {
		case !q.hasLimit() && !q.hasOffset():
			return count, nil
		default:
			if q.hasOffset() {
				count = count - q.offset
			}
			if q.hasLimit() && q.limit < count {
				count = q.limit
			}
			return count, nil
		}
	}
	return 0, nil
}

// Ids returns only the ids of the models without actually retreiving the
// models themselves. Ids will return the first error that occured
// during the lifetime of the query object (if any).
func (q *Query) Ids() ([]string, error) {
	q.tx = NewTransaction()
	idsKey, _, err := q.generateIdsSet()
	if err != nil {
		return nil, err
	}
	limit := int(q.limit)
	if limit == 0 {
		// In our query syntax, a limit of 0 means unlimited
		// But in redis, -1 means unlimited
		limit = -1
	}
	sortArgs := q.modelSpec.sortArgs(idsKey, nil, limit, q.offset, q.order.kind)
	ids := []string{}
	q.tx.Command("SORT", sortArgs, newScanStringsHandler(&ids))
	// if len(tmpKeys) > 0 {
	// 	q.tx.Command("DEL", (redis.Args{}).Add(tmpKeys...), nil)
	// }
	if err := q.tx.Exec(); err != nil {
		return nil, err
	}
	return ids, nil
}

func (q *Query) generateIdsSet() (idsKey string, tmpKeys []interface{}, err error) {
	idsKey = q.modelSpec.allIndexKey()
	tmpKeys = []interface{}{}
	if q.hasOrder() {
		fieldIndexKey, err := q.modelSpec.fieldIndexKey(q.order.fieldName)
		if err != nil {
			return "", nil, err
		}
		fieldSpec := q.modelSpec.fieldsByName[q.order.fieldName]
		if fieldSpec.indexKind == stringIndex {
			// If the order is a string field, we need to extract the ids before
			// we use ZRANGE. Create a temporary set to store the ordered ids
			orderedIdsKey := "tmp:" + q.modelSpec.name + ":" + q.order.redisName + ":orderIds"
			tmpKeys = append(tmpKeys, orderedIdsKey)
			idsKey = orderedIdsKey
			// TODO: if there is a filter on the same field, pass the start and stop
			// parameters to the script
			q.tx.extractIdsFromStringIndex(fieldIndexKey, orderedIdsKey)
		} else {
			idsKey = fieldIndexKey
		}
	}
	if q.hasFilters() {
		filteredIdsKey := "tmp:" + q.modelSpec.name + ":filterIds"
		tmpKeys = append(tmpKeys, filteredIdsKey)
		for i, filter := range q.filters {
			if filter.fieldSpec.name == q.order.fieldName {
				// If there is a filter on the same field that the query is ordered by,
				// it was already covered in the case above
				continue
			}
			if i == 0 {
				// The first time, we should intersect with the ids key from above
				q.intersectFilter(filter, idsKey, filteredIdsKey)
			} else {
				// All other times, we should intersect with the filteredIdsKey itself
				q.intersectFilter(filter, filteredIdsKey, filteredIdsKey)
			}
		}
		idsKey = filteredIdsKey
	}
	return idsKey, tmpKeys, nil
}

func (q *Query) intersectFilter(filter filter, origKey string, destKey string) error {
	fieldIndexKey, err := q.modelSpec.fieldIndexKey(filter.fieldSpec.name)
	if err != nil {
		return err
	}
	q.tx.Command("ZINTERSTORE", redis.Args{destKey, 2, origKey, fieldIndexKey, "WEIGHTS", 0, 1}, nil)
	switch filter.fieldSpec.indexKind {
	case numericIndex:
		return q.intersectNumericFilter(filter, origKey, destKey)
	case booleanIndex:
		return q.intersectBoolFilter(filter, origKey, destKey)
	case stringIndex:
		return q.intersectStringFilter(filter, origKey, destKey)
	}
	return nil
}

func (q *Query) intersectNumericFilter(filter filter, origKey string, destKey string) error {
	if filter.op == equalOp {
		// Special case for equal. We need to do two remove operations.
		// NOTE: is there a more effecient way to do this? Currently we're removing almost everything.
		// 1) Remove everying between -inf and filter.value (exclusive)
		valueExclusive := fmt.Sprintf("(%v", filter.value.Interface())
		q.tx.Command("ZREMRANGEBYSCORE", redis.Args{destKey, "-inf", valueExclusive}, nil)
		// 2) Remove everything between filter.value (exclusive) and +inf
		q.tx.Command("ZREMRANGEBYSCORE", redis.Args{destKey, valueExclusive, "+inf"}, nil)
	} else {
		var min, max interface{}
		switch filter.op {
		case lessOp:
			min = filter.value.Interface()
			max = "+inf"
		case greaterOp:
			min = "-inf"
			max = filter.value.Interface()
		case lessOrEqualOp:
			min = fmt.Sprintf("(%v", filter.value.Interface())
			max = "+inf"
		case greaterOrEqualOp:
			min = "-inf"
			// use "(" for exclusive
			max = fmt.Sprintf("(%v", filter.value.Interface())
		case notEqualOp:
			min, max = filter.value.Interface(), filter.value.Interface()
		}
		q.tx.Command("ZREMRANGEBYSCORE", redis.Args{destKey, min, max}, nil)
	}
	return nil
}

func (q *Query) intersectBoolFilter(filter filter, origKey string, destKey string) error {
	return fmt.Errorf("intersectBoolFilter not yet implemented")
}

func (q *Query) intersectStringFilter(filter filter, origKey string, destKey string) error {
	return fmt.Errorf("intersectStringFilter not yet implemented")
}

// fieldNames parses the includes and excludes properties to return a list of
// field names which should be included in all find operations. If there are no
// includes or excludes, it returns all the field names.
func (q *Query) fieldNames() []string {
	switch {
	case q.hasIncludes():
		return q.includes
	case q.hasExcludes():
		results := q.modelSpec.fieldNames()
		for _, name := range q.excludes {
			results = removeElementFromStringSlice(results, name)
		}
		return results
	default:
		return q.modelSpec.fieldNames()
	}
}

// redisFieldNames parses the includes and excludes properties to return a list of
// redis names for each field which should be included in all find operations. If
// there are no includes or excludes, it returns the redis names for all fields.
func (q *Query) redisFieldNames() []string {
	fieldNames := q.fieldNames()
	redisNames := []string{}
	for _, fieldName := range fieldNames {
		redisNames = append(redisNames, q.modelSpec.fieldsByName[fieldName].redisName)
	}
	return redisNames
}

// converts limit and offset to start and stop values for cases where redis
// requires them. NOTE start cannot be negative, but stop can be
func (q *Query) getStartStop() (start int, stop int) {
	start = int(q.offset)
	stop = -1
	if q.hasLimit() {
		stop = int(start) + int(q.limit) - 1
	}
	return start, stop
}

func (q *Query) hasFilters() bool {
	return len(q.filters) > 0
}

func (q *Query) hasOrder() bool {
	return q.order.fieldName != ""
}

func (q *Query) hasLimit() bool {
	return q.limit != 0
}

func (q *Query) hasOffset() bool {
	return q.offset != 0
}

func (q *Query) hasIncludes() bool {
	return len(q.includes) > 0
}

func (q *Query) hasExcludes() bool {
	return len(q.excludes) > 0
}

func (q *Query) hasError() bool {
	return q.err != nil
}
