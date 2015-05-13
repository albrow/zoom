// Copyright 2015 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File scripts.go contains code related to lua scripts,
// including parsing the scripts in the scripts folder and
// wrapper functions which offer type safety for using them.

package zoom

import (
	"io/ioutil"
	"path/filepath"

	"github.com/garyburd/redigo/redis"
)

var (
	deleteModelsBySetIdsScript      *redis.Script
	deleteStringIndexScript         *redis.Script
	extractIdsFromFieldIndexScript  *redis.Script
	extractIdsFromStringIndexScript *redis.Script
)

// initScripts will parse all the lua script files in scriptsPath and assign them
// to the variables above. It must be run before any scripts are executed.
func initScripts(scriptsPath string) error {
	scriptsToParse := []struct {
		script   **redis.Script
		filename string
		keyCount int
	}{
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
			script:   &extractIdsFromFieldIndexScript,
			filename: "extract_ids_from_field_index.lua",
			keyCount: 2,
		},
		{
			script:   &extractIdsFromStringIndexScript,
			filename: "extract_ids_from_string_index.lua",
			keyCount: 2,
		},
	}
	for _, s := range scriptsToParse {
		// Parse the file corresponding to this script
		fullPath := filepath.Join(scriptsPath, s.filename)
		src, err := ioutil.ReadFile(fullPath)
		if err != nil {
			return err
		}
		// Set the value of the script pointer
		(*s.script) = redis.NewScript(s.keyCount, string(src))
	}
	return nil
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
