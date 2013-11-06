// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File transaction.go contains all code dealing with the
// transactions abstraction, including construction, adding
// commands, and execution of the transaction.

package zoom

import (
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"reflect"
)

type transaction struct {
	conn     redis.Conn
	handlers []func(interface{}) error
	// references is a sort of per transaction cache. used to prevent duplicate queries and infinite recursion
	references map[string]interface{}
}

func newTransaction() *transaction {
	t := &transaction{
		conn:       GetConn(),
		references: make(map[string]interface{}),
	}
	t.conn.Send("MULTI")
	return t
}

func (t *transaction) command(cmd string, args []interface{}, handler func(interface{}) error) error {
	if err := t.conn.Send(cmd, args...); err != nil {
		t.discard()
		return err
	}
	t.handlers = append(t.handlers, handler)
	return nil
}

func (t *transaction) exec() error {
	defer t.conn.Close()

	// invoke redis driver to execute the transaction
	replies, err := redis.MultiBulk(t.conn.Do("EXEC"))
	if err != nil {
		t.discard()
		return err
	}

	// call the handler functions sequentially, passing in
	// the corresponding replies.
	for i, handler := range t.handlers {
		if handler != nil {
			if err := handler(replies[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *transaction) discard() error {
	defer t.conn.Close()
	_, err := t.conn.Do("DISCARD")
	if err != nil {
		return err
	}
	return nil
}

// Useful Handlers

// newScanHandler invokes redis driver to scan single values into the corresponding scannable
func newScanHandler(scannables []interface{}) func(interface{}) error {
	return func(reply interface{}) error {
		replies, err := redis.Values(reply, nil)
		if err != nil {
			return err
		}
		if _, err := redis.Scan(replies, scannables...); err != nil {
			return err
		}
		return nil
	}
}

// newScanStructHandler invokes redis driver to scan multiple values into scannable (a struct)
func newScanModelHandler(mr modelRef) func(interface{}) error {
	return func(reply interface{}) error {
		bulk, err := redis.MultiBulk(reply, nil)
		if err != nil {
			return err
		}
		if err := scanModel(bulk, mr); err != nil {
			return err
		}
		return nil
	}
}

// newScanSliceHandler invokes redis driver to scan multiple values into a single
// slice or array. The reflect.Value of the slice or array should be passed as an argument.
func newScanSliceHandler(scanVal reflect.Value) func(interface{}) error {
	return func(reply interface{}) error {
		bulk, err := redis.MultiBulk(reply, nil)
		if err != nil {
			return err
		}
		scanType := scanVal.Type()
		scanElem := scanType.Elem()
		for _, el := range bulk {
			srcElem := reflect.ValueOf(el)
			converted := srcElem.Convert(scanElem)
			scanVal.Set(reflect.Append(scanVal, converted))
		}

		return nil
	}
}

// Useful Operations for Transactions

// saveModel adds all the necessary commands to save a given model to the redis database
// this includes indeces and external sets/lists
func (t *transaction) saveModel(m Model) error {
	mr, err := newModelRefFromInterface(m)
	if err != nil {
		return err
	}

	// set the id if needed
	if m.getId() == "" {
		m.setId(generateRandomId())
	}

	// add an operation to write data to database
	if err := t.saveStruct(mr); err != nil {
		return err
	}

	// add an operation to add to index for this model
	if err := t.index(mr); err != nil {
		return err
	}

	// add operations to save external lists and sets
	if err := t.saveModelLists(mr); err != nil {
		return err
	}
	if err := t.saveModelSets(mr); err != nil {
		return err
	}

	// add operations to save model relationships
	if err := t.saveModelRelationships(mr); err != nil {
		return err
	}
	return nil
}

func (t *transaction) saveStruct(mr modelRef) error {
	args, err := mr.mainHashArgs()
	if err != nil {
		return err
	}
	if len(args) > 1 {
		if err := t.command("HMSET", args, nil); err != nil {
			return err
		}
	}
	return nil
}

func (t *transaction) index(mr modelRef) error {
	args := redis.Args{}.Add(mr.indexKey()).Add(mr.model.getId())
	if err := t.command("SADD", args, nil); err != nil {
		return err
	}
	return nil
}

func (t *transaction) saveModelLists(mr modelRef) error {
	for _, list := range mr.modelSpec.lists {
		field := mr.value(list.fieldName)
		if field.IsNil() {
			continue // skip empty lists
		}
		listKey := mr.key() + ":" + list.redisName
		args := redis.Args{}.Add(listKey).AddFlat(field.Interface())
		if err := t.command("RPUSH", args, nil); err != nil {
			return err
		}
	}
	return nil
}

func (t *transaction) saveModelSets(mr modelRef) error {
	for _, set := range mr.modelSpec.sets {
		field := mr.value(set.fieldName)
		if field.IsNil() {
			continue // skip empty sets
		}
		setKey := mr.key() + ":" + set.redisName
		args := redis.Args{}.Add(setKey).AddFlat(field.Interface())
		if err := t.command("SADD", args, nil); err != nil {
			return err
		}
	}
	return nil
}

func (t *transaction) saveModelRelationships(mr modelRef) error {
	for _, r := range mr.modelSpec.relationships {
		if r.rType == oneToOne {
			if err := t.saveModelOneToOneRelationship(mr, r); err != nil {
				return err
			}
		} else if r.rType == oneToMany {
			if err := t.saveModelOneToManyRelationship(mr, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *transaction) saveModelOneToOneRelationship(mr modelRef, r relationship) error {
	field := mr.value(r.fieldName)
	if field.IsNil() {
		return nil
	}
	rModel, ok := field.Interface().(Model)
	if !ok {
		msg := fmt.Sprintf("zoom: cannot convert type %s to Model\n", field.Type().String())
		return errors.New(msg)
	}
	if rModel.getId() == "" {
		msg := fmt.Sprintf("zoom: cannot save a relation for a model with no Id: %+v\n. Must save the related model first.", rModel)
		return errors.New(msg)
	}

	// add a command to the transaction to set the relation key
	relationKey := mr.key() + ":" + r.redisName
	args := redis.Args{relationKey, rModel.getId()}
	if err := t.command("SET", args, nil); err != nil {
		return err
	}
	return nil
}

func (t *transaction) saveModelOneToManyRelationship(mr modelRef, r relationship) error {
	field := mr.value(r.fieldName)
	if field.IsNil() {
		return nil
	}

	// get a slice of ids from the elements of the field
	ids := make([]string, 0)
	for i := 0; i < field.Len(); i++ {
		rElem := field.Index(i)

		// convert the individual element to a model
		rModel, ok := rElem.Interface().(Model)
		if !ok {
			msg := fmt.Sprintf("zoom: cannot convert type %s to Model\n", field.Type().String())
			return errors.New(msg)
		}

		// make sure the id is not nil
		if rModel.getId() == "" {
			msg := fmt.Sprintf("zoom: cannot save a relation for a model with no Id: %+v\n. Must save the related model first.", rModel)
			return errors.New(msg)
		}

		// add its id to the slice
		ids = append(ids, rModel.getId())
	}

	if len(ids) > 0 {

		// add a command to the transaction to save the ids
		relationKey := mr.key() + ":" + r.redisName
		args := redis.Args{}.Add(relationKey).AddFlat(ids)
		if err := t.command("SADD", args, nil); err != nil {
			return err
		}
	}
	return nil
}

func (t *transaction) findModel(mr modelRef, includes []string) error {

	// check prior references to prevent infinite recursion or unnecessary queries
	if prior, found := t.references[mr.key()]; found {
		reflect.ValueOf(mr.model).Elem().Set(reflect.ValueOf(prior).Elem())
		return nil
	}
	t.references[mr.key()] = mr.model

	// scan the hash values directly into the struct
	if includes == nil {
		// use HGETALL to get all the fields for the model
		if err := t.command("HGETALL", redis.Args{}.Add(mr.key()), newScanModelHandler(mr)); err != nil {
			return err
		}
	} else {
		// get the appropriate scannable fields
		fields := make([]interface{}, 0)
		for _, fieldName := range includes {
			fields = append(fields, mr.value(fieldName).Addr().Interface())
		}

		// use HMGET to get only certain fields for the model
		if len(fields) != 0 {
			args := redis.Args{}.Add(mr.key()).AddFlat(includes)
			if err := t.command("HMGET", args, newScanHandler(fields)); err != nil {
				return err
			}
		}
	}

	// find all the external sets and lists for the model
	if len(mr.modelSpec.lists) != 0 {
		if err := t.findModelLists(mr, includes); err != nil {
			return err
		}
	}
	if len(mr.modelSpec.sets) != 0 {
		if err := t.findModelSets(mr, includes); err != nil {
			return err
		}
	}

	// find the relationships for the model
	if len(mr.modelSpec.relationships) != 0 {
		if err := t.findModelRelationships(mr, includes); err != nil {
			return err
		}
	}

	return nil
}

func (t *transaction) findModelLists(mr modelRef, includes []string) error {
	for _, list := range mr.modelSpec.lists {
		if includes != nil {
			if !stringSliceContains(list.fieldName, includes) {
				continue // skip field names that are not in includes
			}
		}

		field := mr.value(list.fieldName)

		// use LRANGE to get all the members of the list
		listKey := mr.key() + ":" + list.redisName
		args := redis.Args{listKey, 0, -1}
		if err := t.command("LRANGE", args, newScanSliceHandler(field)); err != nil {
			return err
		}
	}
	return nil
}

func (t *transaction) findModelSets(mr modelRef, includes []string) error {
	for _, set := range mr.modelSpec.sets {
		if includes != nil {
			if !stringSliceContains(set.fieldName, includes) {
				continue // skip field names that are not in includes
			}
		}

		field := mr.value(set.fieldName)

		// use SMEMBERS to get all the members of the set
		setKey := mr.key() + ":" + set.redisName
		args := redis.Args{setKey}
		if err := t.command("SMEMBERS", args, newScanSliceHandler(field)); err != nil {
			return err
		}
	}
	return nil
}

func (t *transaction) findModelRelationships(mr modelRef, includes []string) error {
	for _, r := range mr.modelSpec.relationships {
		if includes != nil {
			if !stringSliceContains(r.fieldName, includes) {
				continue // skip field names that are not in includes
			}
		}
		if r.rType == oneToOne {
			if err := t.findModelOneToOneRelation(mr, r); err != nil {
				return err
			}
		} else if r.rType == oneToMany {
			if err := t.findModelOneToManyRelation(mr, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *transaction) findModelOneToOneRelation(mr modelRef, r relationship) error {

	// TODO: use scripting to retain integrity of the transaction (we want
	// to perform only one round trip per transaction).
	conn := GetConn()
	defer conn.Close()

	// invoke redis driver to get the id
	relationKey := mr.key() + ":" + r.redisName
	id, err := redis.String(conn.Do("GET", relationKey))
	if err != nil {
		return err
	}

	// instantiate the field using reflection
	field := mr.value(r.fieldName)

	// check if the key is already referenced in this transaction
	rModelName, _ := getRegisteredNameFromType(field.Type())
	rModelKey := rModelName + ":" + id
	if prior, found := t.references[rModelKey]; found {
		// use the same pointer (it's the same object)
		field.Set(reflect.ValueOf(prior))
		return nil
	} else {
		// create a new pointer
		field.Set(reflect.New(field.Type().Elem()))
	}

	// convert field to a model
	rModel, ok := field.Interface().(Model)
	if !ok {
		msg := fmt.Sprintf("zoom: cannot convert type %s to Model\n", field.Type().String())
		return errors.New(msg)
	}

	// set id and create modelRef
	rModel.setId(id)
	rModelRef, err := newModelRefFromInterface(rModel)
	if err != nil {
		return err
	}

	// add a find operation to the transaction
	if err := t.findModel(rModelRef, nil); err != nil {
		return err
	}

	return nil
}

func (t *transaction) findModelOneToManyRelation(mr modelRef, r relationship) error {

	// TODO: use scripting to retain integrity of the transaction (we want
	// to perform only one round trip per transaction).
	conn := GetConn()
	defer conn.Close()

	// invoke redis driver to get a set of keys
	relationKey := mr.key() + ":" + r.redisName
	ids, err := redis.Strings(conn.Do("SMEMBERS", relationKey))
	if err != nil {
		return err
	}

	field := mr.value(r.fieldName)
	rType := field.Type().Elem()

	// iterate through the ids and find each model
	for _, id := range ids {

		// check if the key is already referenced in this transaction
		rModelName, _ := getRegisteredNameFromType(rType)
		rModelKey := rModelName + ":" + id
		if prior, found := t.references[rModelKey]; found {
			// use the same pointer (it's the same object)
			sliceVal := reflect.Append(field, reflect.ValueOf(prior))
			field.Set(sliceVal)
			continue
		}

		rVal := reflect.New(rType.Elem())
		rModel, ok := rVal.Interface().(Model)
		if !ok {
			msg := fmt.Sprintf("zoom: cannot convert type %s to Model\n", rType.String())
			return errors.New(msg)
		}

		// set id and create modelRef
		rModel.setId(id)
		rModelRef, err := newModelRefFromInterface(rModel)
		if err != nil {
			return err
		}

		// add a find operation to the transaction
		if err := t.findModel(rModelRef, nil); err != nil {
			return err
		}

		// append to the field slice
		sliceVal := reflect.Append(field, rVal)
		field.Set(sliceVal)
	}

	return nil
}

func (t *transaction) deleteModel(modelName, id string) error {

	// add an operation to delete the model itself
	key := modelName + ":" + id
	if err := t.delete(key); err != nil {
		return err
	}

	// add an operation to remove the model id from the index
	indexKey := modelName + ":all"
	if err := t.unindex(indexKey, id); err != nil {
		return err
	}
	return nil
}

func (t *transaction) delete(key string) error {
	if err := t.command("DEL", redis.Args{}.Add(key), nil); err != nil {
		return err
	}
	return nil
}

func (t *transaction) unindex(key, value string) error {
	args := redis.Args{}.Add(key).Add(value)
	if err := t.command("SREM", args, nil); err != nil {
		return err
	}
	return nil
}
