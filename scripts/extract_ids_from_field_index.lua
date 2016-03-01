-- Copyright 2015 Alex Browne.  All rights reserved.
-- Use of this source code is governed by the MIT
-- license, which can be found in the LICENSE file.

-- exctract_ids_from_field_index is a lua script that takes the following arguments:
-- 	1) setKey: The key of a sorted set for a field index (either numeric or bool)
-- 	2) destKey: The key of a sorted set where the resulting ids will be stored
--		3) min: The min argument for the ZRANGEBYSCORE command
-- 	4) max: The max argument for the ZRANGEBYSCORE command
-- The script then calls ZRANGEBYSCORE on setKey with the given min and max arguments,
-- and then stores the resulting set in destKey. It does not preserve the existing
-- scores, and instead just replaces scores with sequential numbers to keep the members
-- in the same order.

-- IMPORTANT: If you edit this file, you must run go generate . to rewrite ../scripts.go

-- Assign keys to variables for easy access
local setKey = ARGV[1]
local destKey = ARGV[2]
local min = ARGV[3]
local max = ARGV[4]
-- Get all the members (value+id pairs) from the sorted set
local members = redis.call('ZRANGEBYSCORE', setKey, min, max)
-- Iterate over the members and add each to the destKey
for i, member in ipairs(members) do
	redis.call('ZADD', destKey, i, member)
end
