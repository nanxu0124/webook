package ioc

import (
	"github.com/redis/go-redis/v9"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tencentSMS "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
	"time"
	"webook/internal/service/sms"
	smsRatelimit "webook/internal/service/sms/ratelimit"
	"webook/internal/service/sms/tencent"
	"webook/pkg/ratelimit"
)

func InitSMSService(cmd redis.Cmdable) sms.Service {
	return InitRateLimitSMSService(initSmsTencentService(), cmd)
}

func initSmsTencentService() sms.Service {
	c, err := tencentSMS.NewClient(common.NewCredential("***", "***"),
		"ap-guangzhou",
		profile.NewClientProfile())
	if err != nil {
		panic(err)
	}

	s := tencent.NewService(c, "1400881894", "南絮0124公众号")

	return s
}

func InitRateLimitSMSService(svc sms.Service, cmd redis.Cmdable) sms.Service {
	return smsRatelimit.NewRatelimitSMSService(svc, initSMSLimiter(cmd))
}

func initSMSLimiter(cmd redis.Cmdable) ratelimit.Limiter {
	return ratelimit.NewRedisSlidingWindowLimiter(cmd, time.Minute*30, 3)
}
