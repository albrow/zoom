// Copyright 2014 Alex Browne.  All rights reserved.
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
	getIdsPhases, allIds, err := q.getAllIds()
	if err != nil {
		return nil, err
	}

	// create a slice in which to store results using reflection the
	// type of the slice whill match the type of the model being queried
	resultsVal := reflect.New(reflect.SliceOf(q.modelSpec.modelType))
	resultsVal.Elem().Set(reflect.MakeSlice(reflect.SliceOf(q.modelSpec.modelType), 0, 0))

	if err := q.executeAndScan(resultsVal, getIdsPhases, allIds); err != nil {
		return resultsVal.Elem().Interface(), err
	} else {
		return resultsVal.Elem().Interface(), nil
	}
}

// RunOne is exactly like Run, returns only the first model that fits the query
// criteria, or if no models fit the critera, returns an error. If you need to do this
// in a type-safe way, look at the ScanOne method.
func (q *Query) RunOne() (interface{}, error) {
	// optimize the query by limiting number of models to one
	oldLimit := q.limit
	q.limit = 1

	result, err := q.Run()
	if err != nil {
		return nil, err
	}

	// set the limit back to the old limit in case the query will be run again
	q.limit = oldLimit

	// get the first item, if any
	resultVal := reflect.ValueOf(result)
	if resultVal.Len() == 0 {
		return nil, NewModelNotFoundError()
	}
	first := resultVal.Index(0)
	return first.Interface(), nil
}

// Scan (like Run) executes the query but instead of returning the results it
// attempts to scan the results into in. The type of in should be a pointer to a
// slice of pointers to a registered model type.  Scan will return the first
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

	getIdsPhase, allIds, err := q.getAllIds()
	if err != nil {
		return err
	}

	resultsVal := reflect.ValueOf(in)
	resultsVal.Elem().Set(reflect.MakeSlice(reflect.SliceOf(q.modelSpec.modelType), 0, 0))

	return q.executeAndScan(resultsVal, getIdsPhase, allIds)
}

// ScanOne is exactly like Scan but scans only the first model that fits the
// query criteria. If no model fits the criteria, an error will be returned.
// The type of in should be a pointer to a slice of pointers to a registered
// model type.
func (q *Query) ScanOne(in interface{}) error {
	// make sure we are dealing with the right type
	typ := reflect.TypeOf(in)
	if !typeIsPointerToStruct(typ) {
		return fmt.Errorf("zoom: Query.Scan requires a pointer to a model. Got: %T", in)
	}
	if typ != q.modelSpec.modelType {
		return fmt.Errorf("zoom: argument for Query.Scan did not match the type corresponding to the model name given in the NewQuery constructor.\nExpected %T but got %T", reflect.SliceOf(q.modelSpec.modelType), in)
	}

	result, err := q.RunOne()
	if err != nil {
		return err
	}

	resultVal := reflect.ValueOf(result)
	reflect.ValueOf(in).Elem().Set(resultVal.Elem())
	return nil
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
	_, allIds, err := q.getAllIds()
	if err != nil {
		return nil, err
	}
	err = q.trans.exec()
	if err != nil {
		return nil, err
	}
	ids := allIds.ids
	if len(q.filters) > 0 {
		// If there were any filters, then we need to apply limit
		// and offset at this stage because we couldn't apply it
		// earlier
		ids = applyLimitOffset(allIds.ids, q.limit, q.offset)
	}
	return ids, nil
}

// getAllIds adds commands to the query transaction which will gather all the
// ids of the models which should be returned by the query and collects them into
// a slice of strings by doing ordered intersects. Doing so may require multiple
// phases in some instances. It returns the phases that were created and a pointer
// to a slice of the ids that will be filled when the phase is eventually executed.
func (q *Query) getAllIds() ([]*phase, *idSet, error) {
	// Create a slice of phases for getting the ids and
	// add the main phase
	getIdsPhases := make([]*phase, 1)
	var err error
	getIdsPhases[0], err = q.trans.addPhase("getIds", nil, nil)
	if err != nil {
		return nil, nil, err
	}
	allIds := newIdSet()
	if len(q.filters) == 0 {
		if cmd, args, err := q.getAllModelsArgs(true); err != nil {
			return getIdsPhases, allIds, err
		} else {
			if q.order.fieldName != "" && q.order.indexType == indexAlpha {
				getIdsPhases[0].addCommand(cmd, args, newScanAlphaIdsHandler(allIds, false))
			} else {
				getIdsPhases[0].addCommand(cmd, args, newScanIdsHandler(allIds))
			}
			return getIdsPhases, allIds, nil
		}
	} else {
		// with filters, we need to iterate through each filter and get the ids.
		// primaryPhase is the phase that gets the ids in the order specified by the query
		// if no order was specified, primaryPhase will be nil.
		var primaryPhase *phase
		for _, f := range q.filters {
			if f.fieldName == q.order.fieldName {
				// these are the primary ids
				modelAndFieldName := fmt.Sprintf("%s:%s", q.modelSpec.modelName, q.order.fieldName)
				primaryPhase, err = q.trans.addPhase(modelAndFieldName, nil, nil)
				if err != nil {
					return nil, nil, err
				}
				if err := getIdsPhases[0].addDependency(primaryPhase); err != nil {
					return nil, nil, err
				}
				getIdsPhases = append(getIdsPhases, primaryPhase)
				subPhases, err := q.addCommandForFilter(primaryPhase, f, allIds)
				if err != nil {
					return nil, nil, err
				}
				getIdsPhases = append(getIdsPhases, subPhases...)
			} else {
				subPhases, err := q.addCommandForFilter(getIdsPhases[0], f, allIds)
				if err != nil {
					return nil, nil, err
				}
				getIdsPhases = append(getIdsPhases, subPhases...)
			}
		}
		if primaryPhase == nil && q.order.fieldName != "" {
			// no filter had the same field name as the order, so we need to add a
			// command to get the ordered ids and use them as a basis for ordering
			// all the others.
			if cmd, args, err := q.getAllModelsArgs(false); err != nil {
				return getIdsPhases, allIds, err
			} else {
				// create the primaryPhase
				modelAndFieldName := fmt.Sprintf("%s:%s", q.modelSpec.modelName, q.order.fieldName)
				primaryPhase, err = q.trans.addPhase(modelAndFieldName, nil, nil)
				if err != nil {
					return nil, nil, err
				}
				if err := getIdsPhases[0].addDependency(primaryPhase); err != nil {
					return nil, nil, err
				}
				getIdsPhases = append(getIdsPhases, primaryPhase)

				// add the appropriate command to the primaryPhase
				if q.order.fieldName != "" && q.order.indexType == indexAlpha {
					primaryPhase.addCommand(cmd, args, newScanAlphaIdsHandler(allIds, false))
				} else {
					primaryPhase.addCommand(cmd, args, newScanIdsHandler(allIds))
				}
			}
		}
	}
	return getIdsPhases, allIds, nil
}

// newScanAlphaIdsHandler returns a function which, when run, extracts ids from alpha index
// values and intersects the ids with the existing ids in allIds, maintaining the order of
// the ids that are already there, if any.
func newScanAlphaIdsHandler(allIds *idSet, reverse bool) func(interface{}) error {
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
			allIds.intersect(ids)
			return nil
		}
	}
}

// newScanIdsHandler returns a function which, when run, converts a redis response to a slice
// of string ids and then  intersects the ids with the existing ids in allIds, maintaining
// the order of the ids that are already there, if any.
func newScanIdsHandler(allIds *idSet) func(interface{}) error {
	return func(reply interface{}) error {
		ids, err := redis.Strings(reply, nil)
		if err != nil {
			return err
		}
		allIds.intersect(ids)
		return nil
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

// NOTE: models should be the value of a pointer to a slice of pointer to models.
// It's type should be *[]*<T>, where <T> is some type which satisfies the Model
// interface. The type *[]*Model is not equivalent and will not work.
func (q *Query) executeAndScan(models reflect.Value, getIdsPhases []*phase, allIds *idSet) error {

	// Check to make sure include contained all valid fields
	if len(q.includes) > 0 {
		fieldNames := q.modelSpec.fieldNames()
		for _, inc := range q.includes {
			if !stringSliceContains(inc, fieldNames) {
				return fmt.Errorf("zoom: Model of type %s does not have field called %s", q.modelSpec.modelName, inc)
			}
		}
	}

	// the scanModelsPhase will execute after all the ids have been found and intersected.
	// it adds commands to find each model by its id and append the model to models
	scanModelsPhase, err := q.trans.addPhase("scanModels", q.newScanModelsByIdsPreHandler(allIds, models), nil)
	if err != nil {
		return err
	}
	for _, phase := range getIdsPhases {
		if err := scanModelsPhase.addDependency(phase); err != nil {
			return err
		}
	}

	return q.trans.exec()
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

func (q *Query) newScanModelsByIdsPreHandler(allIds *idSet, models reflect.Value) func(*phase) error {
	return func(p *phase) error {
		ids := allIds.ids
		if len(q.filters) > 0 {
			// If there were any filters, then we need to apply limit
			// and offset at this stage because we couldn't apply it
			// earlier
			ids = applyLimitOffset(allIds.ids, q.limit, q.offset)
		}
		for _, id := range ids {
			mr, err := newModelRefFromName(q.modelSpec.modelName)
			if err != nil {
				return err
			}
			mr.model.SetId(id)
			if err := p.scanModel(mr, q.getIncludes()); err != nil {
				return err
			}
			models.Elem().Set(reflect.Append(models.Elem(), mr.modelVal()))
		}
		return nil
	}
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

func (q *Query) addCommandForFilter(p *phase, f filter, allIds *idSet) ([]*phase, error) {
	subPhases := []*phase{}
	if f.byId {
		// special case for id filters
		id := f.filterValue.String()
		allIds.intersect([]string{id})
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
				p.addCommand(command, args, newScanIdsHandler(allIds))
			case notEqual:
				// TODO: make this dryer! (Currently repeated for alpha indexes)
				// special case for not equals:
				// split into two different commands (less and greater) and
				// use union to combine the results. We'll create a new phase
				// for this and add the current phase as a dependency to it

				// the id for this subphase will be identified by the modelName, fieldName, filter kind, and filter value
				subPhaseId := fmt.Sprintf("%s:%s:%s:%s", q.modelSpec.modelName, f.fieldName, f.filterType, f.filterValue.String())

				// lessIds and greaterIds will hold the ids from their respective commands
				lessIds := newIdSet()
				greaterIds := newIdSet()

				// add the new subPhase to the transaction
				var subPhase *phase
				if existingPhase, found := q.trans.phaseById(subPhaseId); found {
					subPhase = existingPhase
				} else {
					// If the phase doesn't already exist for this index, create a new one
					newPhase, err := q.trans.addPhase(subPhaseId, nil, newUnionIdsPostHandler(lessIds, greaterIds, allIds, reverse))
					if err != nil {
						return nil, err
					}
					subPhase = newPhase
				}

				// subPhase should execute after p, in case p is not the primary phase.
				// this way we know that the primary phase will always execute first and
				// determine the ordering for all other intersections.
				if err := subPhase.addDependency(p); err != nil {
					return nil, err
				}

				// set up each command, one for less than and one for greater than, and add
				// them to the subPhase
				max := fmt.Sprintf("(%v", f.filterValue.Interface())
				lessArgs := args.Add("-inf").Add(max)
				subPhase.addCommand("ZRANGEBYSCORE", lessArgs, newScanIdsHandler(lessIds))
				min := fmt.Sprintf("(%v", f.filterValue.Interface())
				greaterArgs := args.Add(min).Add("+inf")
				subPhase.addCommand("ZRANGEBYSCORE", greaterArgs, newScanIdsHandler(greaterIds))
				subPhases = append(subPhases, subPhase)
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
					// no models can meet this criteria, so
					// set ids to an empty slice
					allIds.intersect([]string{})
					return nil, nil
				}
			case greater:
				if f.filterValue.Bool() == true {
					// can't be greater than true (1)
					// no models can meet this criteria, so
					// set ids to an empty slice
					allIds.intersect([]string{})
					return nil, nil
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
				return nil, fmt.Errorf("zoom: Filter operator out of range. Got: %d", f.filterType)
			}
			// execute command to get the ids
			var command string
			if !reverse {
				command = "ZRANGEBYSCORE"
				args = args.Add(min).Add(max)
			} else {
				command = "ZREVRANGEBYSCORE"
				args = args.Add(max).Add(min)
			}
			p.addCommand(command, args, newScanIdsHandler(allIds))

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
				p.addCommand("ZRANGEBYLEX", args, newScanAlphaIdsHandler(allIds, reverse))
			case notEqual:
				// TODO: make this dryer! (Currently repeated for numeric indexes)
				// special case for not equals:
				// split into two different commands (less and greater) and
				// use union to combine the results. We'll create a new phase
				// for this and add the current phase as a dependency to it

				// the id for this subphase will be identified by the modelName, fieldName, filter kind, and filter value
				subPhaseId := fmt.Sprintf("%s:%s:%s:%s", q.modelSpec.modelName, f.fieldName, f.filterType, f.filterValue.String())

				// lessIds and greaterIds will hold the ids from their respective commands
				lessIds := newIdSet()
				greaterIds := newIdSet()

				// add the new subPhase to the transaction
				var subPhase *phase
				if existingPhase, found := q.trans.phaseById(subPhaseId); found {
					subPhase = existingPhase
				} else {
					// If the phase doesn't already exist for this index, create a new one
					newPhase, err := q.trans.addPhase(subPhaseId, nil, newUnionIdsPostHandler(lessIds, greaterIds, allIds, reverse))
					if err != nil {
						return nil, err
					}
					subPhase = newPhase
				}

				// subPhase should execute after p, in case p is not the primary phase.
				// this way we know that the primary phase will always execute first and
				// determine the ordering for all other intersections.
				if err := subPhase.addDependency(p); err != nil {
					return nil, err
				}

				// set up each command, one for less than and one for greater than, and add
				// them to the subPhase
				valString := f.filterValue.String()
				max := "(" + valString
				lessArgs := args.Add("-").Add(max)
				subPhase.addCommand("ZRANGEBYLEX", lessArgs, newScanAlphaIdsHandler(lessIds, false))
				min := "(" + valString + delString
				greaterArgs := args.Add(min).Add("+")
				subPhase.addCommand("ZRANGEBYLEX", greaterArgs, newScanAlphaIdsHandler(greaterIds, false))
				subPhases = append(subPhases, subPhase)
			}

		default:
			return nil, fmt.Errorf("zoom: cannot use filters on unindexed field %s for model name %s.", f.fieldName, q.modelSpec.modelName)
		}
	}
	return subPhases, nil
}

// newUnionIdsPostHandler returns a function which combines lessIds and
// greaterIds into a slice of ids (bothIds), then does an ordered intersect
// with the less than and greater than ids and all the ids for all other
// filters (allIds).
func newUnionIdsPostHandler(lessIds *idSet, greaterIds *idSet, allIds *idSet, reverse bool) func(*phase) error {
	bothIds := []string{}
	return func(p *phase) error {
		if !reverse {
			lessIds.union(greaterIds)
			bothIds = lessIds.ids
		} else {
			// TODO: can we avoid reversing here and do it in the command instead?
			// reverse each slice and then append them
			lessIds.reverse()
			greaterIds.reverse()
			greaterIds.union(lessIds)
			bothIds = greaterIds.ids
		}
		// add bothIds to ids using an ordered intersect
		allIds.intersect(bothIds)
		return nil
	}
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
