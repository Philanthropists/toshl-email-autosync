package logging

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type (
	Field  = zapcore.Field
	Option = zap.Option
)

type LoggerCtxKey struct{}

type zapLogger interface {
	DPanic(msg string, fields ...zapcore.Field)
	Debug(msg string, fields ...zapcore.Field)
	Error(msg string, fields ...zapcore.Field)
	Fatal(msg string, fields ...zapcore.Field)
	Info(msg string, fields ...zapcore.Field)
	Panic(msg string, fields ...zapcore.Field)
	Sync() error
	Warn(msg string, fields ...zapcore.Field)
	With(fields ...zapcore.Field) *zap.Logger
	WithOptions(opts ...zap.Option) *zap.Logger
}

type Logger struct {
	log zapLogger
}

var (
	logOnce      sync.Once
	cachedLogger *Logger
)

func SetCustomGlobalLogger(logger zapLogger) {
	if logger != nil {
		logOnce.Do(func() {
			cachedLogger = &Logger{
				log: logger,
			}
		})
	}
}

func insideContainer() bool {
	return os.Getenv("GO_ENVIRONMENT") == "production"
}

func defaultLogger() *zap.Logger {
	opts := []Option{
		zap.AddCallerSkip(1),
	}

	var logCfg zap.Config
	if insideContainer() {
		logCfg = zap.NewProductionConfig()
	} else {
		logCfg = zap.NewDevelopmentConfig()
		logCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	logCfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)

	logger, err := logCfg.Build(opts...)
	if err != nil {
		log.Panicf("could not create logger: %v", err)
	}

	return logger
}

func New() *Logger {
	if cachedLogger != nil {
		return cachedLogger
	}

	logger := defaultLogger()

	logOnce.Do(func() {
		cachedLogger = &Logger{
			log: logger,
		}
	})

	return cachedLogger
}

func FromContext(ctx context.Context) *Logger {
	if ctx == nil {
		return New()
	}

	if l, ok := ctx.Value(LoggerCtxKey{}).(*Logger); ok {
		return l
	}

	return New()
}

func (l Logger) DPanic(msg string, fields ...Field) {
	l.log.DPanic(msg, fields...)
}

func (l Logger) Debug(msg string, fields ...Field) {
	l.log.Debug(msg, fields...)
}

func (l Logger) Error(msg string, fields ...Field) {
	l.log.Error(msg, fields...)
}

func (l Logger) Fatal(msg string, fields ...Field) {
	l.log.Fatal(msg, fields...)
}

func (l Logger) Info(msg string, fields ...Field) {
	l.log.Info(msg, fields...)
}

func (l Logger) Panic(msg string, fields ...Field) {
	l.log.Panic(msg, fields...)
}

func (l Logger) Sync() error {
	return l.log.Sync()
}

func (l Logger) Warn(msg string, fields ...Field) {
	l.log.Warn(msg, fields...)
}

func (l Logger) With(fields ...Field) *Logger {
	logger := l.log.With(fields...)
	return &Logger{
		log: logger,
	}
}

func (l Logger) WithOptions(opts ...Option) *Logger {
	logger := l.log.WithOptions(opts...)
	return &Logger{
		log: logger,
	}
}

func (l *Logger) GetContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, LoggerCtxKey{}, l)
}
