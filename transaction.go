// Copyright 2015 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File transaction.go contains code related to the
// transactions abstraction.

package zoom

import (
	"fmt"
	"reflect"

	"github.com/garyburd/redigo/redis"
)

// Transaction is an abstraction layer around a redis transaction.
// Transactions consist of a set of actions which are either redis
// commands or lua scripts. Transactions feature delayed execution,
// so nothing toches the database until you call Exec.
type Transaction struct {
	conn    redis.Conn
	actions []*Action
	err     error
}

// Action is a single step in a transaction and must be either a command
// or a script with optional arguments and a reply handler.
type Action struct {
	kind    ActionKind
	name    string
	script  *redis.Script
	args    redis.Args
	handler ReplyHandler
}

// ActionKind is either a command or a script
type ActionKind int

const (
	CommandAction ActionKind = iota
	ScriptAction
)

// ReplyHandler is a function which does something with the reply from a redis
// command or script.
type ReplyHandler func(interface{}) error

// NewTransaction instantiates and returns a new transaction.
func (p *Pool) NewTransaction() *Transaction {
	t := &Transaction{
		conn: p.NewConn(),
	}
	return t
}

// SetError sets the err property of the transaction iff it was not already
// set. This will cause exec to fail immediately.
func (t *Transaction) setError(err error) {
	if t.err == nil {
		t.err = err
	}
}

// Command adds a command action to the transaction with the given args.
// handler will be called with the reply from this specific command when
// the transaction is executed.
func (t *Transaction) Command(name string, args redis.Args, handler ReplyHandler) {
	t.actions = append(t.actions, &Action{
		kind:    CommandAction,
		name:    name,
		args:    args,
		handler: handler,
	})
}

// Script adds a script action to the transaction with the given args.
// handler will be called with the reply from this specific script when
// the transaction is executed.
func (t *Transaction) Script(script *redis.Script, args redis.Args, handler ReplyHandler) {
	t.actions = append(t.actions, &Action{
		kind:    ScriptAction,
		script:  script,
		args:    args,
		handler: handler,
	})
}

// sendAction writes a to a connection buffer using conn.Send()
func (t *Transaction) sendAction(a *Action) error {
	switch a.kind {
	case CommandAction:
		return t.conn.Send(a.name, a.args...)
	case ScriptAction:
		return a.script.Send(t.conn, a.args...)
	}
	return nil
}

// doAction writes a to the connection buffer and then immediately
// flushes the buffer and reads the reply via conn.Do()
func (t *Transaction) doAction(a *Action) (interface{}, error) {
	switch a.kind {
	case CommandAction:
		return t.conn.Do(a.name, a.args...)
	case ScriptAction:
		return a.script.Do(t.conn, a.args...)
	}
	return nil, nil
}

// Exec executes the transaction, sequentially sending each action and
// calling all the action handlers with the corresponding replies.
func (t *Transaction) Exec() error {
	// Return the connection to the pool when we are done
	defer t.conn.Close()

	// If the transaction had an error from a previous command, return it
	// and don't continue
	if t.err != nil {
		return t.err
	}

	if len(t.actions) == 1 {
		// If there is only one command, no need to use MULTI/EXEC
		a := t.actions[0]
		reply, err := t.doAction(a)
		if err != nil {
			return err
		}
		if a.handler != nil {
			if err := a.handler(reply); err != nil {
				return err
			}
		}
	} else {
		// Send all the commands and scripts at once using MULTI/EXEC
		if err := t.conn.Send("MULTI"); err != nil {
			return err
		}
		for _, a := range t.actions {
			if err := t.sendAction(a); err != nil {
				return err
			}
		}

		// Invoke redis driver to execute the transaction
		replies, err := redis.Values(t.conn.Do("EXEC"))
		if err != nil {
			return err
		}

		// Iterate through the replies, calling the corresponding handler functions
		for i, reply := range replies {
			a := t.actions[i]
			if a.handler != nil {
				if err := a.handler(reply); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// newScanIntHandler returns a ReplyHandler which will set the value of i to the
// converted value of reply.
func newScanIntHandler(i *int) ReplyHandler {
	return func(reply interface{}) error {
		var err error
		(*i), err = redis.Int(reply, nil)
		if err != nil {
			return err
		}
		return nil
	}
}

// newScanIntHandler returns a ReplyHandler which will set the value of b to the
// converted value of reply.
func newScanBoolHandler(b *bool) ReplyHandler {
	return func(reply interface{}) error {
		var err error
		(*b), err = redis.Bool(reply, nil)
		if err != nil {
			return err
		}
		return nil
	}
}

// newScanIntHandler returns a ReplyHandler which will set the value of s to the
// converted value of reply.
func newScanStringHandler(s *string) ReplyHandler {
	return func(reply interface{}) error {
		var err error
		(*s), err = redis.String(reply, nil)
		if err != nil {
			return err
		}
		return nil
	}
}

// newScanStringsHandler returns a reply handler which will scan all the replies
// in reply into strings. strings should be a pointer to a slice of some strings.
// The returned replyHandler will grow or shrink strings as needed.
func newScanStringsHandler(strings interface{}) ReplyHandler {
	return func(reply interface{}) error {
		replyStrings, err := redis.Strings(reply, nil)
		if err != nil {
			return err
		}
		stringsVal := reflect.ValueOf(strings).Elem()
		stringsVal.Set(reflect.ValueOf(replyStrings))
		return nil
	}
}

// newScanModelHandler returns a ReplyHandler which will scan all the fields in
// in the reply which are also in fieldNames into the fields of mr.model. It expects
// a reply that looks like the output of an HMGET command, without the field names
// included. The order of fieldNames must correspond to the order of the values in the
// reply. fieldNames should be the actual field names as they appear in the struct
// definition, not the redis names which may be custom.
func newScanModelHandler(fieldNames []string, mr *modelRef) ReplyHandler {
	return func(reply interface{}) error {
		fieldValues, err := redis.Values(reply, nil)
		if err != nil {
			if err == redis.ErrNil {
				return newModelNotFoundError(mr)
			}
			return err
		}
		if err := scanModel(fieldNames, fieldValues, mr); err != nil {
			return err
		}
		return nil
	}
}

// newScanModelsHandler returns a reply handler which will scan all the replies
// in reply into models. It only scans the fields which are in fieldNames. models
// should be a pointer to a slice of some registered model type. The returned
// replyHandler will grow or shrink models as needed. It expects a reply which is
// a flat array of field values, with no separation between the fields for each
// model. The order of fieldNames must correspond to the order of the values in the
// reply. fieldNames should be the actual field names as they appear in the struct
// definition, not the redis names which may be custom. It will use the length of
// fieldNames to determine which fields belong to which models. For example, if
// fieldNames is ["Int", "String", "-"] and the reply from redis looks like this:
//
//   1) "5"
//   2) "Bob"
//   3) "b1C7B0yETtXFYuKinndqoa"
//   4) "7"
//   5) "Alice"
//   6) "NmjzzzDyNJpsCpPKnndqoa"
//
// newScanModelsHandler will recognize that there are three fields and assign the
// first three to the first model, the next three to the second model, etc. This
// is the kind of reply we expect from a SORT command with multiple GET options.
func newScanModelsHandler(spec *modelSpec, fieldNames []string, models interface{}) ReplyHandler {
	return func(reply interface{}) error {
		allFields, err := redis.Values(reply, nil)
		if err != nil {
			if err == redis.ErrNil {
				return ModelNotFoundError{
					Msg: fmt.Sprintf("Could not find %s with the given criteria", spec.name),
				}
			}
			return err
		}
		numFields := len(fieldNames)
		numModels := len(allFields) / numFields
		modelsVal := reflect.ValueOf(models).Elem()
		for i := 0; i < numModels; i++ {
			start := i * numFields
			stop := i*numFields + numFields
			fieldValues := allFields[start:stop]
			var modelVal reflect.Value
			if modelsVal.Len() > i {
				// Use the pre-existing value at index i
				modelVal = modelsVal.Index(i)
				if modelVal.IsNil() {
					// If the value is nil, allocate space for it
					modelsVal.Index(i).Set(reflect.New(spec.typ.Elem()))
				}
			} else {
				// Index i is out of range of the existing slice. Create a
				// new modelVal and append it to modelsVal
				modelVal = reflect.New(spec.typ.Elem())
				modelsVal.Set(reflect.Append(modelsVal, modelVal))
			}
			mr := &modelRef{
				spec:  spec,
				model: modelVal.Interface().(Model),
			}
			if err := scanModel(fieldNames, fieldValues, mr); err != nil {
				return err
			}
		}
		// Trim the slice if it is longer than the number of models we scanned
		// in.
		if numModels < modelsVal.Len() {
			modelsVal.SetLen(numModels)
			modelsVal.SetCap(numModels)
		}
		return nil
	}
}

//go:generate go run scripts/main.go

// deleteModelsBySetIds is a small function wrapper around deleteModelsBySetIdsScript.
// It offers some type safety and helps make sure the arguments you pass through to the are correct.
// The script will delete the models corresponding to the ids in the given set and return the number
// of models that were deleted. You can use the handler to capture the return value.
func (t *Transaction) deleteModelsBySetIds(setKey string, modelName string, handler ReplyHandler) {
	t.Script(deleteModelsBySetIdsScript, redis.Args{setKey, modelName}, handler)
}

// deleteStringIndex is a small function wrapper around deleteStringIndexScript.
// It offers some type safety and helps make sure the arguments you pass through to the are correct.
// The script will atomically remove the existing index, if any, on the given field name.
func (t *Transaction) deleteStringIndex(modelName, modelId, fieldName string) {
	t.Script(deleteStringIndexScript, redis.Args{modelName, modelId, fieldName}, nil)
}

// extractIdsFromFieldIndex is a small function wrapper around extractIdsFromFieldIndexScript.
// It offers some type safety and helps make sure the arguments you pass through to the are correct.
// The script will get all the ids from setKey using ZRANGEBYSCORE with the given min and max, and then
// store them in a sorted set identified by destKey.
func (t *Transaction) extractIdsFromFieldIndex(setKey string, destKey string, min interface{}, max interface{}) {
	t.Script(extractIdsFromFieldIndexScript, redis.Args{setKey, destKey, min, max}, nil)
}

// extractIdsFromStringIndex is a small function wrapper around extractIdsFromStringIndexScript.
// It offers some type safety and helps make sure the arguments you pass through to the are correct.
// The script will extract the ids from setKey using ZRANGEBYLEX with the given min and max, and then
// store them in a sorted set identified by destKey.
func (t *Transaction) extractIdsFromStringIndex(setKey, destKey, min, max string) {
	t.Script(extractIdsFromStringIndexScript, redis.Args{setKey, destKey, min, max}, nil)
}
