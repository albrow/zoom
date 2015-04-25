-- Copyright 2014 Alex Browne.  All rights reserved.
-- Use of this source code is governed by the MIT
-- license, which can be found in the LICENSE file.

-- find_models_by_set_ids is a lua script that takes the following arguments:
-- 	1) The key of a set of model ids
--		2) The name of a registered model
--		3) Limit, i.e. the maximum number of models to return
--		4) Offset, i.e. the number of models to skip
-- The script then gets all the data for the models corresponding to the ids
-- from their respective hashes in the database. It returns an array of arrays
-- where each array contains the fields for a particular model.
-- Here's an example response:
-- [
-- 	[
-- 		"Id", "afj9afjpa30",
-- 		"Name", "Foo",
--			"Age", 23
-- 	],
-- 	[
-- 		"Id", "ape832jdnu4",
-- 		"Name", "Bar",
--			"Age", 27
-- 	]
-- ]

-- Assign keys to variables for easy access
local setKey = KEYS[1]
local modelName = ARGV[1]
local limit = ARGV[2]
local offset = ARGV[3]
-- Get all the ids from the set name
local ids = {}
if (limit == '0' and offset == '0') then
	ids = redis.call('SMEMBERS', setKey)
else
	ids = redis.call('SORT', setKey, 'BY', 'nosort', 'LIMIT', offset, limit)
end
local models = {}
if #ids > 0 then
	-- Iterate over the ids and find each job
	for i, id in ipairs(ids) do
		local key = modelName .. ':' .. id
		local fields = redis.call('HGETALL', key)
		-- Add the id itself to the fields
		table.insert(fields, '-')
		table.insert(fields, id)
		-- Add the field values to models
		table.insert(models, fields)
	end
end
return models