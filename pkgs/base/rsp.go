package base

type CodeErrI interface {
	GetMsg() string
	GetCode() uint
	error
}

type codeErrImpl struct {
	Code uint
	Msg  string
	err  error
}

func NewCodeErr(code uint, msg string) CodeErrI {
	return &codeErrImpl{Code: code, Msg: msg}
}

func (ce *codeErrImpl) GetCode() uint {
	return ce.Code
}

func (ce *codeErrImpl) GetMsg() string {
	return ce.Msg
}

func (ce *codeErrImpl) Error() string {
	err := ce.err
	if err != nil {
		return err.Error()
	}
	return ""
}

func toCodeI(err error) (CodeErrI, bool) {
	if err == nil {
		return nil, true
	}
	var errInterface interface{} = err
	codeE, ok := errInterface.(CodeErrI)
	return codeE, ok
}

func GetCodeI(err error) CodeErrI {
	if err == nil {
		return nil
	}
	codeE, ok := toCodeI(err)
	if ok {
		return codeE
	}
	return CodeErrSystem
}
