package handlers

import (
	"time"
	"vocalin-backend/internal/models"
	"vocalin-backend/internal/response"
	"vocalin-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *service.AuthService
}

type RegisterRequest struct {
	Nickname        string `json:"nickname" binding:"required,min=2,max=50"`
	Phone           string `json:"phone" binding:"required,min=6,max=20"`
	Password        string `json:"password" binding:"required,min=6,max=72"`
	ConfirmPassword string `json:"confirm_password" binding:"required,min=6,max=72"`
}

type LoginRequest struct {
	Nickname string `json:"nickname" binding:"required,min=2,max=50"`
	Password string `json:"password" binding:"required,min=6,max=72"`
}

type LoginResponse struct {
	AccessToken           string      `json:"access_token"`
	AccessTokenExpiresAt  time.Time   `json:"access_token_expires_at"`
	RefreshToken          string      `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time   `json:"refresh_token_expires_at"`
	User                  models.User `json:"user"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type RefreshTokenResponse struct {
	AccessToken           string    `json:"access_token"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshToken          string    `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register godoc
// @Summary 用户注册
// @Description 使用昵称、手机号和密码创建账号，成功后直接返回 JWT
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Register Request"
// @Success 200 {object} response.APIResponse{data=LoginResponse}
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}
	result, err := h.authService.Register(c.Request.Context(), req.Nickname, req.Phone, req.Password, req.ConfirmPassword)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, "注册成功", LoginResponse{
		AccessToken:           result.AccessToken,
		AccessTokenExpiresAt:  result.AccessTokenExpiresAt,
		RefreshToken:          result.RefreshToken,
		RefreshTokenExpiresAt: result.RefreshTokenExpiresAt,
		User:                  *result.User,
	})
}

// Login godoc
// @Summary 用户登录
// @Description 当前使用昵称和密码登录，后续可扩展手机号验证码登录
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login Request"
// @Success 200 {object} response.APIResponse{data=LoginResponse}
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}

	result, err := h.authService.Login(c.Request.Context(), req.Nickname, req.Password)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response.Success(c, "登录成功", LoginResponse{
		AccessToken:           result.AccessToken,
		AccessTokenExpiresAt:  result.AccessTokenExpiresAt,
		RefreshToken:          result.RefreshToken,
		RefreshTokenExpiresAt: result.RefreshTokenExpiresAt,
		User:                  *result.User,
	})
}

// Refresh godoc
// @Summary 刷新访问令牌
// @Description 使用 refresh token 换取新的 access token 和 refresh token
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "Refresh Token Request"
// @Success 200 {object} response.APIResponse{data=RefreshTokenResponse}
// @Router /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}
	result, err := h.authService.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, "刷新令牌成功", RefreshTokenResponse{
		AccessToken:           result.AccessToken,
		AccessTokenExpiresAt:  result.AccessTokenExpiresAt,
		RefreshToken:          result.RefreshToken,
		RefreshTokenExpiresAt: result.RefreshTokenExpiresAt,
	})
}

// Logout godoc
// @Summary 登出当前会话
// @Description 撤销当前 refresh token，使其无法再次刷新访问令牌
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body LogoutRequest true "Logout Request"
// @Success 200 {object} response.APIResponse
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}
	if err := h.authService.Logout(c.Request.Context(), currentUserID(c), req.RefreshToken); err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, "登出成功", nil)
}
