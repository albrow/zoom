-- Copyright 2014 Alex Browne.  All rights reserved.
-- Use of this source code is governed by the MIT
-- license, which can be found in the LICENSE file.

-- save_string_index is a lua script that takes the following arguments:
-- 	1) The name of a registered model
--		2) The id of the model to be saved
--		3) The name of the field to be indexed
--		4) The value of the field to be indexed
-- The script then atomically saves the string index for the given model and
-- field. It does this by first checking if there was an old value in the database,
-- and if so, removes it from the index. Then it adds the new value to the index.
-- Because it checks the old value of the model field, this script should be called
-- before the main hash for the model is saved.

-- Assign keys to variables for easy access
local modelName = ARGV[1]
local modelId = ARGV[2]
local fieldName = ARGV[3]
local fieldValue = ARGV[4]
-- Get the old value from the existing model hash (if any)
local modelKey = modelName .. ":" .. modelId
local oldValue = redis.call("HGET", modelKey, fieldName)
local indexKey = modelName .. ":" .. fieldName
if oldValue ~= false then
	-- Remove the old index
	local oldMember = oldValue .. " " .. modelId
	redis.call("ZREM", indexKey, oldMember)
end
-- Add the new index
local newMember = fieldValue .. " " .. modelId
redis.call("ZADD", indexKey, 0, newMember)