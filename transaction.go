// Copyright 2014 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File transaction.go contains code related to the
// transactions abstraction.

package zoom

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"reflect"
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
// command or script. Each ReplyHandler is executed immediately after its
// corresponding script or command is run.
type ReplyHandler func(interface{}) error

// NewTransaction instantiates and returns a new transaction.
func NewTransaction() *Transaction {
	t := &Transaction{
		conn: Conn(),
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
// reply into the fields of mr.model.
func newScanModelHandler(mr *modelRef) ReplyHandler {
	return func(reply interface{}) error {
		replies, err := redis.Values(reply, nil)
		if err != nil {
			return err
		}
		if len(replies) == 0 {
			var msg string
			if mr.model.Id() != "" {
				msg = fmt.Sprintf("Could not find %s with id = %s", mr.spec.name, mr.model.Id())
			} else {
				msg = fmt.Sprintf("Could not find %s with the given criteria", mr.spec.name)
			}
			return ModelNotFoundError{Msg: msg}
		}
		if err := scanModel(replies, mr); err != nil {
			return err
		}
		return nil
	}
}

// newScanModelsHandler returns a reply handler which will scan all the replies
// in reply into models. models should be a pointer to a slice of some registered model
// type. The returned replyHandler will grow or shrink models as needed.
func newScanModelsHandler(spec *modelSpec, models interface{}) ReplyHandler {
	return func(reply interface{}) error {
		modelsFields, err := redis.Values(reply, nil)
		if err != nil {
			return err
		}
		modelsVal := reflect.ValueOf(models).Elem()
		for i, reply := range modelsFields {
			fields, err := redis.Values(reply, nil)
			if err != nil {
				return err
			}
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
			if err := scanModel(fields, mr); err != nil {
				return err
			}
		}
		// Trim the slice if it is longer than the number of models we scanned
		// in.
		if len(modelsFields) < modelsVal.Len() {
			modelsVal.SetLen(len(modelsFields))
			modelsVal.SetCap(len(modelsFields))
		}
		return nil
	}
}
