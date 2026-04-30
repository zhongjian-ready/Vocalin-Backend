package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const maxLoggedRequestBodyBytes = 4096

var sensitiveFieldNames = map[string]struct{}{
	"access_token":  {},
	"authorization": {},
	"password":      {},
	"refresh_token": {},
	"token":         {},
}

// RequestLogger 使用 Zap 记录 HTTP 请求摘要，便于排查线上问题。
func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
	requestLogger := logger.Named("http")
	return func(c *gin.Context) {
		startedAt := time.Now()
		requestFields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("raw_path", c.Request.URL.Path),
			zap.String("client_ip", c.ClientIP()),
		}

		if fullPath := c.FullPath(); fullPath != "" {
			requestFields = append(requestFields, zap.String("path", fullPath))
		}
		if rawQuery := c.Request.URL.RawQuery; rawQuery != "" {
			requestFields = append(requestFields, zap.Any("query", sanitizeValues(c.Request.URL.Query())))
		}
		if len(c.Params) > 0 {
			requestFields = append(requestFields, zap.Any("path_params", sanitizeParams(c.Params)))
		}
		if bodyField, ok := buildRequestBodyField(c.Request); ok {
			requestFields = append(requestFields, bodyField)
		}

		c.Next()

		requestLogger.Info("HTTP 请求",
			append(requestFields,
				zap.Int("status", c.Writer.Status()),
				zap.Duration("latency", time.Since(startedAt)),
			)...,
		)
	}
}

func buildRequestBodyField(r *http.Request) (zap.Field, bool) {
	if r == nil || r.Body == nil || r.ContentLength == 0 {
		return zap.Skip(), false
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		mediaType = ""
	}

	if !shouldLogBody(mediaType) {
		return zap.Skip(), false
	}

	payload, truncated, err := readAndRestoreBody(r)
	if err != nil || len(payload) == 0 {
		return zap.Skip(), false
	}

	if strings.EqualFold(mediaType, "application/json") {
		var decoded any
		if err := json.Unmarshal(payload, &decoded); err == nil {
			body := sanitizeJSON(decoded)
			if truncated {
				return zap.Any("body", map[string]any{
					"payload":   body,
					"truncated": true,
				}), true
			}
			return zap.Any("body", body), true
		}
	}

	body := sanitizeRawBody(string(payload))
	if truncated {
		return zap.Any("body", map[string]any{
			"payload":   body,
			"truncated": true,
		}), true
	}
	return zap.String("body", body), true
}

func shouldLogBody(mediaType string) bool {
	switch {
	case strings.EqualFold(mediaType, "application/json"):
		return true
	case strings.HasPrefix(mediaType, "text/"):
		return true
	case strings.EqualFold(mediaType, "application/x-www-form-urlencoded"):
		return true
	default:
		return false
	}
}

func readAndRestoreBody(r *http.Request) ([]byte, bool, error) {
	limited := io.LimitReader(r.Body, maxLoggedRequestBodyBytes+1)
	payload, err := io.ReadAll(limited)
	if err != nil {
		return nil, false, err
	}
	truncated := len(payload) > maxLoggedRequestBodyBytes
	if truncated {
		payload = payload[:maxLoggedRequestBodyBytes]
	}
	r.Body = io.NopCloser(bytes.NewReader(payload))
	return payload, truncated, nil
}

func sanitizeValues(values map[string][]string) map[string]any {
	if len(values) == 0 {
		return nil
	}

	sanitized := make(map[string]any, len(values))
	for key, item := range values {
		if isSensitiveField(key) {
			sanitized[key] = "[REDACTED]"
			continue
		}
		if len(item) == 1 {
			sanitized[key] = item[0]
			continue
		}
		copied := make([]string, len(item))
		copy(copied, item)
		for index := range copied {
			copied[index] = sanitizeRawBody(copied[index])
		}
		if len(copied) == 1 {
			sanitized[key] = copied[0]
			continue
		}
		sanitized[key] = copied
	}
	return sanitized
}

func sanitizeParams(params gin.Params) map[string]any {
	if len(params) == 0 {
		return nil
	}

	sanitized := make(map[string]any, len(params))
	for _, param := range params {
		if isSensitiveField(param.Key) {
			sanitized[param.Key] = "[REDACTED]"
			continue
		}
		sanitized[param.Key] = sanitizeRawBody(param.Value)
	}
	return sanitized
}

func sanitizeJSON(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		sanitized := make(map[string]any, len(typed))
		for key, item := range typed {
			if isSensitiveField(key) {
				sanitized[key] = "[REDACTED]"
				continue
			}
			sanitized[key] = sanitizeJSON(item)
		}
		return sanitized
	case []any:
		sanitized := make([]any, len(typed))
		for index, item := range typed {
			sanitized[index] = sanitizeJSON(item)
		}
		return sanitized
	case string:
		return sanitizeRawBody(typed)
	default:
		return typed
	}
}

func sanitizeRawBody(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return trimmed
	}
	if len(trimmed) > maxLoggedRequestBodyBytes {
		return trimmed[:maxLoggedRequestBodyBytes]
	}
	return trimmed
}

func isSensitiveField(key string) bool {
	_, ok := sensitiveFieldNames[strings.ToLower(strings.TrimSpace(key))]
	return ok
}
