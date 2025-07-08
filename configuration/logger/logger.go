package logger

import (
	"go.uber.org/zap"
	
)

var (
	log *zap.Logger
)

func init() {
	log, _ = zap.NewProduction()
}

func Info(message string, tags ...zap.Field) {
	log.Info(message, tags...)
}

func Error(message string, err error, tags ...zap.Field) {
	tags = append(tags, zap.NamedError("error", err))
	log.Error(message, tags...)
}
