// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File query.go contains code related to the query abstraction.

package zoom

import (
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// Query represents a query which will retrieve some models from
// the database. A Query may consist of one or more query modifiers
// and can be run in several different ways with different query
// finishers.
type Query struct {
	modelSpec modelSpec
	trans     *transaction
	includes  []string
	excludes  []string
	order     order
	limit     uint
	offset    uint
	filters   []filter
	idData    []string
	err       error
}

type order struct {
	fieldName string
	redisName string
	orderType orderType
	indexed   bool
	indexType indexType
}

type orderType int

const (
	ascending = iota
	descending
)

type filter struct {
	fieldName   string
	redisName   string
	filterType  filterType
	filterValue reflect.Value
	indexType   indexType
	byId        bool
}

type filterType int

const (
	equal = iota
	notEqual
	greater
	less
	greaterOrEqual
	lessOrEqual
)

var filterSymbols = map[string]filterType{
	"=":  equal,
	"!=": notEqual,
	">":  greater,
	"<":  less,
	">=": greaterOrEqual,
	"<=": lessOrEqual,
}

// used as a prefix for alpha index tricks this is a string which equals ASCII
// DEL
var delString string = string([]byte{byte(127)})

// NewQuery is used to construct a query. modelName should be the name of a
// registered model. The query returned can be chained together with one or more
// query modifiers, and then executed using the Run, Scan, Count, or IdsOnly
// methods. If no query modifiers are used, running the query will return all
// models that match the type corresponding to modelName in uspecified order.
// NewQuery will set an error on the query if modelName is not the name of some
// registered model type. The error, same as any other error that occurs during
// the lifetime of the query, is not returned until the Query is executed. When
// the query is executed the first error that occured during the lifetime of the
// query object (if any) will be returned.
func NewQuery(modelName string) *Query {
	q := &Query{}
	spec, found := modelSpecs[modelName]
	if !found {
		q.setErrorIfNone(NewModelNameNotRegisteredError(modelName))
	} else {
		q.modelSpec = spec
	}
	return q
}

func (q *Query) setErrorIfNone(e error) {
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
		q.setErrorIfNone(errors.New("zoom: error in Query.Order: previous order already specified. Only one order per query is allowed."))
	}
	var ot orderType
	if strings.HasPrefix(fieldName, "-") {
		ot = descending
		// remove the "-" prefix
		fieldName = fieldName[1:]
	} else {
		ot = ascending
	}
	if _, found := q.modelSpec.field(fieldName); found {
		indexType, found := q.modelSpec.indexTypeForField(fieldName)
		if !found {
			// the field was not indexed
			// TODO: add support for ordering unindexed fields in some cases?
			err := fmt.Errorf("zoom: error in Query.Order: field %s in type %s is not indexed. Can only order by indexed fields", fieldName, q.modelSpec.modelType.String())
			q.setErrorIfNone(err)
		}
		redisName, _ := q.modelSpec.redisNameForFieldName(fieldName)
		q.order = order{
			fieldName: fieldName,
			redisName: redisName,
			orderType: ot,
			indexType: indexType,
			indexed:   true,
		}
	} else {
		// fieldName was invalid
		err := fmt.Errorf("zoom: error in Query.Order: could not find field %s in type %s", fieldName, q.modelSpec.modelType.String())
		q.setErrorIfNone(err)
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
		q.setErrorIfNone(errors.New("zoom: cannot use both Include and Exclude modifiers on a query"))
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
		q.setErrorIfNone(errors.New("zoom: cannot use both Include and Exclude modifiers on a query"))
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
		q.setErrorIfNone(err)
		return q
	}
	if fieldName == "Id" {
		// special case for Id
		return q.filterById(operator, value)
	}
	f := filter{
		fieldName: fieldName,
	}
	// get the filterType based on the operator
	if typ, found := filterSymbols[operator]; !found {
		q.setErrorIfNone(errors.New("zoom: invalid operator in fieldStr. should be one of =, !=, >, <, >=, or <=."))
		return q
	} else {
		f.filterType = typ
	}
	// get the redisName based on the fieldName
	if redisName, found := q.modelSpec.redisNameForFieldName(fieldName); !found {
		err := fmt.Errorf("zoom: invalid fieldName in filterString.\nType %s has no field %s", q.modelSpec.modelType.String(), fieldName)
		q.setErrorIfNone(err)
		return q
	} else {
		f.redisName = redisName
	}
	// get the indexType based on the fieldName
	if indexType, found := q.modelSpec.indexTypeForField(fieldName); !found {
		err := fmt.Errorf("zoom: filters are only allowed on indexed fields.\n%s.%s is not indexed.", q.modelSpec.modelType.String(), fieldName)
		q.setErrorIfNone(err)
		return q
	} else {
		f.indexType = indexType
	}
	// get type of the field and make sure it matches type of value arg
	// Here we iterate through pointer inderections. This is so you can
	// just pass in a primative instead of a pointer to a primative for
	// filtering on fields which have pointer values.
	structField, _ := q.modelSpec.field(fieldName)
	fieldType := structField.Type
	valueType := reflect.TypeOf(value)
	valueVal := reflect.ValueOf(value)
	for valueType.Kind() == reflect.Ptr {
		valueType = valueType.Elem()
		valueVal = valueVal.Elem()
		if !valueVal.IsValid() {
			q.setErrorIfNone(errors.New("zoom: invalid value arg for Filter. Is it a nil pointer?"))
			return q
		}
	}
	if valueType != fieldType {
		err := fmt.Errorf("zoom: invalid value arg for Filter. Parsed type of value (%s) does not match type of field (%s).", valueType.String(), fieldType.String())
		q.setErrorIfNone(err)
		return q
	} else {
		f.filterValue = valueVal
	}
	q.filters = append(q.filters, f)
	return q
}

func splitFilterString(filterString string) (fieldName string, operator string, err error) {
	split := strings.Split(filterString, " ")
	if len(split) != 2 {
		return "", "", errors.New("zoom: too many spaces in fieldStr argument. should be a field name, a space, and an operator.")
	}
	return split[0], split[1], nil
}

func (q *Query) filterById(operator string, value interface{}) *Query {
	if operator != "=" {
		q.setErrorIfNone(errors.New("zoom: only the = operator can be used with Filter on Id field."))
		return q
	}
	idVal := reflect.ValueOf(value)
	if idVal.Kind() != reflect.String {
		err := fmt.Errorf("zoom: for a Filter on Id field, value must be a string type. Was type %s", idVal.Kind().String())
		q.setErrorIfNone(err)
		return q
	}
	f := filter{
		fieldName:   "Id",
		redisName:   "Id",
		filterType:  equal,
		filterValue: idVal,
		byId:        true,
	}
	q.filters = append(q.filters, f)
	return q
}

// Run executes the query and returns the results in the form of an interface.
// The true type of the return value will be a slice of pointers to some
// regestired model type. If you need a type-safe way to run queries, look at
// the Scan method. Run will also return the first error that occured during the
// lifetime of the query object (if any). Otherwise, the second return value
// will be nil.
func (q *Query) Run() (interface{}, error) {
	if q.err != nil {
		return nil, q.err
	}
	q.trans = newTransaction()
	if err := q.sendIdData(); err != nil {
		return nil, err
	}

	// create a slice in which to store results using reflection the
	// type of the slice whill match the type of the model being queried
	resultsVal := reflect.New(reflect.SliceOf(q.modelSpec.modelType))
	resultsVal.Elem().Set(reflect.MakeSlice(reflect.SliceOf(q.modelSpec.modelType), 0, 0))

	if err := q.executeAndScan(resultsVal); err != nil {
		return resultsVal.Elem().Interface(), err
	} else {
		return resultsVal.Elem().Interface(), nil
	}
}

// Scan (like Run) executes the query but instead of returning the results it
// attempts to scan the results into in. The type of in should be a pointer to a
// slice of pointers to a registered model type.  Run will return the first
// error that occured during the lifetime of the query object (if any), or will
// return an error if you provided an interface with an invalid type. Otherwise,
// the return value will be nil.
func (q *Query) Scan(in interface{}) error {
	if q.err != nil {
		return q.err
	}
	q.trans = newTransaction()

	// make sure we are dealing with the right type
	typ := reflect.TypeOf(in).Elem()
	if !(typ.Kind() == reflect.Slice) {
		return fmt.Errorf("zoom: Query.Scan requires a pointer to a slice or array as an argument. Got: %T", in)
	}
	elemType := typ.Elem()
	if !typeIsPointerToStruct(elemType) {
		return fmt.Errorf("zoom: Query.Scan requires a pointer to a slice of pointers to model structs. Got: %T", in)
	}
	if elemType != q.modelSpec.modelType {
		return fmt.Errorf("zoom: argument for Query.Scan did not match the type corresponding to the model name given in the NewQuery constructor.\nExpected %T but got %T", reflect.SliceOf(q.modelSpec.modelType), in)
	}

	if err := q.sendIdData(); err != nil {
		return err
	}

	resultsVal := reflect.ValueOf(in)
	resultsVal.Elem().Set(reflect.MakeSlice(reflect.SliceOf(q.modelSpec.modelType), 0, 0))

	return q.executeAndScan(resultsVal)
}

// Count counts the number of models that would be returned by the query without
// actually retreiving the models themselves. Count will also return the first
// error that occured during the lifetime of the query object (if any).
// Otherwise, the second return value will be nil.
func (q *Query) Count() (int, error) {
	if len(q.filters) != 0 {
		if ids, err := q.IdsOnly(); err != nil {
			return 0, err
		} else {
			return len(ids), nil
		}
	}
	if q.err != nil {
		return 0, q.err
	}

	conn := GetConn()
	defer conn.Close()

	args := redis.Args{}
	var command string
	if q.order.fieldName == "" {
		// without ordering
		if q.offset != 0 {
			return 0, errors.New("zoom: offset cannot be applied to queries without an order.")
		}
		command = "SCARD"
		indexKey := q.modelSpec.modelName + ":all"
		args = args.Add(indexKey)
		count, err := redis.Int(conn.Do("SCARD", args...))
		if err != nil {
			return 0, err
		}
		if q.limit == 0 {
			// limit of 0 is the same as unlimited
			return count, nil
		} else {
			limitInt := int(q.limit)
			if count > limitInt {
				return limitInt, nil
			} else {
				return count, nil
			}
		}
	} else {
		// with ordering
		// this is a little more complicated
		command = "ZCARD"
		indexKey := q.modelSpec.modelName + ":" + q.order.redisName
		args = args.Add(indexKey)
		count, err := redis.Int(conn.Do(command, args...))
		if err != nil {
			return 0, err
		}
		if q.limit == 0 && q.offset == 0 {
			// simple case (no limit, no offset)
			return count, nil
		} else {
			// we need to take limit and offset into account
			// in order to return the correct number of models which
			// would have been returned by running the query
			if q.offset > uint(count) {
				// special case for offset > count
				return 0, nil
			} else if q.limit == 0 {
				// special case if limit = 0 (really means unlimited)
				return count - int(q.offset), nil
			} else {
				// holy type coercion, batman!
				// it's ugly but it works
				return int(math.Min(float64(count-int(q.offset)), float64(q.limit))), nil
			}
		}
	}
}

// IdsOnly returns only the ids of the models without actually retreiving the
// models themselves. IdsOnly will also return the first error that occured
// during the lifetime of the query object (if any). Otherwise, the second
// return value will be nil.
func (q *Query) IdsOnly() ([]string, error) {
	if q.err != nil {
		return nil, q.err
	}
	q.trans = newTransaction()
	if err := q.sendIdData(); err != nil {
		return nil, err
	}

	// wait for all the id data dependencies to be resolved,
	// then simply set results to be ids
	results := make([]string, 0)
	q.trans.doWhenDataReady(q.idData, func() error {
		if ids, err := q.intersectAllIds(); err != nil {
			return err
		} else {
			results = ids
			return nil
		}
	})
	if err := q.trans.exec(); err != nil {
		return results, err
	} else {
		return results, nil
	}
}

// sendIdData adds commands to the query transaction which will gather all the
// ids of the models which should be returned by the query and send them as
// shared transaction data. The sets of ids will be given string identifiers,
// which will be added to the idData slice of the query. When the query
// transaction is eventually executed, each of the keys in idData should be
// considered data dependencies.
func (q *Query) sendIdData() error {
	if len(q.filters) == 0 {
		if cmd, args, err := q.getAllModelsArgs(true); err != nil {
			return err
		} else {
			idsDataKey := "modelIds"
			q.idData = append(q.idData, idsDataKey)
			if q.order.fieldName != "" && q.order.indexType == indexAlpha {
				// special case for parsing ids from the redis response
				q.trans.command(cmd, args, newSendAlphaIdsHandler(q.trans, idsDataKey, false))
			} else {
				// send the database response directly
				q.trans.command(cmd, args, newSendDataHandler(q.trans, idsDataKey))
			}
			return nil
		}
	} else {
		// with filters, we need to iterate through each filter and get the ids
		primaryCovered := false
		for i, f := range q.filters {
			filterIdsKey := "filter" + strconv.Itoa(i)
			if f.fieldName == q.order.fieldName {
				filterIdsKey = "primaryIds"
				primaryCovered = true
			}
			q.idData = append(q.idData, filterIdsKey)
			q.sendIdDataForFilter(f, filterIdsKey)
		}
		if !primaryCovered {
			// no filter had the same field name as the order, so we need to add a
			// command to get the ordered ids and use them as a basis for ordering
			// all the others.
			if cmd, args, err := q.getAllModelsArgs(false); err != nil {
				return err
			} else {
				orderedIdsKey := "primaryIds"
				q.idData = append(q.idData, orderedIdsKey)
				if q.order.fieldName != "" && q.order.indexType == indexAlpha {
					// special case for parsing ids from the redis response
					q.trans.command(cmd, args, newSendAlphaIdsHandler(q.trans, orderedIdsKey, false))
				} else {
					// send the database response directly
					q.trans.command(cmd, args, newSendDataHandler(q.trans, orderedIdsKey))
				}
			}
		}
	}
	return nil
}

// returns a function which, when run, extracts ids from alpha index values and then sends the ids as transaction data
func newSendAlphaIdsHandler(t *transaction, key string, reverse bool) func(interface{}) error {
	return func(reply interface{}) error {
		if ids, err := redis.Strings(reply, nil); err != nil {
			return err
		} else {
			for i, valueAndId := range ids {
				ids[i] = extractModelIdFromAlphaIndexValue(valueAndId)
			}
			if reverse {
				// TODO: if redis adds a ZREVRANGEBYLEX, remove this manual reverse and use that instead
				for i, j := 0, len(ids)-1; i <= j; i, j = i+1, j-1 {
					ids[i], ids[j] = ids[j], ids[i]
				}
			}
			t.sendData(key, ids)
			return nil
		}
	}
}

// Alpha indexes are stored as "<fieldValue> <modelId>", so we need to
// extract the modelId. While fieldValue may have a space, modelId CANNOT
// have a space in it, so we can simply take the part of the stored value
// after the last space.
func extractModelIdFromAlphaIndexValue(valueAndId string) string {
	slices := strings.Split(valueAndId, " ")
	return slices[len(slices)-1]
}

// NOTE: sliceVal should be the value of a pointer to a slice of pointer to models.
// It's type should be *[]*<T>, where <T> is some type which satisfies the Model
// interface. The type *[]*Model is not equivalent and will not work.
func (q *Query) executeAndScan(sliceVal reflect.Value) error {
	// Check to make sure include contained all valid fields
	if len(q.includes) > 0 {
		fieldNames := q.modelSpec.fieldNames()
		for _, inc := range q.includes {
			if !stringSliceContains(inc, fieldNames) {
				return fmt.Errorf("zoom: Model of type %s does not have field called %s", q.modelSpec.modelName, inc)
			}
		}
	}

	// wait for all the id data dependencies and then scan them
	// into models
	q.trans.doWhenDataReady(q.idData, func() error {
		if ids, err := q.intersectAllIds(); err != nil {
			return err
		} else {
			return q.scanModelsByIds(ids, sliceVal)
		}
	})
	return q.trans.exec()
}

// NOTE: should be placed inside a doWhenDataReady function, otherwise
// it won't work
func (q *Query) intersectAllIds() ([]string, error) {
	var allModelIds []string
	for i, idDataKey := range q.idData {
		theseIds, err := convertDataToStrings(q.trans.data[idDataKey])
		if err != nil {
			return nil, err
		}
		switch i {
		case 0:
			// on the first iteration just set allModelIds
			allModelIds = theseIds
		default:
			// on subsequent iterations, do an ordered intersect with allModelIds
			if idDataKey == "primaryIds" {
				// indicates we should order with respect to these ids
				allModelIds = orderedIntersectStrings(theseIds, allModelIds)
			} else {
				allModelIds = orderedIntersectStrings(allModelIds, theseIds)
			}
		}
	}
	if len(q.filters) > 0 {
		allModelIds = applyLimitOffset(allModelIds, q.limit, q.offset)
	}
	return allModelIds, nil
}

func convertDataToStrings(data interface{}) ([]string, error) {
	// first try direct type assertion
	if strings, ok := data.([]string); ok {
		return strings, nil
	}
	// then try using redigo to convert
	if strings, err := redis.Strings(data, nil); err != nil {
		return nil, err
	} else {
		return strings, nil
	}
}

// NOTE: sliceVal should have an underlying type of pointer to a slice of
// pointers to model structs
func (q *Query) scanModelsByIds(ids []string, sliceVal reflect.Value) error {
	for _, id := range ids {
		mr, err := newModelRefFromName(q.modelSpec.modelName)
		if err != nil {
			return err
		}
		mr.model.SetId(id)
		if err := q.trans.findModel(mr, q.getIncludes()); err != nil {
			return err
		}
		sliceVal.Elem().Set(reflect.Append(sliceVal.Elem(), mr.modelVal()))
	}
	return nil
}

// NOTE: only use this method for queries without any filters.
func (q *Query) getAllModelsArgs(applyLimitOffset bool) (string, redis.Args, error) {
	args := redis.Args{}
	var command string
	if q.order.fieldName == "" {
		if q.offset != 0 {
			return "", nil, errors.New("zoom: offset cannot be applied to queries without an order.")
		}
		indexKey := q.modelSpec.modelName + ":all"
		args = args.Add(indexKey)
		if q.limit == 0 || !applyLimitOffset {
			command = "SMEMBERS"
		} else {
			command = "SRANDMEMBER"
			args = args.Add(q.limit)
		}
	} else {
		if q.order.orderType == ascending {
			command = "ZRANGE"
		} else if q.order.orderType == descending {
			command = "ZREVRANGE"
		}
		indexKey := q.modelSpec.modelName + ":" + q.order.redisName
		args = args.Add(indexKey)
		if applyLimitOffset {
			start, stop := q.getStartStop()
			args = args.Add(start).Add(stop)
		} else {
			args = args.Add(0).Add(-1)
		}
	}
	return command, args, nil
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

func (q *Query) sendIdDataForFilter(f filter, dataKey string) error {
	// special case for id filters
	if f.byId {
		id := f.filterValue.String()
		q.trans.sendData(dataKey, []string{id})
	} else {
		setKey := q.modelSpec.modelName + ":" + f.redisName
		reverse := q.order.orderType == descending && q.order.fieldName == f.fieldName
		switch f.indexType {

		case indexNumeric:
			args := redis.Args{}.Add(setKey)
			switch f.filterType {
			case equal, less, greater, lessOrEqual, greaterOrEqual:
				min, max := getMinMaxForNumericFilter(f)
				var command string
				if !reverse {
					command = "ZRANGEBYSCORE"
					args = args.Add(min).Add(max)
				} else {
					command = "ZREVRANGEBYSCORE"
					args = args.Add(max).Add(min)
				}
				q.trans.command(command, args, newSendDataHandler(q.trans, dataKey))
			case notEqual:
				// special case for not equals
				// split into two different queries (less and greater) and
				// use union to combine the results
				max := fmt.Sprintf("(%v", f.filterValue.Interface())
				lessArgs := args.Add("-inf").Add(max)
				lessIdsKey := dataKey + "lessIds"
				q.trans.command("ZRANGEBYSCORE", lessArgs, newSendDataHandler(q.trans, lessIdsKey))
				min := fmt.Sprintf("(%v", f.filterValue.Interface())
				greaterIdsKey := dataKey + "greaterIds"
				greaterArgs := args.Add(min).Add("+inf")
				q.trans.command("ZRANGEBYSCORE", greaterArgs, newSendDataHandler(q.trans, greaterIdsKey))

				// when both lessIds and greaterIds are ready, combine them into a single set of ids
				q.trans.doWhenDataReady([]string{lessIdsKey, greaterIdsKey}, func() error {
					lessIds, err := convertDataToStrings(q.trans.data[lessIdsKey])
					if err != nil {
						return err
					}
					greaterIds, err := convertDataToStrings(q.trans.data[greaterIdsKey])
					if err != nil {
						return err
					}
					allFilterIds := make([]string, 0)
					if !reverse {
						allFilterIds = append(lessIds, greaterIds...)
					} else {
						for i, j := 0, len(lessIds)-1; i <= j; i, j = i+1, j-1 {
							lessIds[i], lessIds[j] = lessIds[j], lessIds[i]
						}
						for i, j := 0, len(greaterIds)-1; i <= j; i, j = i+1, j-1 {
							greaterIds[i], greaterIds[j] = greaterIds[j], greaterIds[i]
						}
						allFilterIds = append(greaterIds, lessIds...)
					}
					q.trans.sendData(dataKey, allFilterIds)
					return nil
				})
			}

		case indexBoolean:
			args := redis.Args{}.Add(setKey)
			var min, max interface{}
			switch f.filterType {
			case equal:
				if f.filterValue.Bool() == true {
					// use 1 for true
					min, max = 1, 1
				} else {
					// use 0 for false
					min, max = 0, 0
				}
			case less:
				if f.filterValue.Bool() == true {
					// false is less than true
					// 0 < 1
					min, max = 0, 0
				} else {
					// can't be less than false (0)
					q.trans.sendData(dataKey, []string{})
					return nil
				}
			case greater:
				if f.filterValue.Bool() == true {
					// can't be greater than true (1)
					q.trans.sendData(dataKey, []string{})
					return nil
				} else {
					// true is greater than false
					// 1 > 0
					min, max = 1, 1
				}
			case lessOrEqual:
				if f.filterValue.Bool() == true {
					// true and false are <= true
					// 1 <= 1 and 0 <= 1
					min = 0
					max = 1
				} else {
					// false <= false
					// 0 <= 0
					min, max = 0, 0
				}
			case greaterOrEqual:
				if f.filterValue.Bool() == true {
					// true >= true
					// 1 >= 1
					min, max = 1, 1
				} else {
					// false and true are >= false
					// 0 >= 0 and 1 >= 0
					min = 0
					max = 1
				}
			case notEqual:
				if f.filterValue.Bool() == true {
					// not true means false
					// false == 0
					min, max = 0, 0
				} else {
					// not false means true
					// true == 1
					min, max = 1, 1
				}
			default:
				return fmt.Errorf("zoom: Filter operator out of range. Got: %d", f.filterType)
			}
			// execute command to get the ids
			// TODO: try and do this inside of a transaction
			var command string
			if !reverse {
				command = "ZRANGEBYSCORE"
				args = args.Add(min).Add(max)
			} else {
				command = "ZREVRANGEBYSCORE"
				args = args.Add(max).Add(min)
			}
			q.trans.command(command, args, newSendDataHandler(q.trans, dataKey))

		case indexAlpha:
			args := redis.Args{}.Add(setKey)
			switch f.filterType {
			case equal, less, greater, lessOrEqual, greaterOrEqual:
				var min, max string
				valString := f.filterValue.String()
				switch f.filterType {
				case equal:
					min = "(" + valString
					max = "(" + valString + delString
				case less:
					min = "-"
					max = "(" + valString
				case greater:
					min = "(" + valString + delString
					max = "+"
				case lessOrEqual:
					min = "-"
					max = "(" + valString + delString
				case greaterOrEqual:
					min = "(" + valString
					max = "+"
				}
				args = args.Add(min).Add(max)
				q.trans.command("ZRANGEBYLEX", args, newSendAlphaIdsHandler(q.trans, dataKey, reverse))
			case notEqual:
				// special case for not equals
				// split into two different queries (less and greater) and
				// combine the results
				valString := f.filterValue.String()
				max := "(" + valString
				lessArgs := args.Add("-").Add(max)
				lessIdsKey := dataKey + "lessIds"
				q.trans.command("ZRANGEBYLEX", lessArgs, newSendAlphaIdsHandler(q.trans, lessIdsKey, false))
				min := "(" + valString + delString
				greaterIdsKey := dataKey + "greaterIds"
				greaterArgs := args.Add(min).Add("+")
				q.trans.command("ZRANGEBYLEX", greaterArgs, newSendAlphaIdsHandler(q.trans, greaterIdsKey, false))

				// when both lessIds and greaterIds are ready, combine them into a single set of ids
				q.trans.doWhenDataReady([]string{lessIdsKey, greaterIdsKey}, func() error {
					lessIds, err := convertDataToStrings(q.trans.data[lessIdsKey])
					if err != nil {
						return err
					}
					greaterIds, err := convertDataToStrings(q.trans.data[greaterIdsKey])
					if err != nil {
						return err
					}
					allFilterIds := make([]string, 0)
					if !reverse {
						allFilterIds = append(lessIds, greaterIds...)
					} else {
						for i, j := 0, len(lessIds)-1; i <= j; i, j = i+1, j-1 {
							lessIds[i], lessIds[j] = lessIds[j], lessIds[i]
						}
						for i, j := 0, len(greaterIds)-1; i <= j; i, j = i+1, j-1 {
							greaterIds[i], greaterIds[j] = greaterIds[j], greaterIds[i]
						}
						allFilterIds = append(greaterIds, lessIds...)
					}
					q.trans.sendData(dataKey, allFilterIds)
					return nil
				})
			}

		default:
			return fmt.Errorf("zoom: cannot use filters on unindexed field %s for model name %s.", f.fieldName, q.modelSpec.modelName)
		}
	}
	return nil
}

// do some math wrt limit and offset and return the results
func applyLimitOffset(slice []string, limit uint, offset uint) []string {
	start := offset
	var end uint
	if limit == 0 {
		end = uint(len(slice))
	} else {
		end = start + limit
	}
	if int(start) > len(slice) {
		return []string{}
	} else if int(end) > len(slice) {
		return slice[start:]
	} else {
		return slice[start:end]
	}
	return slice
}

func getMinMaxForNumericFilter(f filter) (min interface{}, max interface{}) {
	switch f.filterType {
	case equal:
		min, max = f.filterValue.Interface(), f.filterValue.Interface()
	case less:
		min = "-inf"
		// use "(" for exclusive
		max = fmt.Sprintf("(%v", f.filterValue.Interface())
	case greater:
		// use "(" for exclusive
		min = fmt.Sprintf("(%v", f.filterValue.Interface())
		max = "+inf"
	case lessOrEqual:
		min = "-inf"
		max = f.filterValue.Interface()
	case greaterOrEqual:
		min = f.filterValue.Interface()
		max = "+inf"
	}
	return min, max
}

// string returns a string representation of the filterType
func (ft filterType) string() string {
	switch ft {
	case equal:
		return "="
	case notEqual:
		return "!="
	case greater:
		return ">"
	case less:
		return "<"
	case greaterOrEqual:
		return ">="
	case lessOrEqual:
		return "<="
	}
	return ""
}

// string returns a string representation of the filter
func (f filter) string() string {
	return fmt.Sprintf("(filter %s %s %v)", f.fieldName, f.filterType.string(), f.filterValue.Interface())
}

// string returns a string representation of the order
func (o order) string() string {
	if o.fieldName == "" {
		return ""
	}
	switch o.orderType {
	case ascending:
		return fmt.Sprintf("(order %s)", o.fieldName)
	case descending:
		return fmt.Sprintf("(order -%s)", o.fieldName)
	}
	return ""
}

// String returns a string representation of the query and its modifiers
func (q *Query) String() string {
	modelName := q.modelSpec.modelName
	filters := ""
	for _, f := range q.filters {
		filters += f.string() + " "
	}
	order := q.order.string()
	limit := ""
	offset := ""
	if q.limit != 0 {
		limit = fmt.Sprintf("(limit %v)", q.limit)
	}
	if q.offset != 0 {
		offset = fmt.Sprintf("(offset %v)", q.offset)
	}
	includes := ""
	if len(q.includes) > 0 {
		fields := "["
		for i, in := range q.includes {
			fields += in
			if i != len(q.includes)-1 {
				fields += ", "
			}
		}
		fields += "]"
		includes = fmt.Sprintf("(include %s)", fields)
	}
	excludes := ""
	if len(q.excludes) > 0 {
		fields := "["
		for i, ex := range q.excludes {
			fields += ex
			if i != len(q.excludes)-1 {
				fields += ", "
			}
		}
		fields += "]"
		excludes = fmt.Sprintf("(exclude %s)", fields)
	}
	return fmt.Sprintf("%s: %s%s %s %s %s%s", modelName, filters, order, limit, offset, includes, excludes)
}
