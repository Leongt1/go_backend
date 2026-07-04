package handler

import (
	"backend-go/internal/auth/domain"
	"backend-go/internal/auth/service"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service    *service.Service
	refreshTTL time.Duration
}

func NewAuthHandler(auth *service.Service, refreshTTL time.Duration) *AuthHandler {
	return &AuthHandler{service: auth, refreshTTL: refreshTTL}
}

func (h *AuthHandler) setRefreshCookie(c *gin.Context, token string, maxAge int) {
	secure := os.Getenv("ENV") == "production"
	if secure {
		c.SetSameSite(http.SameSiteNoneMode)
	} else {
		c.SetSameSite(http.SameSiteLaxMode)
	}
	c.SetCookie(
		"refresh_token",
		token,
		maxAge,
		"/",
		"",
		secure,
		true,
	)
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	resp, err := h.service.Login(c.Request.Context(), &service.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		c.Error(err)
		return
	}

	h.setRefreshCookie(c, resp.RefreshToken, int(h.refreshTTL.Seconds()))

	c.JSON(http.StatusOK, gin.H{
		"access_token": resp.AccessToken,
		"expires_in":   resp.ExpiresIn,
	})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	refreshTokenStr, err := c.Cookie("refresh_token")
	if err != nil {
		c.Error(domain.ErrInvalidRefreshToken)
		return
	}

	resp, err := h.service.Refresh(c.Request.Context(), refreshTokenStr)
	if err != nil {
		c.Error(err)
		return
	}

	h.setRefreshCookie(c, resp.RefreshToken, int(h.refreshTTL.Seconds()))

	c.JSON(http.StatusOK, gin.H{
		"access_token": resp.AccessToken,
		"expires_in":   resp.ExpiresIn,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	refreshTokenStr, err := c.Cookie("refresh_token")
	if err != nil {
		c.Status(http.StatusNoContent) // pass even if cookie missing
		return
	}

	// revoke the refresh token
	_ = h.service.Logout(c.Request.Context(), refreshTokenStr)
	// if err != nil {
	// 	c.Error(err)
	// 	return
	// }

	// clear the refresh token cookie
	h.setRefreshCookie(c, "", -1)

	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

type SignupRequest struct {
	Name            string     `json:"name" binding:"required"`
	Email           string     `json:"email" binding:"required,email"`
	Password        string     `json:"password" binding:"required"`
	ConfirmPassword string     `json:"confirm_password" binding:"required"`
	Gender          string     `json:"gender" binding:"required"`
	DateOfBirth     *time.Time `json:"date_of_birth"`
}

func (h *AuthHandler) Signup(c *gin.Context) {
	var req SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	if req.Password != req.ConfirmPassword {
		c.Error(domain.ErrInvalidInput)
		return
	}

	err := h.service.Signup(c.Request.Context(), &service.SignupInput{
		Name:        req.Name,
		Email:       req.Email,
		Password:    req.Password,
		Gender:      req.Gender,
		DateOfBirth: req.DateOfBirth,
	})
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Signup successful",
	})
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	err := h.service.ForgotPassword(c.Request.Context(), &service.ForgotPasswordInput{
		Email: req.Email,
	})
	if err != nil {
		c.Error(domain.ErrPasswordResetFailed)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "If that email exists, a reset link has been sent",
	})
}

type ResetPasswordRequest struct {
	Password        string `json:"password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	rawToken := c.Query("token")
	if rawToken == "" {
		c.Error(domain.ErrInvalidCredentials)
		return
	}

	if req.Password != req.ConfirmPassword {
		c.Error(domain.ErrInvalidInput)
		return
	}

	if err := h.service.PasswordReset(c.Request.Context(), &service.PasswordResetInput{
		ResetToken: rawToken,
		Password:   req.Password,
	}); err != nil {
		if err == domain.ErrWeakPassword {
			c.Error(err)
		} else {
			c.Error(domain.ErrInvalidPasswordResetToken)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password reset successfully. Login now",
	})
}
