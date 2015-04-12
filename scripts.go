// Copyright 2014 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File scripts.go contains code related to lua scripts,
// including parsing the scripts in the scripts folder and
// wrapper functions which offer type safety for using them.

package zoom

import (
	"github.com/garyburd/redigo/redis"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	findModelsBySetIdsScript              *redis.Script
	deleteModelsBySetIdsScript            *redis.Script
	deleteStringIndexScript               *redis.Script
	findModelsBySortedSetIdsScript        *redis.Script
	findModelsByReverseSortedSetIdsScript *redis.Script
)

var (
	scriptsPath = filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "albrow", "zoom", "scripts")
)

func init() {
	// Parse all the script templates and create redis.Script objects
	scriptsToParse := []struct {
		script   **redis.Script
		filename string
		keyCount int
	}{
		{
			script:   &findModelsBySetIdsScript,
			filename: "find_models_by_set_ids.lua",
			keyCount: 1,
		},
		{
			script:   &deleteModelsBySetIdsScript,
			filename: "delete_models_by_set_ids.lua",
			keyCount: 1,
		},
		{
			script:   &deleteStringIndexScript,
			filename: "delete_string_index.lua",
			keyCount: 0,
		},
		{
			script:   &findModelsBySortedSetIdsScript,
			filename: "find_models_by_sorted_set_ids.lua",
			keyCount: 1,
		},
		{
			script:   &findModelsByReverseSortedSetIdsScript,
			filename: "find_models_by_reverse_sorted_set_ids.lua",
			keyCount: 1,
		},
	}
	for _, s := range scriptsToParse {
		// Parse the file corresponding to this script
		fullPath := filepath.Join(scriptsPath, s.filename)
		src, err := ioutil.ReadFile(fullPath)
		if err != nil {
			panic(err)
		}
		// Set the value of the script pointer
		(*s.script) = redis.NewScript(s.keyCount, string(src))
	}
}

// findModelsBySetIds is a small function wrapper around findModelsBySetIdsScript.
// It offers some type safety and helps make sure the arguments you pass through to the are correct.
// The script will return all the fields for models which are identified by ids in the given set.
// You can use the handler to scan the models into a slice of models.
func (t *Transaction) findModelsBySetIds(setKey string, modelName string, handler ReplyHandler) {
	t.Script(findModelsBySetIdsScript, redis.Args{setKey, modelName}, handler)
}

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

// findModelsBySortedSetIds is a small function wrapper around findModelsBySortedSetIdsScript.
// It offers some type safety and helps make sure the arguments you pass through to the are correct.
// The script will return all the fields for models (in order) which are identified by ids in the
// given sorted set.
// You can use the handler to scan the models into a slice of models.
func (t *Transaction) findModelsBySortedSetIds(setKey string, modelName string, handler ReplyHandler) {
	t.Script(findModelsBySortedSetIdsScript, redis.Args{setKey, modelName}, handler)
}

// findModelsByReverseSortedSetIds is a small function wrapper around findModelsByReverseSortedSetIdsScript.
// It offers some type safety and helps make sure the arguments you pass through to the are correct.
// The script will return all the fields for models (in *reverse* order) which are identified by ids in the
// given sorted set.
// You can use the handler to scan the models into a slice of models.
func (t *Transaction) findModelsByReverseSortedSetIds(setKey string, modelName string, handler ReplyHandler) {
	t.Script(findModelsByReverseSortedSetIdsScript, redis.Args{setKey, modelName}, handler)
}
