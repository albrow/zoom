package zoom

import (
	"github.com/stephenalexbrowne/zoom/redis"
	"reflect"
)

type transaction struct {
	conn     redis.Conn
	handlers []func(interface{}) error
}

func newTransaction() *transaction {
	t := &transaction{
		conn: GetConn(),
	}
	t.conn.Send("MULTI")
	return t
}

func (t *transaction) add(cmd string, args []interface{}, handler func(interface{}) error) error {
	if err := t.conn.Send(cmd, args...); err != nil {
		t.discard()
		return err
	}
	t.handlers = append(t.handlers, handler)
	return nil
}

func (t *transaction) exec() error {
	defer t.conn.Close()
	replies, err := redis.MultiBulk(t.conn.Do("EXEC"))
	if err != nil {
		t.discard()
		return err
	}
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

// useful handlers
func newScanStructHandler(scannable interface{}) func(interface{}) error {
	return func(reply interface{}) error {
		// invoke redis driver to scan values into the data struct
		bulk, err := redis.MultiBulk(reply, nil)
		if err != nil {
			return err
		}
		if err := redis.ScanStruct(bulk, scannable); err != nil {
			return err
		}
		return nil
	}
}

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

// useful operations for transactions
func (t *transaction) addModelSave(m Model) error {

	name, err := getRegisteredNameFromInterface(m)
	if err != nil {
		return err
	}

	// set the id if needed
	if m.GetId() == "" {
		m.SetId(generateRandomId())
	}

	// add a command to write data to database
	key := name + ":" + m.GetId()
	if err := t.addStructSave(key, m); err != nil {
		return err
	}

	// add a command to add to index for this model
	indexKey := name + ":index"
	if err := t.addIndex(indexKey, m.GetId()); err != nil {
		return err
	}

	return nil
}

func (t *transaction) addStructSave(key string, in interface{}) error {
	args := redis.Args{}.Add(key).AddFlat(in)
	if err := t.add("HMSET", args, nil); err != nil {
		return err
	}

	return nil
}

func (t *transaction) addIndex(key, value string) error {
	args := redis.Args{}.Add(key).Add(value)
	if err := t.add("SADD", args, nil); err != nil {
		return err
	}

	return nil
}

func (t *transaction) addModelFind(key, scannable interface{}) error {
	if err := t.add("HGETALL", redis.Args{}.Add(key), newScanStructHandler(scannable)); err != nil {
		return err
	}

	return nil
}

func (t *transaction) addDelete(key string) error {
	if err := t.add("DEL", redis.Args{}.Add(key), nil); err != nil {
		return err
	}

	return nil
}

func (t *transaction) addUnindex(key, value string) error {
	args := redis.Args{}.Add(key).Add(value)
	if err := t.add("SREM", args, nil); err != nil {
		return err
	}

	return nil
}
