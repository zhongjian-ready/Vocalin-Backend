package database

import (
	"fmt"
	"time"
	"vocalin-backend/internal/config"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// New 创建数据库连接，并统一配置连接池和 GORM 日志行为。
func New(cfg config.DatabaseConfig, logger *zap.Logger) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: newGORMLogger(logger),
	})
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("open sql database: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	logger.Info("数据库连接成功",
		zap.String("host", cfg.Host),
		zap.String("port", cfg.Port),
		zap.String("name", cfg.Name),
	)

	return db, nil
}

type gormZapWriter struct {
	logger *zap.Logger
}

func (w *gormZapWriter) Printf(format string, args ...any) {
	w.logger.Sugar().Debugf(format, args...)
}

func newGORMLogger(logger *zap.Logger) gormlogger.Interface {
	return gormlogger.New(&gormZapWriter{logger: logger.Named("gorm")}, gormlogger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  gormlogger.Warn,
		IgnoreRecordNotFoundError: true,
		Colorful:                  false,
	})
}
