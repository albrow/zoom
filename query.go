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
	"strings"
)

type RunScanner interface {
	Run() (interface{}, error)
	Scan(interface{}) error
}

type Query struct {
	modelSpec modelSpec
	includes  []string
	excludes  []string
	order     order
	limit     uint
	offset    uint
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

// Order specifies a field by which to sort the records and the order in which
// records should be sorted. fieldName should be a field in the struct type specified by
// the modelName argument in the query constructor. By default, the records are sorted
// by ascending order. To sort by descending order, put a negative sign before
// the field name. Zoom can only sort by fields which have been indexed, i.e. those which
// have the `zoom:"index"` struct tag. However, in the future this may change.
// Only one order may be specified per query. However in the future, secondary orders may be
// allowed, and will take effect when two or more models have the same value for the primary
// order field. Order will set an error on the query if the fieldName is invalid, if another
// order has already been applied to the query, or if the fieldName specified does not correspond
// to an indexed field.
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
			msg := fmt.Sprintf("zoom: error in Query.Order: field %s in type %s is not indexed. Can only order by indexed fields", fieldName, q.modelSpec.modelType.String())
			q.setErrorIfNone(errors.New(msg))
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
		msg := fmt.Sprintf("zoom: error in Query.Order: could not find field %s in type %s", fieldName, q.modelSpec.modelType.String())
		q.setErrorIfNone(errors.New(msg))
	}

	return q
}

// Limit specifies an upper limit on the number of records to return.
// If amount is 0, no limit will be applied.
func (q *Query) Limit(amount uint) *Query {
	q.limit = amount
	return q
}

// Offset specifies a starting index from which to start counting records that
// will be returned.
func (q *Query) Offset(amount uint) *Query {
	q.offset = amount
	return q
}

// Include specifies one or more field names which will be read from the database
// and scanned into the resulting models when the query is run. Field names which
// are not specified in Include will not be read or scanned. You can only use one
// of Include or Exclude, not both on the same query.
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
// Exclude, not both on the same query.
func (q *Query) Exclude(fields ...string) *Query {
	if len(q.includes) > 0 {
		q.setErrorIfNone(errors.New("zoom: cannot use both Include and Exclude modifiers on a query"))
		return q
	}
	q.excludes = append(q.excludes, fields...)
	return q
}

func (q *Query) Run() (interface{}, error) {
	if q.err != nil {
		return nil, q.err
	}

	ids, err := q.getIds()
	if err != nil {
		return nil, err
	}

	// create a slice in which to store results using reflection the
	// type of the slice whill match the type of the model being queried
	resultsVal := reflect.New(reflect.SliceOf(q.modelSpec.modelType))
	resultsVal.Elem().Set(reflect.MakeSlice(reflect.SliceOf(q.modelSpec.modelType), 0, 0))

	if err := scanModelsByIds(resultsVal, q.modelSpec.modelName, ids, q.getIncludes()); err != nil {
		return resultsVal.Elem().Interface(), err
	}
	return resultsVal.Elem().Interface(), nil
}

func (q *Query) Scan(in interface{}) error {
	if q.err != nil {
		return q.err
	}

	// make sure we are dealing with the right type
	typ := reflect.TypeOf(in).Elem()
	if !(typ.Kind() == reflect.Slice) {
		msg := fmt.Sprintf("zoom: Query.Scan requires a pointer to a slice or array as an argument. Got: %T", in)
		return errors.New(msg)
	}
	elemType := typ.Elem()
	if !typeIsPointerToStruct(elemType) {
		msg := fmt.Sprintf("zoom: Query.Scan requires a pointer to a slice of pointers to model structs. Got: %T", in)
		return errors.New(msg)
	}
	if elemType != q.modelSpec.modelType {
		msg := fmt.Sprintf("zoom: argument for Query.Scan did not match the type corresponding to the model name given in the NewQuery constructor.\nExpected %T but got %T", reflect.SliceOf(q.modelSpec.modelType), in)
		return errors.New(msg)
	}

	ids, err := q.getIds()
	if err != nil {
		return err
	}

	resultsVal := reflect.ValueOf(in)
	resultsVal.Elem().Set(reflect.MakeSlice(reflect.SliceOf(q.modelSpec.modelType), 0, 0))

	return scanModelsByIds(resultsVal, q.modelSpec.modelName, ids, q.getIncludes())
}

func (q *Query) Count() (int, error) {
	if q.err != nil {
		return 0, q.err
	}
	return q.getIdCount()
}

func (q *Query) IdsOnly() ([]string, error) {
	if q.err != nil {
		return nil, q.err
	}
	return q.getIds()
}

func (q *Query) setErrorIfNone(e error) {
	if q.err == nil {
		q.err = e
	}
}

// getIncludes parses the includes and excludes properties to return
// a list of fieldNames which should be included in all find operations.
// a return value of nil means that all fields should be considered.
func (q *Query) getIncludes() []string {
	if len(q.includes) != 0 {
		return q.includes
	} else if len(q.excludes) != 0 {
		results := q.modelSpec.fieldNames
		for _, name := range q.excludes {
			results = removeElementFromStringSlice(results, name)
		}
		return results
	}
	return nil
}

// getIds() executes a single command and returns the ids of every
// model which should be found for the query.
func (q *Query) getIds() ([]string, error) {
	conn := GetConn()
	defer conn.Close()

	// construct a redis command to get the ids
	if q.order.fieldName == "" {
		// without ordering
		if q.offset != 0 {
			return nil, errors.New("zoom: offset cannot be applied to queries without an order.")
		}
		indexKey := q.modelSpec.modelName + ":all"
		args := redis.Args{}.Add(indexKey)
		var command string
		if q.limit == 0 {
			command = "SMEMBERS"
		} else {
			command = "SRANDMEMBER"
			args = args.Add(q.limit)
		}
		return redis.Strings(conn.Do(command, args...))
	} else {
		// with ordering
		args := redis.Args{}
		var command string
		if q.order.orderType == ascending {
			command = "ZRANGE"
		} else if q.order.orderType == descending {
			command = "ZREVRANGE"
		}
		start := q.offset
		stop := -1
		if q.limit != 0 {
			stop = int(start) + int(q.limit) - 1
		}
		indexKey := q.modelSpec.modelName + ":" + q.order.redisName
		args = args.Add(indexKey).Add(start).Add(stop)
		if q.order.indexType == indexNumeric || q.order.indexType == indexBoolean {
			// ordered by numeric or boolean index
			// just return results of command directly
			return redis.Strings(conn.Do(command, args...))
		} else {
			// ordered by alpha index
			// we need to parse ids from the result of the command
			ids, err := redis.Strings(conn.Do(command, args...))
			if err != nil {
				return nil, err
			}
			for i, valueAndId := range ids {
				ids[i] = extractModelIdFromAlphaIndexValue(valueAndId)
			}
			return ids, nil
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
func scanModelsByIds(sliceVal reflect.Value, modelName string, ids []string, includes []string) error {
	t := newTransaction()
	for _, id := range ids {
		mr, err := newModelRefFromName(modelName)
		if err != nil {
			return err
		}
		mr.model.setId(id)
		if err := t.findModel(mr, includes); err != nil {
			if _, ok := err.(*KeyNotFoundError); ok {
				continue // key not found errors are fine
				// TODO: update the index in this case? Or maybe if it keeps happening?
			}
		}
		sliceVal.Elem().Set(reflect.Append(sliceVal.Elem(), mr.modelVal()))
	}
	return t.exec()
}

// getIdCount returns the number of models that would be found if the
// query were executed, but does not actually find them.
func (q *Query) getIdCount() (int, error) {
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
