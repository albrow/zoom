package zoom

import (
	"github.com/garyburd/redigo/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"testing"
)

func TestWatch(t *testing.T) {
	testingSetUp()
	defer testingTearDown()
	model := &testModel{
		Int:    42,
		String: "foo",
		Bool:   true,
	}
	require.NoError(t, testModels.Save(model))
	tx := testPool.NewTransaction()
	// Issue a WATCH command
	require.NoError(t, tx.Watch(model))
	// Update the model directly using a different connection. This should
	// trigger WATCH
	expectedString := "bar"
	model.String = expectedString
	require.NoError(t, testModels.Save(model))
	// Try to update the model using the transaction. We expect this to fail
	// and return a WatchError
	tx.Command("HSET", redis.Args{testModels.ModelKey(model.ID), "Int", 35}, nil)
	err := tx.Exec()
	assert.Error(t, err)
	assert.IsType(t, WatchError{}, err)
	// Finally retrieve the model to make sure the changes in the transaction
	// were not committed.
	other := testModel{}
	require.NoError(t, testModels.Find(model.ID, &other))
	assert.Equal(t, expectedString, other.String, "First update *was not* committed")
	assert.Equal(t, model.Int, other.Int, "Second update *was* committed")
}

func TestWatchKey(t *testing.T) {
	testingSetUp()
	defer testingTearDown()
	conn1 := testPool.NewConn()
	defer func() {
		_ = conn1.Close()
	}()
	key := "mykey"
	_, err := conn1.Do("SET", key, "foo")
	require.NoError(t, err)
	tx := testPool.NewTransaction()
	// Issue a WATCH command
	require.NoError(t, tx.WatchKey(key))
	// Update the key directly using a different connection. This should
	// trigger WATCH
	expectedVal := "bar"
	conn2 := testPool.NewConn()
	defer func() {
		_ = conn2.Close()
	}()
	_, err = conn2.Do("SET", key, expectedVal)
	require.NoError(t, err)
	// Try to update the key using the transaction. We expect this to fail
	// and return a WatchError
	tx.Command("SET", redis.Args{key, "should_not_be_set"}, nil)
	err = tx.Exec()
	assert.Error(t, err)
	assert.IsType(t, WatchError{}, err)
	// Finally get the key to make sure the changes in the transaction were not
	// committed.
	conn3 := testPool.NewConn()
	defer func() {
		_ = conn3.Close()
	}()
	got, err := redis.String(conn3.Do("GET", key))
	require.NoError(t, err)
	require.Exactly(t, expectedVal, got)
}
