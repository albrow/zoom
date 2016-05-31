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
	tx.Command("HSET", redis.Args{testModels.ModelKey(model.Id), "Int", 35}, nil)
	err := tx.Exec()
	assert.Error(t, err)
	assert.IsType(t, WatchError{}, err)
	// Finally retrieve the model to make sure the changes in the transaction
	// were not committed.
	other := testModel{}
	require.NoError(t, testModels.Find(model.Id, &other))
	assert.Equal(t, expectedString, other.String, "First update *was not* committed")
	assert.Equal(t, model.Int, other.Int, "Second update *was* committed")
}
