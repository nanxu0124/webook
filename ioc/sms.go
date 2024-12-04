package ioc

import (
	"github.com/redis/go-redis/v9"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tencentSMS "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
	"os"
	"time"
	"webook/internal/service/sms"
	smsRatelimit "webook/internal/service/sms/ratelimit"
	"webook/internal/service/sms/tencent"
	pkgRatelimit "webook/pkg/ratelimit"
)

func InitSmsService(cmd redis.Cmdable) sms.Service {
	return initRedisSlidingWindowLimiter(cmd)
}

func initSmsTencentService() sms.Service {
	secretId, ok := os.LookupEnv("Tencent_SMS_Secret_Id")
	if !ok {
		panic("没有找到环境变量 Tencent_SMS_Secret_Id ")
	}
	secretKey, ok := os.LookupEnv("Tencent_SMS_Secret_Key")
	if !ok {
		panic("没有找到环境变量 Tencent_SMS_Secret_Key ")
	}

	c, err := tencentSMS.NewClient(common.NewCredential(secretId, secretKey),
		"ap-beijing",
		profile.NewClientProfile())
	if err != nil {
		panic("tencentSMS 初始化失败 ")
	}
	return tencent.NewService(c, "1400952398", "南絮0124公众号")
}

func initRedisSlidingWindowLimiter(cmd redis.Cmdable) sms.Service {
	limiter := pkgRatelimit.NewRedisSlidingWindowLimiter(cmd, time.Minute, 3)
	tencentSMSService := initSmsTencentService()
	return smsRatelimit.NewRatelimitSMSService(tencentSMSService, limiter)
}
