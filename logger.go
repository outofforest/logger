package logger

import (
	"context"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type logFiedType int

const logField logFiedType = iota

var mu sync.Mutex
var verbose = true

// EncoderConfig is the config of log encoder
var EncoderConfig = zapcore.EncoderConfig{
	TimeKey:        "ts",
	LevelKey:       "level",
	NameKey:        "logger",
	CallerKey:      "caller",
	FunctionKey:    zapcore.OmitKey,
	MessageKey:     "msg",
	StacktraceKey:  "stack",
	LineEnding:     zapcore.DefaultLineEnding,
	EncodeLevel:    zapcore.LowercaseLevelEncoder,
	EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
	EncodeDuration: zapcore.SecondsDurationEncoder,
	EncodeCaller:   zapcore.ShortCallerEncoder,
}

// New creates new logger
func New() *zap.Logger {
	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
		Development:      true,
		Encoding:         "console",
		EncoderConfig:    EncoderConfig,
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	log, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	return log
}

// With adds new logger to context
func With(ctx context.Context, fields ...zap.Field) context.Context {
	return context.WithValue(ctx, logField, Get(ctx).With(fields...))
}

// Get gets logger from context
func Get(ctx context.Context) *zap.Logger {
	mu.Lock()
	defer mu.Unlock()

	var log = ctx.Value(logField).(*zap.Logger)
	if !verbose {
		log = log.WithOptions(zap.IncreaseLevel(zapcore.InfoLevel))
	}
	return log
}

// WithLogger adds existing logger to context
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, logField, logger)
}

// VerboseOff turns off verbose logging
func VerboseOff() {
	mu.Lock()
	defer mu.Unlock()

	verbose = false
}
