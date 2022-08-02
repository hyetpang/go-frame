package lognotice

import (
	"runtime"
	"strconv"
	"time"

	"github.com/HyetPang/go-frame/pkgs/logs"
	"github.com/guonaihong/gout"

	"go.uber.org/zap"
)

// 企业微信
type wecomNotice struct {
	conf     *config
	noticeCh chan noticeContent
}

type noticeContent struct {
	msg, filename string
	line          int
}

// 通知
func (wecomNotice *wecomNotice) noticeMsg() {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if ok {
				logs.ErrorWithoutNotice("企业微信错误通知panic", zap.Error(err))
			} else {
				logs.ErrorWithoutNotice("企业微信错误通知panic", zap.Any("msg", r))
			}
		}
	}()
	for noticeMsg := range wecomNotice.noticeCh {
		wecomNotice.notice(noticeMsg)
	}
}

func (wecomNotice *wecomNotice) Notice(msg string, fields ...zap.Field) {
	_, filename, line, _ := runtime.Caller(3)
	wecomNotice.noticeCh <- noticeContent{
		msg:      msg,
		filename: filename,
		line:     line,
	}
}

func (wecomNotice *wecomNotice) notice(msg noticeContent) {
	params := make(map[string]interface{}, 3)
	params["msgtype"] = "markdown"
	params["markdown"] = map[string]interface{}{
		"content": "服务[<font color=\"warning\">" + wecomNotice.conf.Name + "</font>]出错啦,请排查问题,出错概览如下:\n>描述:" + msg.msg + "\n>代码行数:<font color=\"warning\">" + msg.filename + ":" + strconv.Itoa(msg.line) + "</font>" + "\n详情请查看具体日志文件.",
	}
	response := make(map[string]interface{})
	err := gout.POST(wecomNotice.conf.WecomURL).SetTimeout(time.Second * 5).SetJSON(params).BindJSON(&response).Do()
	if err != nil {
		logs.ErrorWithoutNotice("企业微信发送消息出错", zap.Error(err), zap.Any("params", params))
		return
	}
	errcode, ok := response["errcode"]
	if !ok {
		logs.ErrorWithoutNotice("error日志通知出错,响应码不包含errcode", zap.Any("response", response), zap.Any("params", params))
		return
	}
	if errcode.(float64) != 0 {
		logs.ErrorWithoutNotice("error日志通知出错,响应码errcode不是0", zap.Any("response", response), zap.Any("params", params))
		return
	}
}
