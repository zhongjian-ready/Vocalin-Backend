package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"vocalin-backend/internal/response"
	"vocalin-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

const userIDContextKey = "userID"

func currentUserID(c *gin.Context) uint {
	return c.MustGet(userIDContextKey).(uint)
}

func writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrUnauthorized):
		response.Error(c, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", err.Error())
	case errors.Is(err, service.ErrInvalidCredentials):
		response.Error(c, http.StatusUnauthorized, "AUTH_INVALID_CREDENTIALS", err.Error())
	case errors.Is(err, service.ErrInvalidRefreshToken), errors.Is(err, service.ErrRefreshTokenRevoked):
		response.Error(c, http.StatusUnauthorized, "AUTH_REFRESH_TOKEN_INVALID", err.Error())
	case errors.Is(err, service.ErrNicknameAlreadyExists), errors.Is(err, service.ErrPhoneAlreadyExists):
		response.Error(c, http.StatusConflict, "AUTH_REGISTER_CONFLICT", err.Error())
	case errors.Is(err, service.ErrPasswordMismatch):
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, service.ErrUserNotInGroup), errors.Is(err, service.ErrTimedNoteRequiresShowAt):
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
	case errors.Is(err, service.ErrUserAlreadyInGroup):
		response.Error(c, http.StatusConflict, "GROUP_CONFLICT", err.Error())
	case errors.Is(err, service.ErrInvalidInviteCode), errors.Is(err, service.ErrGroupNotFound), errors.Is(err, service.ErrWishlistItemNotFound):
		response.Error(c, http.StatusNotFound, "RESOURCE_NOT_FOUND", err.Error())
	case errors.Is(err, service.ErrForbidden):
		response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "服务器内部错误")
	}
}

func writeBindError(c *gin.Context, err error) {
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		messages := make([]string, 0, len(validationErrors))
		for _, fieldErr := range validationErrors {
			messages = append(messages, humanizeValidationError(fieldErr))
		}
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", strings.Join(messages, "; "))
		return
	}
	if strings.TrimSpace(err.Error()) == "EOF" {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "请求体不能为空")
		return
	}
	response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
}

func parsePagination(c *gin.Context) (page int, pageSize int) {
	pagination := service.NewPagination(
		parsePositiveInt(c.DefaultQuery("page", "1"), 1),
		parsePositiveInt(c.DefaultQuery("page_size", "20"), 20),
	)
	return pagination.Page, pagination.PageSize
}

func parsePositiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func humanizeValidationError(fieldErr validator.FieldError) string {
	fieldName := fieldErr.Field()
	switch fieldErr.Tag() {
	case "required":
		return fmt.Sprintf("字段 %s 为必填项", fieldName)
	case "max":
		return fmt.Sprintf("字段 %s 超过最大长度限制", fieldName)
	case "min":
		return fmt.Sprintf("字段 %s 未达到最小长度要求", fieldName)
	case "url":
		return fmt.Sprintf("字段 %s 不是合法 URL", fieldName)
	case "invite_code":
		return fmt.Sprintf("字段 %s 不是合法邀请码", fieldName)
	case "note_type":
		return fmt.Sprintf("字段 %s 仅支持 normal、burn、timed", fieldName)
	default:
		return fmt.Sprintf("字段 %s 校验失败", fieldName)
	}
}
