package sms

import "context"

// Service 发送短信的抽象接口
// 该接口定义了发送短信的基本操作，目的是为了适配不同的短信供应商。
// 通过该接口，应用可以支持不同的短信服务提供商，而不需要修改其他业务逻辑部分。
// 目前，接口方法只有一个：Send，用于发送短信。
// 具体的短信发送逻辑由实现该接口的具体类型提供。
// 在实现该接口时，可以根据不同的短信供应商 API 来实现短信发送功能。
type Service interface {
	// Send 发送短信的方法
	// ctx: 上下文，携带请求的元数据，方便做超时控制等操作
	// tplId: 模板 ID，用于指定使用的短信模板
	// args: 模板替换参数，短信模板中的占位符会被这些参数替换
	// numbers: 目标手机号，可以传入一个或多个手机号，表示要发送短信的用户
	// 返回值：发送失败时返回 error，成功时返回 nil
	// 注意：该方法是发送短信的核心功能，不同的供应商会在这里实现具体的发送逻辑
	Send(ctx context.Context, tplId string, args []string, numbers ...string) error
}
