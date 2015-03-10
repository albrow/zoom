// Copyright 2014 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File transaction.go contains all code dealing with the
// transactions abstraction, including construction, adding
// commands, and execution of the transaction.

package zoom

import (
	"container/list"
	"fmt"
	"github.com/garyburd/redigo/redis"
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
	phases     *list.List
	modelCache map[string]interface{}
}

func newTransaction() *transaction {
	return &transaction{
		phases:     list.New(),
		conn:       GetConn(),
		modelCache: map[string]interface{}{},
	}
}

func (t *transaction) exec() error {
	for e := t.phases.Front(); e != nil; e = e.Next() {
		phase, ok := e.Value.(*phase)
		if !ok {
			return fmt.Errorf("zoom: in transaction.exec(): Could not convert %v of type %T to *zoom.phase", e.Value, e.Value)
		}
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
	deps     []*phase
	pre      func(*phase) error
	post     func(*phase) error
}

func (t *transaction) addPhase(id string, pre func(*phase) error, post func(*phase) error) (*phase, error) {
	if id == "" {
		id = generateRandomId()
	}
	for e := t.phases.Front(); e != nil; e = e.Next() {
		existingPhase, ok := e.Value.(*phase)
		if !ok {
			return nil, fmt.Errorf("zoom: in addPhase(): Could not convert %v of type %T to *zoom.phase", e.Value, e.Value)
		}
		if id == existingPhase.id {
			panic("phase id already exists")
			return nil, fmt.Errorf("zoom: in addPhase(): Phase with id = %s already exists.", id)
		}
	}
	p := &phase{
		t:    t,
		id:   id,
		pre:  pre,
		post: post,
	}
	t.phases.PushBack(p)
	return p, nil
}

func (t *transaction) phaseById(id string) (*phase, bool) {
	for e := t.phases.Front(); e != nil; e = e.Next() {
		p, ok := e.Value.(*phase)
		if !ok {
			msg := fmt.Sprintf("zoom: in addPhase(): Could not convert %v of type %T to *zoom.phase", e.Value, e.Value)
			panic(msg)
		}
		if id == p.id {
			return p, true
		}
	}
	return nil, false
}

func (t *transaction) phaseIds() []string {
	ids := []string{}
	for e := t.phases.Front(); e != nil; e = e.Next() {
		p, ok := e.Value.(*phase)
		if !ok {
			msg := fmt.Sprintf("zoom: in transaction.phaseIds(): Could not convert %v of type %T to *zoom.phase", e.Value, e.Value)
			panic(msg)
		}
		ids = append(ids, p.id)
	}
	return ids
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
		if len(p.commands) == 1 {
			// no need to use MULTI/EXEC for just one command
			c := p.commands[0]
			reply, err := conn.Do(c.name, c.args...)
			if err != nil {
				return err
			}
			if c.handler != nil {
				if err := c.handler(reply); err != nil {
					return err
				}
			}
		} else {
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
	}

	// execute the post-handler for this phase
	if p.post != nil {
		return p.post(p)
	}
	return nil
}

func (p *phase) addDependency(dep *phase) error {
	if p.id == dep.id {
		return fmt.Errorf("zoom: in phase.addDependency: Cannot add a phase as dependent on itself!")
	}

	var pEl, depEl *list.Element
	canMoveBefore, canMoveAfter := true, true
	for e := p.t.phases.Front(); e != nil; e = e.Next() {
		currentPhase, ok := e.Value.(*phase)
		if !ok {
			return fmt.Errorf("zoom: in phase.addDependency(): Could not convert %v of type %T to *zoom.phase", e.Value, e.Value)
		}
		switch currentPhase.id {
		case p.id:
			// We found p
			if depEl != nil {
				// If we already found dep, that means dep came before p
				// in the list and the dependency is already satisfied. We
				// don't need to change the order.
				p.deps = append(p.deps, dep)
				return nil
			}
			// If we're here it means we haven't found dep yet. The dependency
			// may be satisfiable but we'll have to move things around. We set
			// p and pEl here, so when we find dep we can figure out what to move
			// and where to move it. If we never find dep, we'll reach the end of
			// the function and return an error.
			pEl = e
		case dep.id:
			// We found dep
			depEl = e
			if pEl != nil {
				// We found p but it was not before dep in the list. That means we'll
				// need to move some things around.
				p.deps = append(p.deps, dep)

				// If p depends on dep and dep depends on p, we have a pretty clear cycle
				for _, depdep := range dep.deps {
					if depdep.id == p.id {
						return NewDependencyCycleError(p, dep)
					}
				}

				if canMoveAfter {
					// If we reached here, we found a placement that works!
					// We can move p directly after dep
					p.t.phases.MoveAfter(pEl, depEl)
					return nil
				} else if canMoveBefore {
					// If we reached here, we found a placement that works!
					// We can move dep directly before p
					p.t.phases.MoveBefore(depEl, pEl)
					return nil
				}
			}
		default:
			if pEl != nil {
				// If pEl is not nil, then it must mean that we found p and
				// are now moving through the list until we find dep. These
				// elements are the elements in between p and dep.

				if canMoveAfter {
					// Check if moving p after dep is still a possibility
					for _, dep := range currentPhase.deps {
						if dep.id == p.id {
							// If any of the phases in between p and dep depend on
							// p, we cannot move p after dep, because it would break
							// those dependencies
							canMoveAfter = false
						}
					}
				}

				if canMoveBefore {
					// Check if moving dep before p is still a possiblity
					for _, depdep := range dep.deps {
						if depdep.id == currentPhase.id {
							// If dep depends on any of the phases in between p and dep,
							// then we cannot move dep before p, because that would break
							// the dependencies.
							canMoveBefore = false
						}
					}
				}

				if !canMoveBefore && !canMoveAfter {
					// If we've reached here, we cannot move p direcly after dep
					// or dep directly before p, so we must have a cycle.
					return NewDependencyCycleError(p, dep)
				}
			}
		}
	}

	if pEl == nil {
		return fmt.Errorf("zoom: in phase.addDependency(): Could not find phase with id = %s", p.id)
	}
	if depEl == nil {
		return fmt.Errorf("zoom: in phase.addDependency(): Could not find phase with id = %s", dep.id)
	}
	return nil
}

func (p *phase) depIds() []string {
	ids := []string{}
	for _, dep := range p.deps {
		ids = append(ids, dep.id)
	}
	return ids
}

// anyDependsOn returns true iff any phase in phases depends on p
func anyDependsOn(phases []*phase, p *phase) bool {
	for _, phase := range phases {
		for _, dep := range phase.deps {
			if dep.id == p.id {
				return true
			}
		}
	}
	return false
}

// dependsOnAny returns true iff phase depends on any phases
func dependsOnAny(p *phase, phases []*phase) bool {
	for _, phase := range phases {
		for _, dep := range p.deps {
			if dep.id == phase.id {
				return true
			}
		}
	}
	return false
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
			// This would mean the model was not found
			return NewKeyNotFoundError(mr.key(), mr.modelSpec.modelType)
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
			// This would mean the model was not found
			return NewKeyNotFoundError(mr.key(), mr.modelSpec.modelType)
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
// This includes indexes.
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

func (p *phase) saveModelIndexes(mr modelRef) error {
	for _, pi := range mr.modelSpec.primativeIndexes {
		if pi.indexType == indexNumeric {
			if err := p.saveModelPrimativeIndexNumeric(mr, pi); err != nil {
				return err
			}
		} else if pi.indexType == indexString {
			p.saveModelPrimativeIndexString(mr, pi)
		} else if pi.indexType == indexBoolean {
			p.saveModelPrimativeIndexBoolean(mr, pi)
		}
	}

	for _, pi := range mr.modelSpec.pointerIndexes {
		if pi.indexType == indexNumeric {
			if err := p.saveModelPointerIndexNumeric(mr, pi); err != nil {
				return err
			}
		} else if pi.indexType == indexString {
			p.saveModelPointerIndexString(mr, pi)
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

func (p *phase) saveModelPrimativeIndexString(mr modelRef, primative *fieldSpec) {
	p.removeOldStringIndex(mr, primative.fieldName, primative.redisName)
	indexKey := mr.modelSpec.modelName + ":" + primative.redisName
	value := mr.value(primative.fieldName).String()
	id := mr.model.GetId()
	p.indexString(indexKey, value, id)
}

func (p *phase) saveModelPointerIndexString(mr modelRef, pointer *fieldSpec) {
	p.removeOldStringIndex(mr, pointer.fieldName, pointer.redisName)
	if mr.value(pointer.fieldName).IsNil() {
		// TODO: special case for indexing nil pointers?
		return // skip nil pointers for now
	}
	indexKey := mr.modelSpec.modelName + ":" + pointer.redisName
	value := mr.value(pointer.fieldName).Elem().String()
	id := mr.model.GetId()
	p.indexString(indexKey, value, id)
}

func (p *phase) indexString(indexKey, value, id string) {
	member := value + " " + id
	args := redis.Args{}.Add(indexKey).Add(0).Add(member)
	p.addCommand("ZADD", args, nil)
}

// Remove the string index that may have existed before an update or resave of the model
// this requires a read before write, which is a performance penalty but unfortunatlely
// is unavoidable because of the hacky way we're indexing string fields.
func (p *phase) removeOldStringIndex(mr modelRef, fieldName string, redisName string) {
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
			stringIndexKey := mr.modelSpec.modelName + ":" + fieldName
			member := oldFieldValue + " " + mr.model.GetId()
			if _, err := conn.Do("ZREM", stringIndexKey, member); err != nil {
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
		} else if pi.indexType == indexString {
			p.removeModelPrimativeIndexString(mr, pi)
		} else if pi.indexType == indexBoolean {
			p.removeModelPrimativeIndexBoolean(mr, pi)
		}
	}

	for _, pi := range mr.modelSpec.pointerIndexes {
		if pi.indexType == indexNumeric {
			p.removeModelPointerIndexNumeric(mr, pi)
		} else if pi.indexType == indexString {
			p.removeModelPointerIndexString(mr, pi)
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

func (p *phase) removeModelPrimativeIndexString(mr modelRef, primative *fieldSpec) {
	indexKey := mr.modelSpec.modelName + ":" + primative.redisName
	value := mr.value(primative.fieldName).String()
	id := mr.model.GetId()
	p.unindexString(indexKey, value, id)
}

func (p *phase) removeModelPointerIndexString(mr modelRef, pointer *fieldSpec) {
	if mr.value(pointer.fieldName).IsNil() {
		// TODO: special case for indexing nil pointers?
		return // skip nil pointers for now
	}
	indexKey := mr.modelSpec.modelName + ":" + pointer.redisName
	value := mr.value(pointer.fieldName).Elem().String()
	id := mr.model.GetId()
	p.unindexString(indexKey, value, id)
}

func (p *phase) unindexString(indexKey, value, id string) {
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
