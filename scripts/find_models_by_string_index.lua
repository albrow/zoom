-- Copyright 2014 Alex Browne.  All rights reserved.
-- Use of this source code is governed by the MIT
-- license, which can be found in the LICENSE file.

-- find_models_by_string_index is a lua script that takes the following arguments:
-- 	1) The key of a sorted set for a string index, where each member is of the
--			form: value + " " + id
--		2) The name of a registered model
--		3) The order (must be either "ascending" or "descending")
-- The script then extracts the ids from the string index in the specified order. Then for
-- each id, it finds the corresponding model and gets all the fields from the main hash.
-- It returns an array of arrays where each array contains the fields for a particular model.
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
local order = ARGV[2]
-- Get all the members (value+id pairs) from the sorted set
local members = {}
if order == 'ascending' then
	members = redis.call('ZRANGE', setKey, 0, -1)
else
	members = redis.call('ZREVRANGE', setKey, 0, -1)
end
local models = {}
if #members > 0 then
	-- Iterate over the members, extract the ids, and get the
	-- fields for the corresponding models
	for i, member in ipairs(members) do
		-- The id is everything after the last space
		-- Find the index of the last space
		local idStart = string.find(member, ' [^ ]*$')
		local id = string.sub(member, idStart+1)
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