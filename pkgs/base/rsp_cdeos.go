package base

// 预留错误码，建议业务逻辑错误码从100开始，1-99给框架预留
var (
	CodeErrOK            = NewCodeErr(0, "success")
	CodeErrSystem        = NewCodeErr(1, "系统出错")
	CodeErrParamsInvalid = NewCodeErr(2, "参数无效")
	CodeErrNotFound      = NewCodeErr(3, "404 not found")
)
