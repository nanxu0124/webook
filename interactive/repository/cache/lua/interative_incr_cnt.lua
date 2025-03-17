-- 获取传入的 Redis 键（哈希表键）和参数
local key = KEYS[1]        -- KEYS[1] 是传入的 Redis 键（哈希表的名称）
local cntKey = ARGV[1]      -- ARGV[1] 是哈希表中的字段名称
local delta = tonumber(ARGV[2])  -- ARGV[2] 是增量（delta），转为数字类型

-- 检查键是否存在
local exists = redis.call("EXISTS", key)  -- 调用 Redis 的 EXISTS 命令，检查 key 是否存在

-- 如果键存在，执行自增操作
if exists == 1 then
    -- 如果哈希表 key 存在，则对哈希表中的 cntKey 字段执行 HINCRBY 操作，自增 delta
    redis.call("HINCRBY", key, cntKey, delta)
    -- 返回 1，表示自增操作成功
    return 1
else
    -- 如果键不存在，返回 0
    return 0
end
