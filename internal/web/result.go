package web

type Result struct {
	// 业务错误码
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}
