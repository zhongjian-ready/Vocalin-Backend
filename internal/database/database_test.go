package database

import (
	"strings"
	"testing"
	"time"

	"vocalin-backend/internal/config"
)

func TestBuildDSNUsesShanghaiTimezone(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("LoadLocation() error = %v", err)
	}

	dsn := buildDSN(config.DatabaseConfig{
		Host:     "127.0.0.1",
		Port:     "3306",
		User:     "root",
		Password: "secret",
		Name:     "vocalin",
	}, location)

	if !strings.Contains(dsn, "loc=Asia%2FShanghai") {
		t.Fatalf("dsn = %q, want loc=Asia%%2FShanghai", dsn)
	}

	if !strings.Contains(dsn, "parseTime=True") {
		t.Fatalf("dsn = %q, want parseTime=True", dsn)
	}
}
