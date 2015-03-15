// Copyright 2014 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File transaction.go contains code related to the
// transactions abstraction.

package zoom

import (
	"github.com/garyburd/redigo/redis"
)

// transaction is an abstraction layer around a redis transaction.
// transactions feature delayed execution, so nothing touches the database
// until exec is called.
type transaction struct {
	conn    redis.Conn
	actions []*action
}

// action is a single step in a transaction and must be either a command
// or a script with optional arguments.
type action struct {
	kind    actionKind
	name    string
	script  *redis.Script
	args    redis.Args
	handler replyHandler
}

// actionKind is either a command or a script
type actionKind int

const (
	actionCommand actionKind = iota
	actionScript
)

// replyHandler is a function which does something with the reply from a redis
// command or script.
type replyHandler func(interface{}) error

// newTransaction instantiates and returns a new transaction.
func newTransaction() *transaction {
	t := &transaction{
		conn: GetConn(),
	}
	return t
}

// command adds a command action to the transaction with the given args.
// handler will be called with the reply from this specific command when
// the transaction is executed.
func (t *transaction) command(name string, args redis.Args, handler replyHandler) {
	t.actions = append(t.actions, &action{
		kind:    actionCommand,
		name:    name,
		args:    args,
		handler: handler,
	})
}

// command adds a script action to the transaction with the given args.
// handler will be called with the reply from this specific script when
// the transaction is executed.
func (t *transaction) script(script *redis.Script, args redis.Args, handler replyHandler) {
	t.actions = append(t.actions, &action{
		kind:    actionScript,
		script:  script,
		args:    args,
		handler: handler,
	})
}

// sendAction writes a to a connection buffer using conn.Send()
func (t *transaction) sendAction(a *action) error {
	switch a.kind {
	case actionCommand:
		return t.conn.Send(a.name, a.args...)
	case actionScript:
		return a.script.Send(t.conn, a.args...)
	}
	return nil
}

// doAction writes a to the connection buffer and then immediately
// flushes the buffer and reads the reply via conn.Do()
func (t *transaction) doAction(a *action) (interface{}, error) {
	switch a.kind {
	case actionCommand:
		return t.conn.Do(a.name, a.args...)
	case actionScript:
		return a.script.Do(t.conn, a.args...)
	}
	return nil, nil
}

// exec executes the transaction, sequentially sending each action and
// calling all the action handlers with the corresponding replies.
func (t *transaction) exec() error {
	// Return the connection to the pool when we are done
	defer t.conn.Close()

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
		t.conn.Send("MULTI")
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
