package logger

import (
	"go-ws/utils/zaplog"
	"go.uber.org/zap"

)

var (
	// Logger is a run log instance
	Logger *zap.Logger
	cfg    *zaplog.Config
)

func init() {
	cfg := zaplog.Config{
		EncodeLogsAsJson:   true,
		FileLoggingEnabled: true,
		Directory:          "/data/logs/ws/",
		Filename:           "run.log",
		MaxSize:            512,
		MaxBackups:         30,
		MaxAge:             7,
	}
	Logger = zaplog.GetLogger(cfg)
}
