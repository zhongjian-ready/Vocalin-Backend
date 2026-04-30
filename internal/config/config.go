package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config 聚合整个应用的基础配置，避免在业务层直接读取环境变量。
type Config struct {
	App      AppConfig
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Log      LogConfig
}

type AppConfig struct {
	Name        string
	Environment string
}

type ServerConfig struct {
	Port         string
	Mode         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
}

type AuthConfig struct {
	JWTSecret       string
	Issuer          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	ClockSkew       time.Duration
}

type LogConfig struct {
	Level      string
	Format     string
	Stacktrace bool
}

// Load 使用 Viper 统一加载配置，并保留 .env 的本地开发体验。
func Load() (*Config, error) {
	_ = godotenv.Load()

	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)

	if configFile := v.GetString("config.file"); configFile != "" {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("read config file: %w", err)
		}
	}

	cfg := &Config{
		App: AppConfig{
			Name:        v.GetString("app.name"),
			Environment: v.GetString("app.environment"),
		},
		Server: ServerConfig{
			Port:         v.GetString("server.port"),
			Mode:         v.GetString("server.mode"),
			ReadTimeout:  v.GetDuration("server.read_timeout"),
			WriteTimeout: v.GetDuration("server.write_timeout"),
		},
		Database: DatabaseConfig{
			Host:            v.GetString("database.host"),
			Port:            v.GetString("database.port"),
			User:            v.GetString("database.user"),
			Password:        v.GetString("database.password"),
			Name:            v.GetString("database.name"),
			MaxIdleConns:    v.GetInt("database.max_idle_conns"),
			MaxOpenConns:    v.GetInt("database.max_open_conns"),
			ConnMaxLifetime: v.GetDuration("database.conn_max_lifetime"),
		},
		Auth: AuthConfig{
			JWTSecret:       v.GetString("auth.jwt_secret"),
			Issuer:          v.GetString("auth.issuer"),
			AccessTokenTTL:  v.GetDuration("auth.access_token_ttl"),
			RefreshTokenTTL: v.GetDuration("auth.refresh_token_ttl"),
			ClockSkew:       v.GetDuration("auth.clock_skew"),
		},
		Log: LogConfig{
			Level:      v.GetString("log.level"),
			Format:     v.GetString("log.format"),
			Stacktrace: v.GetBool("log.stacktrace"),
		},
	}

	if cfg.Auth.JWTSecret == "" && !strings.EqualFold(cfg.App.Environment, "production") {
		cfg.Auth.JWTSecret = "dev-only-jwt-secret-change-me"
	}

	if cfg.Auth.JWTSecret == "" {
		return nil, fmt.Errorf("auth.jwt_secret is required")
	}

	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "Vocalin Backend")
	v.SetDefault("app.environment", "development")

	v.SetDefault("server.port", "8080")
	v.SetDefault("server.mode", "debug")
	v.SetDefault("server.read_timeout", "15s")
	v.SetDefault("server.write_timeout", "15s")

	v.SetDefault("database.host", "127.0.0.1")
	v.SetDefault("database.port", "3306")
	v.SetDefault("database.user", "root")
	v.SetDefault("database.password", "")
	v.SetDefault("database.name", "vocalin")
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.max_open_conns", 20)
	v.SetDefault("database.conn_max_lifetime", "30m")

	v.SetDefault("auth.issuer", "vocalin-backend")
	v.SetDefault("auth.jwt_secret", "")
	v.SetDefault("auth.access_token_ttl", "72h")
	v.SetDefault("auth.refresh_token_ttl", "720h")
	v.SetDefault("auth.clock_skew", "1m")

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "console")
	v.SetDefault("log.stacktrace", false)
	// 允许通过 CONFIG_FILE 指定配置文件。
	v.BindEnv("config.file", "CONFIG_FILE")
}
