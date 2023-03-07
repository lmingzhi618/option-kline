package common

import (
	"bufio"
	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/pkg/errors"
	"github.com/rifflock/lfshook"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"time"
)

var DefaultLoggerFormat = &log.JSONFormatter{
	TimestampFormat: LOGTIME_FORMAT,
}

// 配置业务日志系统
func ConfigLogger() {
	if CURMODE != ENV_ONLINE {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	if DEBUG == "1" {
		log.SetLevel(log.DebugLevel)
	}
	log.SetFormatter(DefaultLoggerFormat)
	setNull()
	configLocalFs(LOGDIR, LOGFILENAME, time.Hour*time.Duration(LOGKEEPDAYS*24), time.Hour*time.Duration(LOGRATEHOURS))
}

// 配置本地文件系统并按周期分割
func configLocalFs(logPath string, logFileName string, maxAge time.Duration, rotationTime time.Duration) {
	baseLogPath := path.Join(logPath, logFileName)
	writer, err := rotatelogs.New(
		baseLogPath+".%Y%m%d",
		rotatelogs.WithLinkName(baseLogPath),      // 生成软链，指向最新日志文件
		rotatelogs.WithMaxAge(maxAge),             // 文件最大保存时间
		rotatelogs.WithRotationTime(rotationTime), // 日志切割时间间隔
	)
	if err != nil {
		log.Errorf("config local file system logger error. %+v", errors.WithStack(err))
	}
	lfHook := lfshook.NewHook(lfshook.WriterMap{
		log.DebugLevel: writer, // 为不同级别设置不同的输出目的
		log.InfoLevel:  writer,
		log.WarnLevel:  writer,
		log.ErrorLevel: writer,
		log.FatalLevel: writer,
		log.PanicLevel: writer,
	}, DefaultLoggerFormat)
	log.AddHook(lfHook)
}

func setNull() {
	src, err := os.OpenFile(os.DevNull, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		log.Errorf("setNull failed:", err)
	}
	writer := bufio.NewWriter(src)
	log.SetOutput(writer)
}
