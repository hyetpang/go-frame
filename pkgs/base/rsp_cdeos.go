package base

var (
	CodeErrOK            = NewCodeErr(0, "success")
	CodeErrSystem        = NewCodeErr(1, "系统出错")
	CodeErrParamsInvalid = NewCodeErr(2, "参数无效")
	CodeErrNotFound      = NewCodeErr(3, "404 not found")
)
