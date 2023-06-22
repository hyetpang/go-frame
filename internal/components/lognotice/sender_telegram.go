package lognotice

import (
	"errors"
	"strconv"
	"time"

	"github.com/guonaihong/gout"
	"github.com/hyetpang/go-frame/pkgs/logs"

	"go.uber.org/zap"
)

// telegram
type telegramSender struct {
	chatID string
}

func newTelegramSender(chatID string) sender {
	return &telegramSender{
		chatID: chatID,
	}
}

type telegramSenderMsgRsp struct {
	Description string `json:"description"`
	ErrorCode   int    `json:"error_code"`
	OK          bool   `json:"ok"`
}

// 通知
func (telegram *telegramSender) Send(name, url string, msg noticeContent) error {
	params := make(map[string]interface{}, 3)
	params["chat_id"] = telegram.chatID
	params["text"] = "服务[<u>" + name + "</u>]出错啦,请排查问题,出错概览如下:\n描述:" + msg.msg + "\n代码行数:" + msg.filename + ":" + strconv.Itoa(msg.line) + "\n详情请查看具体日志文件"
	params["disable_notification"] = true
	params["parse_mode"] = "HTML"
	response := new(telegramSenderMsgRsp)
	err := gout.POST(url).SetTimeout(time.Second * 5).SetJSON(params).BindJSON(response).Do()
	if err != nil {
		logs.ErrorWithoutNotice("telegram发送消息出错", zap.Error(err), zap.Any("params", params))
		return err
	}
	if response.ErrorCode > 0 || !response.OK {
		logs.ErrorWithoutNotice("telegram发送消息出错，telegram接口返回码不是0", zap.Any("params", params), zap.Int("error_code", response.ErrorCode), zap.String("description", response.Description))
		return errors.New("telegram发送消息出错，telegram接口返回码不是0")
	}
	return nil
}
