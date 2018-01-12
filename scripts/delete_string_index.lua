-- Copyright 2015 Alex Browne.  All rights reserved.
-- Use of this source code is governed by the MIT
-- license, which can be found in the LICENSE file.

-- delete_string_index is a lua script that takes the following arguments:
-- 	1) The name of a registered model
--		2) The id of the model to be deleted from the index
--		3) The name of the indexed string field
-- The script then checks if there is a value for the given field name stored in the
-- model hash, and if there is, removes the model from the index on the given field.
-- NOTE: This script *must* be called before the main hash for the model is updated/deleted.

-- IMPORTANT: If you edit this file, you must run go generate . to rewrite ../scripts.go

-- Assign keys to variables for easy access
local collectionName = ARGV[1]
local modelID = ARGV[2]
local fieldName = ARGV[3]
-- Get the old value from the existing model hash (if any)
local modelKey = collectionName .. ":" .. modelID
local oldValue = redis.call("HGET", modelKey, fieldName)
local indexKey = collectionName .. ":" .. fieldName
if oldValue ~= false then
	-- Remove the model from the field index
	local oldMember = oldValue .. "\0" .. modelID
	redis.call("ZREM", indexKey, oldMember)
end
