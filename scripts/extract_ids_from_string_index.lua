-- Copyright 2014 Alex Browne.  All rights reserved.
-- Use of this source code is governed by the MIT
-- license, which can be found in the LICENSE file.

-- exctract_ids_from_string_index is a lua script that takes the following arguments:
-- 	1) The key of a sorted set for a string index, where each member is of the
--			form: value + " " + id
--		2) The key of a sorted set where the resulting ids will be stored
-- The script then extracts the ids from the string index and store them
-- in the given key with the appropriate scores in ascending order.

-- Assign keys to variables for easy access
local setKey = KEYS[1]
local storeKey = KEYS[2]
-- Get all the members (value+id pairs) from the sorted set
local members = redis.call('ZRANGE', setKey, 0, -1)
if #members > 0 then
	-- Iterate over the members and extract the ids
	for i, member in ipairs(members) do
		-- The id is everything after the last space
		-- Find the index of the last space
		local idStart = string.find(member, ' [^ ]*$')
		local id = string.sub(member, idStart+1)
		redis.call('ZADD', storeKey, i, id)
	end
end