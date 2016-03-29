// Copyright 2015 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File transaction.go contains code related to the
// transactions abstraction.

package zoom

import (
	"fmt"

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
			if err, ok := reply.(error); ok {
				return err
			}
			if a.handler != nil {
				if err := a.handler(reply); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

//go:generate go run scripts/main.go

// DeleteModelsBySetIds is a small function wrapper around a Lua script. The
// script will atomically delete the models corresponding to the ids in set
// (not sorted set) identified by setKey and return the number of models that
// were deleted. You can pass in a handler (e.g. NewScanIntHandler) to capture
// the return value of the script. You can use the Name method of a Collection
// to get the name.
func (t *Transaction) DeleteModelsBySetIds(setKey string, collectionName string, handler ReplyHandler) {
	t.Script(deleteModelsBySetIdsScript, redis.Args{setKey, collectionName}, handler)
}

// deleteStringIndex is a small function wrapper around a Lua script. The script
// will atomically remove the existing string index, if any, on the given
// fieldName for the model with the given modelId. You can use the Name method
// of a Collection to get its name. fieldName should be the name as it is stored
// in Redis.
func (t *Transaction) deleteStringIndex(collectionName, modelId, fieldName string) {
	t.Script(deleteStringIndexScript, redis.Args{collectionName, modelId, fieldName}, nil)
}

// ExtractIdsFromFieldIndex is a small function wrapper around a Lua script. The
// script will get all the ids from the sorted set identified by setKey using
// ZRANGEBYSCORE with the given min and max, and then store them in a sorted set
// identified by destKey. The members of the sorted set should be model ids.
// Note that this method will not work on sorted sets that represents string
// indexes because they are stored differently.
func (t *Transaction) ExtractIdsFromFieldIndex(setKey string, destKey string, min interface{}, max interface{}) {
	t.Script(extractIdsFromFieldIndexScript, redis.Args{setKey, destKey, min, max}, nil)
}

// ExtractIdsFromStringIndex is a small function wrapper around a Lua script.
// The script will extract the ids from a sorted set identified by setKey using
// ZRANGEBYLEX with the given min and max, and then store them in a sorted set
// identified by destKey. All the scores for the sorted set should be 0, and the
// members should follow the format <value>\x00<id>, where <value> is the string
// value, \x000 is the NULL ASCII character and <id> is the id of the model
// with that value. As with all string indexes in Zoom, the value cannot contain
// the NULL ASCII character or the DEL character (codepoints 0 and 127
// respectively). Note that the stored ids are sorted in ASCII order according
// to their corresponding string values.
func (t *Transaction) ExtractIdsFromStringIndex(setKey, destKey, min, max string) {
	t.Script(extractIdsFromStringIndexScript, redis.Args{setKey, destKey, min, max}, nil)
}

func (t *Transaction) FindModelsByIdsKey(collection *Collection, idsKey string, fieldNames []string, limit uint, offset uint, reverse bool, models interface{}) {
	if err := collection.checkModelsType(models); err != nil {
		t.setError(fmt.Errorf("zoom: error in FindModelsByIdKey: %s", err.Error()))
		return
	}
	redisNames, err := collection.spec.redisNamesForFieldNames(fieldNames)
	if err != nil {
		t.setError(fmt.Errorf("zoom: error in FindModelsByIdKey: %s", err.Error()))
		return
	}
	sortArgs := collection.spec.sortArgs(idsKey, redisNames, int(limit), offset, reverse)
	t.Command("SORT", sortArgs, newScanModelsHandler(collection.spec, append(fieldNames, "-"), models))
}
