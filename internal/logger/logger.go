package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Logger *zap.Logger
)

func Init(environment string) error {
	var config zap.Config

	if environment == "production" {
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	} else {
		config = zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.StacktraceKey = "stacktrace"

	var err error
	Logger, err = config.Build(
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return err
	}

	zap.ReplaceGlobals(Logger)

	return nil
}

func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}

func WithRequestID(requestID string) *zap.Logger {
	return Logger.With(zap.String("request_id", requestID))
}

func Debug(msg string, fields ...zap.Field) {
	Logger.Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	Logger.Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	Logger.Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	Logger.Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	Logger.Fatal(msg, fields...)
}
