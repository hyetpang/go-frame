package lognotice

import (
	"context"
	"errors"
	"strconv"

	"github.com/hyetpang/go-frame/pkgs/logs"
	"go.uber.org/zap"
)

// telegram
type telegramSender struct {
	*senderBase
	chatID string
}

func newTelegramSender(base *senderBase, chatID string) sender {
	return &telegramSender{
		senderBase: base,
		chatID:     chatID,
	}
}

type telegramSenderMsgRsp struct {
	Description string `json:"description"`
	ErrorCode   int    `json:"error_code"`
	OK          bool   `json:"ok"`
}

func (s *telegramSender) Send(name, url string, msg noticeContent) error {
	// telegram 使用 parse_mode=HTML,任何用户输入都需要 HTML 转义,
	// 否则会被解析为 <a>/<b> 等标签,可能伪造钓鱼链接或破坏消息结构
	safeName := escapeHTML(name)
	safeMsg := escapeHTML(msg.msg)
	safeFilename := escapeHTML(msg.filename)
	params := map[string]any{
		"chat_id":              s.chatID,
		"text":                 "服务[<u>" + safeName + "</u>]出错啦,请排查问题,出错概览如下:\n描述:" + safeMsg + "\n代码行数:" + safeFilename + ":" + strconv.Itoa(msg.line) + "\n详情请查看具体日志文件",
		"disable_notification": true,
		"parse_mode":           "HTML",
	}
	response := new(telegramSenderMsgRsp)
	if err := s.postJSON(context.Background(), url, params, response); err != nil {
		logs.ErrorWithoutNotice("telegram发送消息出错", zap.Error(err), zap.Any("params", params))
		return err
	}
	if response.ErrorCode > 0 || !response.OK {
		logs.ErrorWithoutNotice("telegram发送消息出错，telegram接口返回码不是0", zap.Any("params", params), zap.Int("error_code", response.ErrorCode), zap.String("description", response.Description))
		return errors.New("telegram发送消息出错，telegram接口返回码不是0")
	}
	return nil
}
