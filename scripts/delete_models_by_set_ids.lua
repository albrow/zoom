-- Copyright 2015 Alex Browne.  All rights reserved.
-- Use of this source code is governed by the MIT
-- license, which can be found in the LICENSE file.

-- delete_models_by_set_ids is a lua script that takes the following arguments:
-- 	1) The key of a set of model ids
--		2) The name of a registered model
-- The script then deletes all the models corresponding to the ids in the given
-- set. It returns the number of models that were deleted. It does not delete the
-- given set.

-- IMPORTANT: If you edit this file, you must run go generate . to rewrite ../scripts.go

-- Assign keys to variables for easy access
local setKey = ARGV[1]
local collectionName = ARGV[2]
-- Get all the ids from the set name
local ids = redis.call('SMEMBERS', setKey)
local count = 0
if #ids > 0 then
	-- Iterate over the ids
	for i, id in ipairs(ids) do
		-- Delete the main hash for each model
		local key = collectionName .. ':' .. id
		count = count + redis.call('DEL', key)
		-- Remove the model id from the set of all ids
		-- NOTE: this is not necessarily the same as the
		-- setName we were given
		local setKey = collectionName .. ':all'
		redis.call('SREM', setKey, id)
	end
end
return count
