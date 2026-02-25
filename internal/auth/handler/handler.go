package handler

import (
	"backend-go/internal/auth/domain"
	"backend-go/internal/auth/service"
	"net/http"
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

	c.SetCookie(
		"refresh_token",
		resp.RefreshToken,
		int(h.refreshTTL.Seconds()),
		"/",
		"",
		false, // false for dev/testing
		true,  // HttpOnly
	)

	c.JSON(http.StatusOK, gin.H{
		"access_token": resp.AccessToken,
		"expires_in":   resp.ExpiresIn,
	})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	refreshTokenStr, err := c.Cookie("refresh_token")
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	resp, err := h.service.Refresh(c.Request.Context(), refreshTokenStr)
	if err != nil {
		c.Error(err)
		return
	}

	c.SetCookie(
		"refresh_token",
		resp.RefreshToken,
		int(h.refreshTTL.Seconds()),
		"/",
		"",
		false, // false for dev/testing
		true,  // HttpOnly
	)

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
	err = h.service.Logout(c.Request.Context(), refreshTokenStr)
	if err != nil {
		c.Error(err)
		return
	}

	// clear the refresh token cookie
	c.SetCookie(
		"refresh_token",
		"",
		-1,
		"/",
		"",
		false, // false for dev/testing
		true,  // HttpOnly
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}
