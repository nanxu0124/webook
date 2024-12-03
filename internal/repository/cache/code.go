package cache

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
)

var (
	//go:embed lua/set_code.lua
	luaSetCode string
	//go:embed lua/verify_code.lua
	luaVerifyCode             string
	ErrCodeSendTooMany        = errors.New("发送验证码太频繁")
	ErrUnknownForCode         = errors.New("发送验证码遇到未知错误")
	ErrCodeVerifyTooManyTimes = errors.New("验证次数太多")
)

// CodeCache 是用于存储和验证验证码的接口
type CodeCache interface {

	// Set 用于存储验证码到缓存
	// 该方法将验证码与手机号和业务标识 (biz) 关联，并设置缓存过期时间
	// 参数:
	//   - ctx: 上下文，用于控制请求的生命周期
	//   - biz: 业务标识，用于区分不同的验证码用途 (例如，登录、注册等)
	//   - phone: 用户的手机号码，用于唯一标识验证码的存储
	//   - code: 要存储的验证码
	// 返回:
	//   - error: 如果出现错误，返回错误信息；否则返回 nil
	Set(ctx context.Context, biz string, phone string, code string) error

	// Verify 用于验证输入的验证码是否与存储的验证码匹配
	// 如果验证码有效且未过期，返回 true，否则返回 false
	// 参数:
	//   - ctx: 上下文，用于控制请求的生命周期
	//   - biz: 业务标识，用于查找相应的验证码
	//   - phone: 用户的手机号码，用于定位验证码
	//   - inputCode: 用户输入的验证码，用于与缓存中的验证码进行对比
	// 返回:
	//   - bool: 如果验证码正确且有效，返回 true；否则返回 false
	//   - error: 如果发生错误，返回错误信息；如果没有错误发生，则返回 nil
	Verify(ctx context.Context, biz string, phone string, inputCode string) (bool, error)
}

// RedisCodeCache 实现 CodeCache 接口
type RedisCodeCache struct {
	redis redis.Cmdable
}

func NewRedisCodeCache(cmd redis.Cmdable) CodeCache {
	return &RedisCodeCache{
		redis: cmd,
	}
}

// Set 设置验证码
// 该方法使用 Redis 执行 Lua 脚本来设置验证码，并根据不同情况做出相应处理：
// - 如果该手机号码在该业务场景下没有验证码，或者验证码已经过期，则发送新验证码。
// - 如果已发送验证码且超过一分钟，允许重新发送验证码。
// - 如果验证码没有过期且不到一分钟，拒绝发送验证码。
// - 验证码有效期为 10 分钟。
func (c *RedisCodeCache) Set(ctx context.Context, biz string, phone string, code string) error {
	// 使用 Redis 执行 Lua 脚本，设置验证码
	// `luaSetCode` 是设置验证码的 Lua 脚本字符串，`c.key(biz, phone)` 是生成存储验证码的 Redis 键，`code` 是验证码值
	res, err := c.redis.Eval(ctx, luaSetCode, []string{c.key(biz, phone)}, code).Int()
	if err != nil {
		// 如果执行 Redis 命令时出错，返回错误
		return err
	}

	// 根据 Lua 脚本返回的结果做出相应处理
	switch res {
	case 0:
		// 如果返回值为 0，表示验证码设置成功，返回 nil
		return nil
	case -1:
		// 如果返回值为 -1，表示最近已发送验证码，不允许重复发送
		// 返回验证码发送过多的错误提示
		return ErrCodeSendTooMany
	default:
		// 如果返回值不是 0 或 -1，表示系统错误，可能是 key 冲突或未知错误
		// TODO：这里应当记录日志来追踪错误
		return ErrUnknownForCode
	}
}

// Verify 验证用户输入的验证码
// 该方法使用 Redis 执行 Lua 脚本来验证验证码，避免了多个 Redis 操作的性能问题。
// - biz：业务标识，用于区分不同业务场景的验证码。
// - phone：用户的手机号码，用于唯一标识验证码。
// - inputCode：用户输入的验证码。
func (c *RedisCodeCache) Verify(ctx context.Context, biz string, phone string, inputCode string) (bool, error) {
	// 使用 Redis 执行 Lua 脚本验证验证码
	// `luaVerifyCode` 是一个 Lua 脚本字符串，负责验证验证码的有效性。
	// `c.key(biz, phone)` 是生成存储验证码的 Redis 键。
	// `inputCode` 是用户输入的验证码。
	res, err := c.redis.Eval(ctx, luaVerifyCode, []string{c.key(biz, phone)}, inputCode).Int()
	if err != nil {
		// 如果执行 Redis 命令时出错，返回错误
		return false, err
	}

	// 根据 Lua 脚本返回的结果做出相应处理
	switch res {
	case 0:
		// 如果返回值为 0，表示验证码验证成功
		return true, nil
	case -1:
		// 如果返回值为 -1，表示验证次数已耗尽，可能是恶意请求
		// 返回验证码验证失败，并给出错误提示
		return false, ErrCodeVerifyTooManyTimes
	default:
		// 如果返回值不是 0 或 -1，表示验证码错误
		// 返回验证码错误并且不做进一步处理
		return false, nil
	}
}

func (c *RedisCodeCache) key(biz string, phone string) string {
	return fmt.Sprintf("phone_code:%s:%s", biz, phone)
}
