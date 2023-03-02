package lognotice

import (
	"errors"
	"strconv"
	"time"

	"github.com/HyetPang/go-frame/pkgs/logs"
	"github.com/guonaihong/gout"

	"go.uber.org/zap"
)

// 企业微信
type feiShuSender struct{}

func newFeiShuNotice() sender {
	return &feiShuSender{}
}

type feiShuSendMsgRsp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// 通知
func (feiShuSender *feiShuSender) Send(name, url string, msg noticeContent) error {
	params := make(map[string]interface{}, 3)
	params["msg_type"] = "text"
	var content string
	content = "{\"text\":\"服务[" + name + "]出错啦,请排查问题,出错概览如下: \\n描述:" + msg.msg + " \\n代码行数:" + msg.filename + ":" + strconv.Itoa(msg.line) + "  \\n详情请查看具体日志文件\"}"
	params["content"] = content
	response := new(feiShuSendMsgRsp)
	err := gout.POST(url).SetTimeout(time.Second * 5).SetJSON(params).BindJSON(response).Do()
	if err != nil {
		logs.ErrorWithoutNotice("飞书发送消息出错", zap.Error(err), zap.Any("params", params))
		return err
	}
	if response.Code > 0 {
		logs.ErrorWithoutNotice("飞书发送消息出错，飞书接口返回码不是0", zap.Error(err), zap.Any("params", params), zap.Int("code", response.Code), zap.String("msg", response.Msg))
		return errors.New("飞书发送消息出错，飞书接口返回码不是0")
	}
	return nil
}
