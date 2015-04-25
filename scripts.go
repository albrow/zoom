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
	deleteModelsBySetIdsScript      *redis.Script
	deleteStringIndexScript         *redis.Script
	extractIdsFromStringIndexScript *redis.Script
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
			script:   &extractIdsFromStringIndexScript,
			filename: "extract_ids_from_string_index.lua",
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

// extractIdsFromStringIndex is a small function wrapper around extractIdsFromStringIndexScript.
// It offers some type safety and helps make sure the arguments you pass through to the are correct.
// The script will extract and return the ids in the given string index. You can use the handler to
// scan the ids into a slice of strings.
func (t *Transaction) extractIdsFromStringIndex(setKey string, orderKind orderKind, handler ReplyHandler) {
	t.Script(extractIdsFromStringIndexScript, redis.Args{setKey, orderKind.String()}, handler)
}
