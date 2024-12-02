-- 获取输入的 key，代表验证码的唯一标识（如手机号、邮箱等）
local key = KEYS[1]

-- 构造验证次数的 Redis 键
-- 验证次数的键名是原始键名 + ":cnt"
local cntKey = key .. ":cnt"

-- 获取预期中的验证码（用户输入的验证码）
local expectedCode = ARGV[1]

-- 从 Redis 获取当前验证码验证次数
local cnt = tonumber(redis.call("get", cntKey))
-- 从 Redis 获取存储的验证码
local code = redis.call("get", key)

-- 验证次数已经耗尽，返回 -1 表示验证次数已用完
if cnt <= 0 then
    return -1
end

-- 如果用户输入的验证码与存储的验证码相等
if code == expectedCode then
    -- 验证成功，将验证码的验证次数标记为 -1，表示验证码不可用
    -- 这样下一次就无法再使用这个验证码进行验证
    redis.call("set", cntKey, -1)
    return 0  -- 返回 0 表示验证码验证成功

else
    -- 验证码错误，可能是用户输入错误
    -- 递减剩余的验证次数
    redis.call("decr", cntKey)
    return -2  -- 返回 -2 表示验证码错误
end
