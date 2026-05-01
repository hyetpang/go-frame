package lognotice

import (
	"context"
	"errors"
	"strconv"

	"github.com/hyetpang/go-frame/pkgs/logs"
	"go.uber.org/zap"
)

// 飞书
type feiShuSender struct {
	*senderBase
}

func newFeiShuNotice(base *senderBase) sender {
	return &feiShuSender{senderBase: base}
}

type feiShuSendMsgRsp struct {
	Msg  string `json:"msg"`
	Code int    `json:"code"`
}

func (s *feiShuSender) Send(name, url string, msg noticeContent) error {
	// 飞书使用 text 模式,做基础清理:截断 + 控制字符过滤,
	// 避免超长输入或终端控制字符注入伪造多行结构
	safeName := escapePlain(name)
	safeMsg := escapePlain(msg.msg)
	safeFilename := escapePlain(msg.filename)
	params := map[string]any{
		"msg_type": "text",
		"content": map[string]string{
			"text": "服务[" + safeName + "]出错啦,请排查问题,出错概览如下: \n描述:" + safeMsg + " \n代码行数:" + safeFilename + ":" + strconv.Itoa(msg.line) + "  \n详情请查看具体日志文件",
		},
	}
	response := new(feiShuSendMsgRsp)
	if err := s.postJSON(context.Background(), url, params, response); err != nil {
		logs.ErrorWithoutNotice("飞书发送消息出错", zap.Error(err), zap.Any("params", params))
		return err
	}
	if response.Code > 0 {
		logs.ErrorWithoutNotice("飞书发送消息出错，飞书接口返回码不是0", zap.Any("params", params), zap.Int("code", response.Code), zap.String("msg", response.Msg))
		return errors.New("飞书发送消息出错，飞书接口返回码不是0")
	}
	return nil
}
