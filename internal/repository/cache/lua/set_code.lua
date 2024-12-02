-- 发送到的 key，表示验证码存储的唯一标识，比如"code:业务:手机号码"
local key = KEYS[1]

-- 使用次数，也就是验证码的验证次数，存储在另一个 key 中
local cntKey = key .. ":cnt"

-- 获取传入的验证码值
local val = ARGV[1]

-- 获取 key 的剩余有效时间，单位是秒
-- ttl 返回的是当前 key 剩余的生存时间
-- 验证码的有效时间是十分钟，600 秒
local ttl = tonumber(redis.call("ttl", key))

-- -1 表示 key 存在，但没有设置过期时间
if ttl == -1 then
    -- 如果 TTL 为 -1，表示该 key 存在但是没有设置过期时间，
    -- 可能是误操作导致的 key 冲突，返回 -2 表示错误。
    return -2

    -- -2 表示 key 不存在，或者 TTL 小于 540 秒（即验证码已过期，超过一分钟），
    -- 允许重新发送验证码
elseif ttl == -2 or ttl < 540 then
    -- 如果 key 不存在，或者验证码已过期，重新设置验证码和使用次数
    -- 将验证码存储到 Redis，并设置有效期为 600 秒（10 分钟）
    redis.call("set", key, val)
    redis.call("expire", key, 600)

    -- 设置验证码的使用次数为 3，表示最多可以验证 3 次
    redis.call("set", cntKey, 3)
    redis.call("expire", cntKey, 600)  -- 同样设置使用次数的有效期为 600 秒（10 分钟）

    -- 返回 0 表示验证码发送成功
    return 0

else
    -- 如果验证码还没有过期，并且距离上次发送的时间小于一分钟，则不能发送新的验证码
    -- 返回 -1 表示验证码频繁，禁止发送
    return -1
end
