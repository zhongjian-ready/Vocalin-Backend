package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"vocalin-backend/internal/auth"
	"vocalin-backend/internal/response"
)

// AuthMiddleware 负责解析 Bearer Token，并将用户身份写入上下文。
func AuthMiddleware(tokenManager *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		if authorization == "" {
			response.Error(c, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "缺少 Authorization 请求头")
			c.Abort()
			return
		}

		parts := strings.SplitN(authorization, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			response.Error(c, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authorization 格式应为 Bearer <token>")
			c.Abort()
			return
		}

		claims, err := tokenManager.ParseAccessToken(strings.TrimSpace(parts[1]))
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "无效或已过期的访问令牌")
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("claims", claims)
		c.Next()
	}
}
