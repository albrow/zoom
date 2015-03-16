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
	findModelsBySetIdsScript   *redis.Script
	deleteModelsBySetIdsScript *redis.Script
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
func (t *transaction) findModelsBySetIds(setKey string, modelName string, handler replyHandler) {
	t.script(findModelsBySetIdsScript, redis.Args{setKey, modelName}, handler)
}

// deleteModelsBySetIds is a small function wrapper around deleteModelsBySetIdsScript.
// It offers some type safety and helps make sure the arguments you pass through to the are correct.
// The script will delete the models corresponding to the ids in the given set and return the number
// of models that were deleted. You can use the handler to capture the return value.
func (t *transaction) deleteModelsBySetIds(setKey string, modelName string, handler replyHandler) {
	t.script(deleteModelsBySetIdsScript, redis.Args{setKey, modelName}, handler)
}
