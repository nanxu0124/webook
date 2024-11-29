package service

import (
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"testing"
)

func TestPasswordEncrypt(t *testing.T) {
	pwd := []byte("123456#123456#11adasfasfsfsf2")
	// 加密
	// bcrypt
	// 不需要你自己去生成盐值
	// 不需要额外存储盐值
	// 可以通过控制 cost 来控制加密性能
	// 同样的文本，加密后的结果不同
	encrypted, err := bcrypt.GenerateFromPassword(pwd, bcrypt.DefaultCost)
	// 比较
	println(len(encrypted))
	err = bcrypt.CompareHashAndPassword(encrypted, pwd)
	require.NoError(t, err)
}
