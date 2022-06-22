package lognotice

import (
	"log"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/HyetPang/go-frame/pkgs/interfaces"
	"github.com/HyetPang/go-frame/pkgs/logs"
	"github.com/guonaihong/gout"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// 企业微信
type wecomNotice struct {
	conf *config
}

func (wecomNotice *wecomNotice) Notice(msg string, fields ...zap.Field) {
	params := make(map[string]interface{}, 3)
	params["msgtype"] = "markdown"
	var content strings.Builder
	content.WriteString("服务[<font color=\"warning\">")
	content.WriteString(wecomNotice.conf.Name)
	content.WriteString("</font>]出错啦,请排查问题,出错概览如下:\n")
	content.WriteString(">描述:" + msg)
	_, filename, line, ok := runtime.Caller(3)
	if ok {
		content.WriteString("\n>代码行数:<font color=\"warning\">" + filename + ":" + strconv.Itoa(line) + "</font>")
	}
	content.WriteString("\n详情请查看具体日志文件.")
	params["markdown"] = map[string]interface{}{
		"content": content.String(),
	}
	response := make(map[string]interface{})
	err := gout.POST(wecomNotice.conf.WecomURL).SetTimeout(time.Second * 5).SetJSON(params).BindJSON(&response).Do()
	if err != nil {
		logs.ErrorWithoutNotice("企业微信发送消息出错", zap.Error(err))
		return
	}
	errcode, ok := response["errcode"]
	if !ok {
		logs.ErrorWithoutNotice("error日志通知出错,响应码不包含errcode", zap.Any("response", response))
		return
	}
	if errcode.(float64) != 0 {
		logs.ErrorWithoutNotice("error日志通知出错,响应码errcode不是0", zap.Any("response", response))
		return
	}
}

func New() interfaces.LogNoticeInterface {
	conf := new(config)
	err := viper.UnmarshalKey("log_notice", &conf)
	if err != nil {
		log.Fatal("zap_log配置Unmarshal到对象出错", zap.Error(err))
	}
	return &wecomNotice{
		conf: conf,
	}
}
