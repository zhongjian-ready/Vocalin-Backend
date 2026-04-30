package response

import "github.com/gin-gonic/gin"

// APIResponse 统一规范接口响应结构，便于前端和文档保持一致。
type APIResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Meta    any    `json:"meta,omitempty"`
}

func JSON(c *gin.Context, status int, code, message string, data any, meta any) {
	c.JSON(status, APIResponse{
		Code:    code,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

func Success(c *gin.Context, message string, data any) {
	JSON(c, 200, "SUCCESS", message, data, nil)
}

func Error(c *gin.Context, status int, code, message string) {
	JSON(c, status, code, message, nil, nil)
}
