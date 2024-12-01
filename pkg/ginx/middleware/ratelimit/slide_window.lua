-- ZREMRANGEBYSCORE key1 0 6
-- 执行完之后会将评分为 0 到 6 范围内的元素移除，即删除过期的元素

-- 限流对象
local key = KEYS[1]  -- 限流的 Redis 键（通常是基于客户端 IP 或其他标识符）
-- 窗口大小
local window = tonumber(ARGV[1])  -- 窗口大小，表示限流的时间窗口（单位是毫秒）
-- 阈值
local threshold = tonumber(ARGV[2])  -- 限流的阈值，即在该时间窗口内允许的最大请求次数
local now = tonumber(ARGV[3])  -- 当前时间戳（单位是毫秒）

-- 窗口的起始时间
local min = now - window  -- 计算当前时间窗口的起始时间

-- 移除时间窗口之前的所有请求记录（过期请求）
redis.call('ZREMRANGEBYSCORE', key, '-inf', min)  -- ZREMRANGEBYSCORE 命令会移除 ZSET 中分数（即时间戳）小于 `min` 的所有元素

-- 计算当前时间窗口内的请求数量
local cnt = redis.call('ZCOUNT', key, '-inf', '+inf')  -- ZCOUNT 命令会返回指定时间范围内的元素个数

-- 判断请求数是否超过阈值
if cnt >= threshold then
    -- 如果当前请求数超过了阈值，表示限流
    return "true"  -- 返回 "true" 表示请求被限流，不能继续执行
else
    -- 否则，允许该请求并将其记录到 Redis 中
    redis.call('ZADD', key, now, now)  -- 将当前时间戳 `now` 添加到 ZSET 中，作为分数和成员
    redis.call('PEXPIRE', key, window)  -- 设置该 Redis 键的过期时间，过期时间为窗口大小（单位是毫秒）
    return "false"  -- 返回 "false" 表示请求未被限流，可以继续执行
end
