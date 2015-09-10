// Copyright 2015 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File query.go contains code related to the query abstraction.

package zoom

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/garyburd/redigo/redis"
)

// Query represents a query which will retrieve some models from
// the database. A Query may consist of one or more query modifiers
// (e.g. Filter or Order) and may be executed with a query finisher
// (e.g. Run or Ids).
type Query struct {
	collection *Collection
	pool       *Pool
	tx         *Transaction
	includes   []string
	excludes   []string
	order      order
	limit      uint
	offset     uint
	filters    []filter
	err        error
}

// String satisfies fmt.Stringer and prints out the query in a format that
// matches the go code used to declare it.
func (q *Query) String() string {
	result := fmt.Sprintf("%s.NewQuery()", q.collection.Name())
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

// NewQuery is used to construct a query. The query returned can be chained
// together with one or more query modifiers (e.g. Filter or Order), and then
// executed using the Run, RunOne, Count, or Ids methods. If no query modifiers
// are used, running the query will return all models of the given type in uspecified
// order. Queries use delated execution, so nothing touches the database until you
// execute it.
func (collection *Collection) NewQuery() *Query {
	q := &Query{
		collection: collection,
		pool:       collection.pool,
	}
	// For now, only indexed collections are queryable. This might change in
	// future versions.
	if !collection.index {
		q.setError(fmt.Errorf("zoom: error in NewQuery: Only indexed collections are queryable. To index the collection, pass CollectionOptions to the NewCollection method."))
	}
	return q
}

// setError sets the err property of q only if it has not already been set
func (q *Query) setError(e error) {
	if !q.hasError() {
		q.err = e
	}
}

// Order specifies a field by which to sort the models. fieldName should be
// a field in the struct type corresponding to the Collection used in the query
// constructor. By default, the records are sorted by ascending order by the given
// field. To sort by descending order, put a negative sign before the field name.
// Zoom can only sort by fields which have been indexed, i.e. those which have the
// `zoom:"index"` struct tag. However, in the future this may change. Only one
// order may be specified per query. However in the future, secondary orders may be
// allowed, and will take effect when two or more models have the same value for the
// primary order field. Order will set an error on the query if the fieldName is invalid,
// if another order has already been applied to the query, or if the fieldName specified
// does not correspond to an indexed field. The error, same as any other error
// that occurs during the lifetime of the query, is not returned until the query
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
	fs, found := q.collection.spec.fieldsByName[fieldName]
	if !found {
		err := fmt.Errorf("zoom: error in Query.Order: could not find field %s in type %s", fieldName, q.collection.spec.typ.String())
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
// returned until the query is executed. When the query is executed the first
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
// that occurs during the lifetime of the query, is not returned until the query
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
// query, is not returned until the query is executed. When the query is
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
	fieldSpec, found := q.collection.spec.fieldsByName[fieldName]
	if !found {
		err := fmt.Errorf("zoom: error in Query.Order: could not find field %s in type %s", fieldName, q.collection.spec.typ.String())
		q.setError(err)
		return q
	}
	// Make sure the field is an indexed field
	if fieldSpec.indexKind == noIndex {
		err := fmt.Errorf("zoom: filters are only allowed on indexed fields. %s.%s is not indexed. You can index it by adding the `zoom:\"index\"` struct tag.", q.collection.spec.typ.String(), fieldName)
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
	if q.hasError() {
		return q.err
	}
	if err := q.collection.spec.checkModelsType(models); err != nil {
		return err
	}
	q.tx = q.pool.NewTransaction()
	idsKey, tmpKeys, err := q.generateIdsSet()
	if err != nil {
		if len(tmpKeys) > 0 {
			q.tx.Command("DEL", (redis.Args{}).Add(tmpKeys...), nil)
		}
		return err
	}
	limit := int(q.limit)
	if limit == 0 {
		// In our query syntax, a limit of 0 means unlimited
		// But in redis, -1 means unlimited
		limit = -1
	}
	sortArgs := q.collection.spec.sortArgs(idsKey, q.redisFieldNames(), limit, q.offset, q.order.kind)
	q.tx.Command("SORT", sortArgs, newScanModelsHandler(q.collection.spec, append(q.fieldNames(), "-"), models))
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
	if q.hasError() {
		return q.err
	}
	if err := q.collection.spec.checkModelType(model); err != nil {
		return err
	}
	models := reflect.New(reflect.SliceOf(reflect.TypeOf(model)))
	if err := q.Run(models.Interface()); err != nil {
		return err
	}
	if models.Elem().Len() == 0 {
		msg := fmt.Sprintf("Could not find a model with the given query criteria: %s", q)
		return ModelNotFoundError{Msg: msg}
	} else {
		modelVal := models.Elem().Index(0)
		reflect.ValueOf(model).Elem().Set(modelVal.Elem())
	}
	return nil
}

// Count counts the number of models that would be returned by the query without
// actually retreiving the models themselves. Count will also return the first
// error that occured during the lifetime of the query object (if any).
// Otherwise, the second return value will be nil.
func (q *Query) Count() (uint, error) {
	if q.hasError() {
		return 0, q.err
	}
	if !q.hasFilters() {
		// Just return the number of ids in the all index set
		conn := q.pool.NewConn()
		defer conn.Close()
		count64, err := redis.Uint64(conn.Do("SCARD", q.collection.spec.indexKey()))
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
	} else {
		// If the query has filters, it is difficult to do any optimizations.
		// Instead we'll just count the number of ids that match the query
		// criteria.
		ids, err := q.Ids()
		if err != nil {
			return 0, err
		}
		return uint(len(ids)), nil
	}
}

// Ids returns only the ids of the models without actually retreiving the
// models themselves. Ids will return the first error that occured
// during the lifetime of the query object (if any).
func (q *Query) Ids() ([]string, error) {
	if q.hasError() {
		return nil, q.err
	}
	q.tx = q.pool.NewTransaction()
	idsKey, tmpKeys, err := q.generateIdsSet()
	if err != nil {
		if len(tmpKeys) > 0 {
			q.tx.Command("DEL", (redis.Args{}).Add(tmpKeys...), nil)
		}
		return nil, err
	}
	limit := int(q.limit)
	if limit == 0 {
		// In our query syntax, a limit of 0 means unlimited
		// But in redis, -1 means unlimited
		limit = -1
	}
	sortArgs := q.collection.spec.sortArgs(idsKey, nil, limit, q.offset, q.order.kind)
	ids := []string{}
	q.tx.Command("SORT", sortArgs, newScanStringsHandler(&ids))
	if len(tmpKeys) > 0 {
		q.tx.Command("DEL", (redis.Args{}).Add(tmpKeys...), nil)
	}
	if err := q.tx.Exec(); err != nil {
		return nil, err
	}
	return ids, nil
}

// generateIdsSet will return the key of a set or sorted set that contains all the ids
// which match the query criteria. It may also return some temporary keys which were created
// during the process of creating the set of ids. Note that tmpKeys may contain idsKey itself,
// so the temporary keys should not be deleted until after the ids have been read from idsKey.
func (q *Query) generateIdsSet() (idsKey string, tmpKeys []interface{}, err error) {
	idsKey = q.collection.spec.indexKey()
	tmpKeys = []interface{}{}
	if q.hasOrder() {
		fieldIndexKey, err := q.collection.spec.fieldIndexKey(q.order.fieldName)
		if err != nil {
			return "", nil, err
		}
		fieldSpec := q.collection.spec.fieldsByName[q.order.fieldName]
		if fieldSpec.indexKind == stringIndex {
			// If the order is a string field, we need to extract the ids before
			// we use ZRANGE. Create a temporary set to store the ordered ids
			orderedIdsKey := generateRandomKey("order:" + q.order.fieldName)
			tmpKeys = append(tmpKeys, orderedIdsKey)
			idsKey = orderedIdsKey
			// TODO: if there is a filter on the same field, pass the start and stop
			// parameters to the script
			q.tx.extractIdsFromStringIndex(fieldIndexKey, orderedIdsKey, "-", "+")
		} else {
			idsKey = fieldIndexKey
		}
	}
	if q.hasFilters() {
		filteredIdsKey := generateRandomKey("filter:all")
		tmpKeys = append(tmpKeys, filteredIdsKey)
		for i, filter := range q.filters {
			if i == 0 {
				// The first time, we should intersect with the ids key from above
				if err := q.intersectFilter(filter, idsKey, filteredIdsKey); err != nil {
					return "", tmpKeys, err
				}
			} else {
				// All other times, we should intersect with the filteredIdsKey itself
				if err := q.intersectFilter(filter, filteredIdsKey, filteredIdsKey); err != nil {
					return "", tmpKeys, err
				}
			}
		}
		idsKey = filteredIdsKey
	}
	return idsKey, tmpKeys, nil
}

// intersectFilter adds commands to the query transacation which, when run, will create a
// temporary set which contains all the ids that fit the given filter criteria. Then it will
// intersect them with origKey and stores the result in destKey. The function will automatically
// delete any temporary sets created since, in this case, they are gauranteed to not be needed
// by any other transaction commands.
func (q *Query) intersectFilter(filter filter, origKey string, destKey string) error {
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

// intersectNumericFilter adds commands to the query transaction which, when run, will
// create a temporary set which contains all the ids of models which match the given
// numeric filter criteria, then intersect those ids with origKey and store the result
// in destKey.
func (q *Query) intersectNumericFilter(filter filter, origKey string, destKey string) error {
	fieldIndexKey, err := q.collection.spec.fieldIndexKey(filter.fieldSpec.name)
	if err != nil {
		return err
	}
	if filter.op == notEqualOp {
		// Special case for not equal. We need to use two separate commands
		valueExclusive := fmt.Sprintf("(%v", filter.value.Interface())
		filterKey := generateRandomKey("filter:" + fieldIndexKey)
		// ZADD all ids greater than filter.value
		q.tx.extractIdsFromFieldIndex(fieldIndexKey, filterKey, valueExclusive, "+inf")
		// ZADD all ids less than filter.value
		q.tx.extractIdsFromFieldIndex(fieldIndexKey, filterKey, "-inf", valueExclusive)
		// Intersect filterKey with origKey and store result in destKey
		q.tx.Command("ZINTERSTORE", redis.Args{destKey, 2, origKey, filterKey, "WEIGHTS", 1, 0}, nil)
		// Delete the temporary key
		q.tx.Command("DEL", redis.Args{filterKey}, nil)
	} else {
		var min, max interface{}
		switch filter.op {
		case equalOp:
			min, max = filter.value.Interface(), filter.value.Interface()
		case lessOp:
			min = "-inf"
			// use "(" for exclusive
			max = fmt.Sprintf("(%v", filter.value.Interface())
		case greaterOp:
			min = fmt.Sprintf("(%v", filter.value.Interface())
			max = "+inf"
		case lessOrEqualOp:
			min = "-inf"
			max = filter.value.Interface()
		case greaterOrEqualOp:
			min = filter.value.Interface()
			max = "+inf"
		}
		// Get all the ids that fit the filter criteria and store them in a temporary key caled filterKey
		filterKey := generateRandomKey("filter:" + fieldIndexKey)
		q.tx.extractIdsFromFieldIndex(fieldIndexKey, filterKey, min, max)
		// Intersect filterKey with origKey and store result in destKey
		q.tx.Command("ZINTERSTORE", redis.Args{destKey, 2, origKey, filterKey, "WEIGHTS", 1, 0}, nil)
		// Delete the temporary key
		q.tx.Command("DEL", redis.Args{filterKey}, nil)
	}
	return nil
}

// intersectBoolFilter adds commands to the query transaction which, when run, will
// create a temporary set which contains all the ids of models which match the given
// bool filter criteria, then intersect those ids with origKey and store the result
// in destKey.
func (q *Query) intersectBoolFilter(filter filter, origKey string, destKey string) error {
	fieldIndexKey, err := q.collection.spec.fieldIndexKey(filter.fieldSpec.name)
	if err != nil {
		return err
	}
	var min, max interface{}
	switch filter.op {
	case equalOp:
		if filter.value.Bool() {
			min, max = 1, 1
		} else {
			min, max = 0, 0
		}
	case lessOp:
		if filter.value.Bool() {
			// Only false is less than true
			min, max = 0, 0
		} else {
			// No models are less than false,
			// so we should eliminate all models
			min, max = -1, -1
		}
	case greaterOp:
		if filter.value.Bool() {
			// No models are greater than true,
			// so we should eliminate all models
			min, max = -1, -1
		} else {
			// Only true is greater than false
			min, max = 1, 1
		}
	case lessOrEqualOp:
		if filter.value.Bool() {
			// All models are <= true
			min, max = 0, 1
		} else {
			// Only false is <= false
			min, max = 0, 0
		}
	case greaterOrEqualOp:
		if filter.value.Bool() {
			// Only true is >= true
			min, max = 1, 1
		} else {
			// All models are >= false
			min, max = 0, 1
		}
	case notEqualOp:
		if filter.value.Bool() {
			min, max = 0, 0
		} else {
			min, max = 1, 1
		}
	}
	// Get all the ids that fit the filter criteria and store them in a temporary key caled filterKey
	filterKey := generateRandomKey("filter:" + fieldIndexKey)
	q.tx.extractIdsFromFieldIndex(fieldIndexKey, filterKey, min, max)
	// Intersect filterKey with origKey and store result in destKey
	q.tx.Command("ZINTERSTORE", redis.Args{destKey, 2, origKey, filterKey, "WEIGHTS", 1, 0}, nil)
	// Delete the temporary key
	q.tx.Command("DEL", redis.Args{filterKey}, nil)
	return nil
}

// intersectStringFilter adds commands to the query transaction which, when run, will
// create a temporary set which contains all the ids of models which match the given
// string filter criteria, then intersect those ids with origKey and store the result
// in destKey.
func (q *Query) intersectStringFilter(filter filter, origKey string, destKey string) error {
	fieldIndexKey, err := q.collection.spec.fieldIndexKey(filter.fieldSpec.name)
	if err != nil {
		return err
	}
	valString := filter.value.String()
	if filter.op == notEqualOp {
		// Special case for not equal. We need to use two separate commands
		filterKey := generateRandomKey("filter:" + fieldIndexKey)
		// ZADD all ids greater than filter.value
		min := "(" + valString + nullString + delString
		q.tx.extractIdsFromStringIndex(fieldIndexKey, filterKey, min, "+")
		// ZADD all ids less than filter.value
		max := "(" + valString
		q.tx.extractIdsFromStringIndex(fieldIndexKey, filterKey, "-", max)
		// Intersect filterKey with origKey and store result in destKey
		q.tx.Command("ZINTERSTORE", redis.Args{destKey, 2, origKey, filterKey, "WEIGHTS", 1, 0}, nil)
		// Delete the temporary key
		q.tx.Command("DEL", redis.Args{filterKey}, nil)
	} else {
		var min, max string
		switch filter.op {
		case equalOp:
			min = "[" + valString
			max = "(" + valString + nullString + delString
		case lessOp:
			min = "-"
			max = "(" + valString
		case greaterOp:
			min = "(" + valString + nullString + delString
			max = "+"
		case lessOrEqualOp:
			min = "-"
			max = "(" + valString + nullString + delString
		case greaterOrEqualOp:
			min = "[" + valString
			max = "+"
		}
		// Get all the ids that fit the filter criteria and store them in a temporary key caled filterKey
		filterKey := generateRandomKey("filter:" + fieldIndexKey)
		q.tx.extractIdsFromStringIndex(fieldIndexKey, filterKey, min, max)
		// Intersect filterKey with origKey and store result in destKey
		q.tx.Command("ZINTERSTORE", redis.Args{destKey, 2, origKey, filterKey, "WEIGHTS", 1, 0}, nil)
		// Delete the temporary key
		q.tx.Command("DEL", redis.Args{filterKey}, nil)
	}
	return nil
}

// fieldNames parses the includes and excludes properties to return a list of
// field names which should be included in all find operations. If there are no
// includes or excludes, it returns all the field names.
func (q *Query) fieldNames() []string {
	switch {
	case q.hasIncludes():
		return q.includes
	case q.hasExcludes():
		results := q.collection.spec.fieldNames()
		for _, name := range q.excludes {
			results = removeElementFromStringSlice(results, name)
		}
		return results
	default:
		return q.collection.spec.fieldNames()
	}
}

// redisFieldNames parses the includes and excludes properties to return a list of
// redis names for each field which should be included in all find operations. If
// there are no includes or excludes, it returns the redis names for all fields.
func (q *Query) redisFieldNames() []string {
	fieldNames := q.fieldNames()
	redisNames := []string{}
	for _, fieldName := range fieldNames {
		redisNames = append(redisNames, q.collection.spec.fieldsByName[fieldName].redisName)
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

// generateRandomKey generates a random string that is more or less
// garunteed to be unique and then prepends the given prefix. It is
// used to generate keys for temporary sorted sets in queries.
func generateRandomKey(prefix string) string {
	return prefix + ":" + generateRandomId()
}
