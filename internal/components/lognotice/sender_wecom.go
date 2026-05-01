package lognotice

import (
	"context"
	"errors"
	"strconv"

	"github.com/hyetpang/go-frame/pkgs/logs"
	"go.uber.org/zap"
)

// 企业微信
type wecomSender struct {
	*senderBase
}

func newWecomNotice(base *senderBase) sender {
	return &wecomSender{senderBase: base}
}

type wecomSendMsgRsp struct {
	Errcode int `json:"errcode"`
}

func (s *wecomSender) Send(name, url string, msg noticeContent) error {
	// 用户输入字段经 HTML 转义,防止 <font>/@all 等富文本注入与超长刷屏
	safeName := escapeHTML(name)
	safeMsg := escapeHTML(msg.msg)
	safeFilename := escapeHTML(msg.filename)
	params := map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]any{
			"content": "服务[<font color=\"warning\">" + safeName + "</font>]出错啦,请排查问题,出错概览如下:\n>描述:" + safeMsg + "\n>代码行数:<font color=\"warning\">" + safeFilename + ":" + strconv.Itoa(msg.line) + "</font>" + "\n详情请查看具体日志文件.",
		},
	}
	response := new(wecomSendMsgRsp)
	if err := s.postJSON(context.Background(), url, params, response); err != nil {
		logs.ErrorWithoutNotice("企业微信发送消息出错", zap.Error(err), zap.Any("params", params))
		return err
	}
	if response.Errcode != 0 {
		logs.ErrorWithoutNotice("error日志通知出错,响应码errcode不是0", zap.Int("code", response.Errcode), zap.Any("params", params))
		return errors.New("error日志通知出错,响应码errcode不是0")
	}
	return nil
}
