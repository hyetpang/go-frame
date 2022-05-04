/*
 * @Date: 2022-04-30 16:35:51
 * @LastEditTime: 2022-04-30 16:38:17
 * @FilePath: \go-frame\internal\adapter\logadapter\cron_zap_adapter.go
 */
package log

import (
	"fmt"

	"go.uber.org/zap"
)

type CronLog struct {
	*zap.Logger
}

func (cl CronLog) Printf(msg string, format ...interface{}) {
	cl.Logger.Debug("cron_log", zap.String("cron", fmt.Sprintf(msg, format...)))
}
