package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestRequestLoggerLogsSanitizedRequestContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	core, observedLogs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	router := gin.New()
	router.Use(RequestLogger(logger))
	router.POST("/api/items/:id", func(c *gin.Context) {
		var payload map[string]any
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		if payload["name"] != "demo" {
			c.Status(http.StatusBadRequest)
			return
		}
		c.Status(http.StatusNoContent)
	})

	body, err := json.Marshal(map[string]any{
		"name":          "demo",
		"refresh_token": "secret-refresh-token",
	})
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/items/42?page=1&token=secret-token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, resp.Code)
	}

	entries := observedLogs.AllUntimed()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	fields := entries[0].ContextMap()
	if fields["method"] != http.MethodPost {
		t.Fatalf("expected method POST, got %#v", fields["method"])
	}
	query, ok := fields["query"].(map[string]any)
	if !ok {
		t.Fatalf("expected query to be logged, got %#v", fields["query"])
	}
	if query["page"] != "1" {
		t.Fatalf("expected page query to be logged, got %#v", query["page"])
	}
	if query["token"] != "[REDACTED]" {
		t.Fatalf("expected token query to be redacted, got %#v", query["token"])
	}
	pathParams, ok := fields["path_params"].(map[string]any)
	if !ok {
		t.Fatalf("expected path params to be logged, got %#v", fields["path_params"])
	}
	if pathParams["id"] != "42" {
		t.Fatalf("expected id path param to be logged, got %#v", pathParams["id"])
	}
	loggedBody, ok := fields["body"].(map[string]any)
	if !ok {
		t.Fatalf("expected body to be logged, got %#v", fields["body"])
	}
	if loggedBody["name"] != "demo" {
		t.Fatalf("expected request body to preserve non-sensitive values, got %#v", loggedBody["name"])
	}
	if loggedBody["refresh_token"] != "[REDACTED]" {
		t.Fatalf("expected refresh token to be redacted, got %#v", loggedBody["refresh_token"])
	}
	if fields["status"] != int64(http.StatusNoContent) {
		t.Fatalf("expected status %d, got %#v", http.StatusNoContent, fields["status"])
	}
}
