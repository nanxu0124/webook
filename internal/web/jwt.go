package web

import "github.com/golang-jwt/jwt/v5"

// UserClaims 是一个自定义的结构体，表示JWT中包含的用户信息
// 它继承了 jwt.RegisteredClaims 结构体，用于存储JWT的标准字段（如过期时间、发行者等）
// 另外我们添加了一个 Id 字段，用于存储用户的唯一标识符（如用户ID）
type UserClaims struct {
	// 用户的唯一ID
	Id int64

	// 用户的 UserAgent（通常是浏览器或客户端的标识）
	// 这个字段可以帮助记录发起请求的客户端类型或来源设备，通常用于日志分析或安全审计
	UserAgent string

	// jwt.RegisteredClaims 是一个结构体，包含JWT的标准字段
	// 比如：过期时间（ExpiresAt）、发行者（Issuer）、受众（Audience）等
	// 通过嵌入 RegisteredClaims，我们可以直接访问这些标准字段
	// 例如：UserClaims.ExpiresAt 直接就能访问过期时间
	jwt.RegisteredClaims
}

// JWTKey 是用来签署和验证JWT的密钥
// 这个密钥通常不应该硬编码在代码中，实际应用中可以考虑将其存储在环境变量或配置文件中
// 但是在这个示例中，为了简便起见，我们将其写成了常量
var JWTKey = []byte("moyn8y9abnd7q4zkq2m73yw8tu9j5ixm")
