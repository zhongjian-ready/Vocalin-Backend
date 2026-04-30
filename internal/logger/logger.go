package logger

import (
	"fmt"
	"os"
	"strings"

	"vocalin-backend/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New 根据配置创建统一日志实例，供 HTTP、业务与基础设施层复用。
func New(cfg config.LogConfig) (*zap.Logger, error) {
	level := zap.NewAtomicLevel()
	if err := level.UnmarshalText([]byte(strings.ToLower(cfg.Level))); err != nil {
		return nil, fmt.Errorf("parse log level: %w", err)
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "time"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.LevelKey = "level"

	encoder := zapcore.NewJSONEncoder(encoderCfg)
	if strings.EqualFold(cfg.Format, "console") {
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	}

	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
	logger := zap.New(core, zap.AddCaller())

	if strings.EqualFold(cfg.Format, "json") {
		return zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), zapcore.AddSync(os.Stdout), level), zap.AddCaller()), nil
	}

	return logger, nil
}
