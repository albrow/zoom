// Copyright 2014 Alex Browne.  All rights reserved.
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
	"github.com/twmb/algoimpl/go/graph"

	"reflect"
	"strings"
)

// replyHandler is a callback function used to handle the response from
// some redis command
type replyHandler func(interface{}) error

// transaction is an abstraction around a multi-phase transaction. It
// consists of one or more phases, where each phase is an atomic redis
// transaction.
type transaction struct {
	conn       redis.Conn
	phases     map[string]*phase
	graph      *graph.Graph
	modelCache map[string]interface{}
}

func newTransaction() *transaction {
	return &transaction{
		phases:     map[string]*phase{},
		graph:      graph.New(graph.Directed),
		conn:       GetConn(),
		modelCache: map[string]interface{}{},
	}
}

func (t *transaction) linearize() ([]*phase, error) {
	// If the nodes could be linearized, components is a [][]graph.Node where
	// each inner slice contains a single node. We need to iterate through and
	// 1) get the id of the node, 2) get the phase corresponding to that id,
	// and 3) add each phase to a list of ordered phases, which we'll return
	// Example of correct output:
	// [
	//   [a]
	//   [b]
	//   [c]
	// ]
	// Where a, b, and c represent nodes
	// Example of what the output will look like if there is a cycle:
	// [
	//   [a, b]
	//   [c]
	// ]
	components := t.graph.StronglyConnectedComponents()
	if len(components) != len(t.phases) {
		// There was at least one cycle. Find a piece of components which has
		// more than one node in it. The nodes in that piece are a cycle because
		// they are strongly connected.
		cycleIds := []string{}
		for _, nodes := range components {
			if len(nodes) > 1 {
				// Get the phase ids of the nodes in the cycle
				for _, node := range nodes {
					if id, err := getIdForNode(node); err != nil {
						return nil, err
					} else {
						cycleIds = append(cycleIds, id)
					}
				}
				break
			}
		}
		cycleExplanation := strings.Join(cycleIds, " -> ")
		return nil, fmt.Errorf("Could not linearize transaction phases because there was a cycle: %s", cycleExplanation)
	}
	orderedPhases := []*phase{}
	for _, nodes := range components {
		node := nodes[0]
		phaseId, err := getIdForNode(node)
		if err != nil {
			return nil, err
		}
		phase, found := t.phases[phaseId]
		if !found {
			return nil, fmt.Errorf("Could not find phase with id: %s", phaseId)
		}
		orderedPhases = append(orderedPhases, phase)
	}
	// phaseIds := []string{}
	// for _, phase := range orderedPhases {
	// 	phaseIds = append(phaseIds, phase.id)
	// }
	// fmt.Printf("linearized results: %v\n", phaseIds)
	return orderedPhases, nil
}

func getIdForNode(node graph.Node) (string, error) {
	if id, ok := (*node.Value).(string); !ok {
		msg := fmt.Sprintf("Could not convert node value: %v to string!", node.Value)
		if node.Value != nil {
			typ := reflect.TypeOf(*node.Value)
			msg += fmt.Sprintf(" Had type: %s", typ.String())
		} else {
			msg += " Node was nil"
		}
		return "", errors.New(msg)
	} else {
		return id, nil
	}
}

func (t *transaction) exec() error {
	orderedPhases, err := t.linearize()
	if err != nil {
		return err
	}
	for _, phase := range orderedPhases {
		if err := phase.exec(t.conn); err != nil {
			return err
		}
	}
	// Close this transaction's connection to add it
	// back into the pool
	return t.conn.Close()
}

type phase struct {
	t        *transaction
	id       string
	commands []*command
	node     graph.Node
	pre      func(*phase) error
	post     func(*phase) error
}

func (t *transaction) addPhase(id string, pre func(*phase) error, post func(*phase) error) *phase {
	if id == "" {
		id = generateRandomId()
	}
	p := &phase{
		t:    t,
		id:   id,
		node: t.graph.MakeNode(),
		pre:  pre,
		post: post,
	}
	*p.node.Value = id
	t.phases[p.id] = p
	return p
}

func (p *phase) String() string {
	result := fmt.Sprintf("%s:\n", p.id)
	for _, c := range p.commands {
		result += fmt.Sprintf("\t%s\n", c)
	}
	return result
}

func (p *phase) exec(conn redis.Conn) error {
	// execute the pre-handler for this phase
	if p.pre != nil {
		if err := p.pre(p); err != nil {
			return err
		}
	}

	if len(p.commands) > 0 {
		// send all the commands for this phase using MULTI
		conn.Send("MULTI")
		for _, c := range p.commands {
			if err := conn.Send(c.name, c.args...); err != nil {
				return err
			}
		}
		// use EXEC to execute all the commands at once
		replies, err := redis.Values(conn.Do("EXEC"))
		if err != nil {
			p.t.discard()
			return err
		}
		// iterate through the replies, calling the corresponding handler functions
		for i, reply := range replies {
			handler := p.commands[i].handler
			if handler != nil {
				if err := handler(reply); err != nil {
					return err
				}
			}
		}
	}

	// execute the post-handler for this phase
	if p.post != nil {
		return p.post(p)
	}
	return nil
}

func (p *phase) addDependency(dep *phase) {
	p.t.graph.MakeEdge(dep.node, p.node)
}

type command struct {
	name    string
	args    []interface{}
	handler replyHandler
}

func (p *phase) addCommand(name string, args []interface{}, handler replyHandler) {
	c := &command{
		name:    name,
		args:    args,
		handler: handler,
	}
	p.commands = append(p.commands, c)
}

func (c *command) String() string {
	argStrings := []string{}
	for _, arg := range c.args {
		argStrings = append(argStrings, fmt.Sprint(arg))
	}
	return fmt.Sprintf("\t%s %s", c.name, strings.Join(argStrings, " "))
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

// newScanModelHandler invokes redis driver to scan multiple values into scannable (a struct)
func newScanModelHandler(mr modelRef, includes []string) func(interface{}) error {
	return func(reply interface{}) error {
		bulk, err := redis.MultiBulk(reply, nil)
		if err != nil {
			return err
		}
		if len(bulk) == 0 {
			// if there is nothing in the hash, we should check if the model exists
			// it might still exist if there are relationships, but no other data
			return checkModelExists(mr)
		}
		if err := scanModel(bulk, mr, includes); err != nil {
			return err
		} else {
			return nil
		}
	}
}

// newScanModelSliceHandler invokes redis driver to scan multiple values into a single
// slice or array. The reflect.Value of the slice or array should be passed as an argument.
// it requires a passed in mr and keeps track of miss counts. Will return an error
// if the model does not exist. Use this for scanning into a struct field.
func newScanModelSliceHandler(mr modelRef, scanVal reflect.Value) func(interface{}) error {
	return func(reply interface{}) error {
		bulk, err := redis.MultiBulk(reply, nil)
		if err != nil {
			return err
		}
		if len(bulk) == 0 {
			// there was a miss
			// if there is nothing in the hash, we should check if the model exists
			// it might still exist if there are relationships, but no other data
			return checkModelExists(mr)
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

// Shortcuts for adding common commands to a phase

// saveModel adds all the necessary commands to save a given model to the redis database.
// this includes indexes and relationships
func (p *phase) saveModel(m Model) error {
	mr, err := newModelRefFromModel(m)
	if err != nil {
		return err
	}

	// set the id if needed
	if m.GetId() == "" {
		m.SetId(generateRandomId())
	}

	// add operations to save the model field indexes
	// we do this first becuase it may require a read before write :(
	if err := p.saveModelIndexes(mr); err != nil {
		return err
	}

	// add an operation to write data to database
	if err := p.saveModelStruct(mr); err != nil {
		return err
	}

	// add an operation to add to index this model id in the
	// set of all model ids for this type
	p.index(mr)

	// add operations to save model relationships
	p.saveModelRelationships(mr)
	return nil
}

func (p *phase) saveModelStruct(mr modelRef) error {
	if args, err := mr.mainHashArgs(); err != nil {
		return err
	} else {
		if len(args) > 1 {
			p.addCommand("HMSET", args, nil)
		}
		return nil
	}
}

func (p *phase) index(mr modelRef) {
	args := redis.Args{}.Add(mr.indexKey()).Add(mr.model.GetId())
	p.addCommand("SADD", args, nil)
}

func (p *phase) saveModelLists(mr modelRef) {
	for _, list := range mr.modelSpec.lists {
		field := mr.value(list.fieldName)
		if field.IsNil() {
			continue // skip empty lists
		}
		listKey := mr.key() + ":" + list.redisName
		args := redis.Args{}.Add(listKey).AddFlat(field.Interface())
		p.addCommand("RPUSH", args, nil)
	}
}

func (p *phase) saveModelSets(mr modelRef) {
	for _, set := range mr.modelSpec.sets {
		field := mr.value(set.fieldName)
		if field.IsNil() {
			continue // skip empty sets
		}
		setKey := mr.key() + ":" + set.redisName
		args := redis.Args{}.Add(setKey).AddFlat(field.Interface())
		p.addCommand("SADD", args, nil)
	}
}

func (p *phase) saveModelRelationships(mr modelRef) error {
	for _, r := range mr.modelSpec.relationships {
		if r.relType == oneToOne {
			if err := p.saveModelOneToOneRelationship(mr, r); err != nil {
				return err
			}
		} else if r.relType == oneToMany {
			if err := p.saveModelOneToManyRelationship(mr, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *phase) saveModelOneToOneRelationship(mr modelRef, relationship *fieldSpec) error {
	field := mr.value(relationship.fieldName)
	if field.IsNil() {
		return nil
	}
	rModel, ok := field.Interface().(Model)
	if !ok {
		return fmt.Errorf("zoom: cannot convert type %s to Model\n", field.Type().String())
	}
	if rModel.GetId() == "" {
		return fmt.Errorf("zoom: cannot save a relation for a model with no Id: %+v\n. Must save the related model first.", rModel)
	}

	// add a command to the transaction to set the relation key
	relationKey := mr.key() + ":" + relationship.redisName
	args := redis.Args{relationKey, rModel.GetId()}
	p.addCommand("SET", args, nil)
	return nil
}

func (p *phase) saveModelOneToManyRelationship(mr modelRef, relationship *fieldSpec) error {
	field := mr.value(relationship.fieldName)
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
			return fmt.Errorf("zoom: cannot convert type %s to Model\n", field.Type().String())
		}

		// make sure the id is not nil
		if rModel.GetId() == "" {
			return fmt.Errorf("zoom: cannot save a relation for a model with no Id: %+v\n. Must save the related model first.", rModel)
		}

		// add its id to the slice
		ids = append(ids, rModel.GetId())
	}

	if len(ids) > 0 {

		// add a command to the transaction to save the ids
		relationKey := mr.key() + ":" + relationship.redisName
		args := redis.Args{}.Add(relationKey).AddFlat(ids)
		p.addCommand("SADD", args, nil)
	}
	return nil
}

func (p *phase) saveModelIndexes(mr modelRef) error {
	for _, pi := range mr.modelSpec.primativeIndexes {
		if pi.indexType == indexNumeric {
			if err := p.saveModelPrimativeIndexNumeric(mr, pi); err != nil {
				return err
			}
		} else if pi.indexType == indexAlpha {
			p.saveModelPrimativeIndexAlpha(mr, pi)
		} else if pi.indexType == indexBoolean {
			p.saveModelPrimativeIndexBoolean(mr, pi)
		}
	}

	for _, pi := range mr.modelSpec.pointerIndexes {
		if pi.indexType == indexNumeric {
			if err := p.saveModelPointerIndexNumeric(mr, pi); err != nil {
				return err
			}
		} else if pi.indexType == indexAlpha {
			p.saveModelPointerIndexAlpha(mr, pi)
		} else if pi.indexType == indexBoolean {
			p.saveModelPointerIndexBoolean(mr, pi)
		}
	}
	return nil
}

func (p *phase) saveModelPrimativeIndexNumeric(mr modelRef, primative *fieldSpec) error {
	indexKey := mr.modelSpec.modelName + ":" + primative.redisName
	score, err := convertNumericToFloat64(mr.value(primative.fieldName))
	if err != nil {
		return err
	}
	id := mr.model.GetId()
	p.indexNumeric(indexKey, score, id)
	return nil
}

func (p *phase) saveModelPointerIndexNumeric(mr modelRef, pointer *fieldSpec) error {
	if mr.value(pointer.fieldName).IsNil() {
		// TODO: special case for indexing nil pointers?
		return nil // skip nil pointers for now
	}
	indexKey := mr.modelSpec.modelName + ":" + pointer.redisName
	score, err := convertNumericToFloat64(mr.value(pointer.fieldName).Elem())
	if err != nil {
		return err
	}
	id := mr.model.GetId()
	p.indexNumeric(indexKey, score, id)
	return nil
}

func (p *phase) indexNumeric(indexKey string, score float64, id string) {
	args := redis.Args{}.Add(indexKey).Add(score).Add(id)
	p.addCommand("ZADD", args, nil)
}

func (p *phase) saveModelPrimativeIndexAlpha(mr modelRef, primative *fieldSpec) {
	p.removeOldAlphaIndex(mr, primative.fieldName, primative.redisName)
	indexKey := mr.modelSpec.modelName + ":" + primative.redisName
	value := mr.value(primative.fieldName).String()
	id := mr.model.GetId()
	p.indexAlpha(indexKey, value, id)
}

func (p *phase) saveModelPointerIndexAlpha(mr modelRef, pointer *fieldSpec) {
	p.removeOldAlphaIndex(mr, pointer.fieldName, pointer.redisName)
	if mr.value(pointer.fieldName).IsNil() {
		// TODO: special case for indexing nil pointers?
		return // skip nil pointers for now
	}
	indexKey := mr.modelSpec.modelName + ":" + pointer.redisName
	value := mr.value(pointer.fieldName).Elem().String()
	id := mr.model.GetId()
	p.indexAlpha(indexKey, value, id)
}

func (p *phase) indexAlpha(indexKey, value, id string) {
	member := value + " " + id
	args := redis.Args{}.Add(indexKey).Add(0).Add(member)
	p.addCommand("ZADD", args, nil)
}

// Remove the alpha index that may have existed before an update or resave of the model
// this requires a read before write, which is a performance penalty but unfortunatlely
// is unavoidable because of the hacky way we're indexing alpha fields.
func (p *phase) removeOldAlphaIndex(mr modelRef, fieldName string, redisName string) {
	key := mr.key()
	args := redis.Args{}.Add(key).Add(redisName)
	p.addCommand("HGET", args, func(reply interface{}) error {
		if reply == nil {
			return nil
		}
		oldFieldValue, err := redis.String(reply, nil)
		if err != nil {
			return err
		}
		if oldFieldValue == "" {
			return nil
		}
		if oldFieldValue != "" && oldFieldValue != mr.value(fieldName).String() {
			// TODO: Is there a way to do this without creating a new connection?
			// At the very least can we consolidate these operations into a single transaction
			// if there are more than one old indexes to be removed?
			conn := GetConn()
			defer conn.Close()
			alphaIndexKey := mr.modelSpec.modelName + ":" + fieldName
			member := oldFieldValue + " " + mr.model.GetId()
			if _, err := conn.Do("ZREM", alphaIndexKey, member); err != nil {
				return err
			}
		}
		return nil
	})
}

func (p *phase) saveModelPrimativeIndexBoolean(mr modelRef, primative *fieldSpec) {
	value := mr.value(primative.fieldName).Bool()
	id := mr.model.GetId()
	indexKey := mr.modelSpec.modelName + ":" + primative.redisName
	var score float64
	if value == true {
		score = 1.0
	} else {
		score = 0.0
	}
	p.indexNumeric(indexKey, score, id)
}

func (p *phase) saveModelPointerIndexBoolean(mr modelRef, pointer *fieldSpec) {
	if mr.value(pointer.fieldName).IsNil() {
		// TODO: special case for indexing nil pointers?
		return // skip nil pointers for now
	}
	value := mr.value(pointer.fieldName).Elem().Bool()
	id := mr.model.GetId()
	indexKey := mr.modelSpec.modelName + ":" + pointer.redisName
	var score float64
	if value == true {
		score = 1.0
	} else {
		score = 0.0
	}
	p.indexNumeric(indexKey, score, id)
}

func (p *phase) scanModel(mr modelRef, includes []string) error {
	// check for mutex
	if s, ok := mr.model.(Syncer); ok {
		mutexId := fmt.Sprintf("%T:%s", mr.model, mr.model.GetId())
		s.SetMutexId(mutexId)
		s.Lock()
	}

	// check model cache to prevent infinite recursion or unnecessary queries
	if prior, found := p.t.modelCache[mr.key()]; found {
		reflect.ValueOf(mr.model).Elem().Set(reflect.ValueOf(prior).Elem())
		return nil
	}
	p.t.modelCache[mr.key()] = mr.model

	// scan the hash values directly into the struct
	if includes == nil {
		// use HMGET to get all the fields for the model
		args := redis.Args{}.Add(mr.key()).AddFlat(mr.modelSpec.mainHashFieldNames())
		p.addCommand("HMGET", args, newScanModelHandler(mr, nil))
	} else {
		// get the appropriate scannable fields
		fields := make([]interface{}, 0)
		for _, fieldName := range includes {
			fields = append(fields, mr.value(fieldName).Addr().Interface())
		}

		// use HMGET to get only the included fields for the model
		if len(fields) != 0 {
			args := redis.Args{}.Add(mr.key()).AddFlat(includes)
			p.addCommand("HMGET", args, newScanModelHandler(mr, includes))
		}
	}

	// find the relationships for the model
	if len(mr.modelSpec.relationships) != 0 {
		if err := p.findModelRelationships(mr, includes); err != nil {
			return err
		}
	}

	return nil
}

func (p *phase) findModelLists(mr modelRef, includes []string) {
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
		p.addCommand("LRANGE", args, newScanModelSliceHandler(mr, field))
	}
}

func (p *phase) findModelSets(mr modelRef, includes []string) {
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
		p.addCommand("SMEMBERS", args, newScanModelSliceHandler(mr, field))
	}
}

func (p *phase) findModelRelationships(mr modelRef, includes []string) error {
	for _, r := range mr.modelSpec.relationships {
		if includes != nil {
			if !stringSliceContains(r.fieldName, includes) {
				continue // skip field names that are not in includes
			}
		}
		if r.relType == oneToOne {
			if err := p.findModelOneToOneRelation(mr, r); err != nil {
				return err
			}
		} else if r.relType == oneToMany {
			if err := p.findModelOneToManyRelation(mr, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *phase) findModelOneToOneRelation(mr modelRef, relationship *fieldSpec) error {
	// TODO: use scripting to retain integrity of the transaction (we want
	// to perform only one round trip per transaction).
	conn := GetConn()
	defer conn.Close()

	// invoke redis driver to get the id
	relationKey := mr.key() + ":" + relationship.redisName
	response, err := conn.Do("GET", relationKey)
	if err != nil {
		return err
	} else if response == nil {
		return nil
	}
	id, err := redis.String(response, nil)
	if err != nil {
		return err
	}

	// instantiate the field using reflection
	field := mr.value(relationship.fieldName)

	// check if a model with key is already cached in this transaction
	rModelName, _ := getRegisteredNameFromType(field.Type())
	rModelKey := rModelName + ":" + id
	if prior, found := p.t.modelCache[rModelKey]; found {
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
		return fmt.Errorf("zoom: cannot convert type %s to Model\n", field.Type().String())
	}

	// set id and create modelRef
	rModel.SetId(id)
	rModelRef, err := newModelRefFromModel(rModel)
	if err != nil {
		return err
	}

	// add a find operation to the transaction
	if err := p.scanModel(rModelRef, nil); err != nil {
		return err
	}

	return nil
}

func (p *phase) findModelOneToManyRelation(mr modelRef, relationship *fieldSpec) error {

	// TODO: use scripting to retain integrity of the transaction (we want
	// to perform only one round trip per transaction).
	conn := GetConn()
	defer conn.Close()

	// invoke redis driver to get a set of keys
	relationKey := mr.key() + ":" + relationship.redisName
	ids, err := redis.Strings(conn.Do("SMEMBERS", relationKey))
	if err != nil {
		return err
	}

	field := mr.value(relationship.fieldName)
	rType := field.Type().Elem()

	// iterate through the ids and find each model
	for _, id := range ids {

		// check if a model with key is already cached in this transaction
		rModelName, _ := getRegisteredNameFromType(rType)
		rModelKey := rModelName + ":" + id
		if prior, found := p.t.modelCache[rModelKey]; found {
			// use the same pointer (it's the same object)
			sliceVal := reflect.Append(field, reflect.ValueOf(prior))
			field.Set(sliceVal)
			continue
		}

		rVal := reflect.New(rType.Elem())
		rModel, ok := rVal.Interface().(Model)
		if !ok {
			return fmt.Errorf("zoom: cannot convert type %s to Model\n", rType.String())
		}

		// set id and create modelRef
		rModel.SetId(id)
		rModelRef, err := newModelRefFromModel(rModel)
		if err != nil {
			return err
		}

		// add a find operation to the transaction
		if err := p.scanModel(rModelRef, nil); err != nil {
			return err
		}

		// append to the field slice
		sliceVal := reflect.Append(field, rVal)
		field.Set(sliceVal)
	}

	return nil
}

func (p *phase) deleteModel(mr modelRef) {
	modelName := mr.modelSpec.modelName
	id := mr.model.GetId()

	// add an operation to delete the model itself
	key := modelName + ":" + id
	p.delete(key)

	// add an operation to remove the model id from the index
	indexKey := modelName + ":all"
	p.unindex(indexKey, id)

	// add an operation to remove all the field indexes for the model
	p.removeModelIndexes(mr)
}

func (p *phase) deleteModelById(modelName, id string) error {

	ms, found := modelSpecs[modelName]
	if !found {
		return NewModelNameNotRegisteredError(modelName)
	}

	// add an operation to remove all the field indexes for the model
	// we want to do this first because if there is an error or if the model
	// never existed, there is no need to continue
	if len(ms.primativeIndexes) != 0 || len(ms.pointerIndexes) != 0 {
		m, err := FindById(modelName, id)
		if err != nil {
			if _, ok := err.(*KeyNotFoundError); ok {
				// if it was a key not found error, the model we're trying to delete
				// doesn't exist in the first place. so return nil
				return nil
			} else {
				// this means there was an unexpected error
				return err
			}
		}
		mr, err := newModelRefFromModel(m)
		if err != nil {
			return err
		}
		p.removeModelIndexes(mr)
	}

	// add an operation to delete the model itself
	key := modelName + ":" + id
	p.delete(key)

	// add an operation to remove the model id from the index
	indexKey := modelName + ":all"
	p.unindex(indexKey, id)

	return nil
}

func (p *phase) delete(key string) {
	p.addCommand("DEL", redis.Args{}.Add(key), nil)
}

func (p *phase) unindex(key, value string) {
	args := redis.Args{}.Add(key).Add(value)
	p.addCommand("SREM", args, nil)
}

func (p *phase) removeModelIndexes(mr modelRef) {
	for _, pi := range mr.modelSpec.primativeIndexes {
		if pi.indexType == indexNumeric {
			p.removeModelPrimativeIndexNumeric(mr, pi)
		} else if pi.indexType == indexAlpha {
			p.removeModelPrimativeIndexAlpha(mr, pi)
		} else if pi.indexType == indexBoolean {
			p.removeModelPrimativeIndexBoolean(mr, pi)
		}
	}

	for _, pi := range mr.modelSpec.pointerIndexes {
		if pi.indexType == indexNumeric {
			p.removeModelPointerIndexNumeric(mr, pi)
		} else if pi.indexType == indexAlpha {
			p.removeModelPointerIndexAlpha(mr, pi)
		} else if pi.indexType == indexBoolean {
			p.removeModelPointerIndexBoolean(mr, pi)
		}
	}
}

func (p *phase) removeModelPrimativeIndexNumeric(mr modelRef, primative *fieldSpec) {
	indexKey := mr.modelSpec.modelName + ":" + primative.redisName
	id := mr.model.GetId()
	p.unindexNumeric(indexKey, id)
}

func (p *phase) removeModelPointerIndexNumeric(mr modelRef, pointer *fieldSpec) {
	if mr.value(pointer.fieldName).IsNil() {
		// TODO: special case for indexing nil pointers?
		return // skip nil pointers for now
	}
	indexKey := mr.modelSpec.modelName + ":" + pointer.redisName
	id := mr.model.GetId()
	p.unindexNumeric(indexKey, id)
}

func (p *phase) unindexNumeric(indexKey string, id string) {
	args := redis.Args{}.Add(indexKey).Add(id)
	p.addCommand("ZREM", args, nil)
}

func (p *phase) removeModelPrimativeIndexAlpha(mr modelRef, primative *fieldSpec) {
	indexKey := mr.modelSpec.modelName + ":" + primative.redisName
	value := mr.value(primative.fieldName).String()
	id := mr.model.GetId()
	p.unindexAlpha(indexKey, value, id)
}

func (p *phase) removeModelPointerIndexAlpha(mr modelRef, pointer *fieldSpec) {
	if mr.value(pointer.fieldName).IsNil() {
		// TODO: special case for indexing nil pointers?
		return // skip nil pointers for now
	}
	indexKey := mr.modelSpec.modelName + ":" + pointer.redisName
	value := mr.value(pointer.fieldName).Elem().String()
	id := mr.model.GetId()
	p.unindexAlpha(indexKey, value, id)
}

func (p *phase) unindexAlpha(indexKey, value, id string) {
	member := value + " " + id
	args := redis.Args{}.Add(indexKey).Add(member)
	p.addCommand("ZREM", args, nil)
}

func (p *phase) removeModelPrimativeIndexBoolean(mr modelRef, primative *fieldSpec) {
	id := mr.model.GetId()
	indexKey := mr.modelSpec.modelName + ":" + primative.redisName
	p.unindexNumeric(indexKey, id)
}

func (p *phase) removeModelPointerIndexBoolean(mr modelRef, pointer *fieldSpec) {
	id := mr.model.GetId()
	indexKey := mr.modelSpec.modelName + ":" + pointer.redisName
	p.unindexNumeric(indexKey, id)
}

// check to see if the model id exists in the index. If it doesn't,
// return KeyNotFoundError
func checkModelExists(mr modelRef) error {
	conn := GetConn()
	defer conn.Close()

	indexKey := mr.modelSpec.modelName + ":all"
	if exists, err := redis.Bool(conn.Do("SISMEMBER", indexKey, mr.model.GetId())); err != nil {
		return err
	} else if !exists {
		return NewKeyNotFoundError(mr.key(), mr.modelSpec.modelType)
	}
	return nil
}
