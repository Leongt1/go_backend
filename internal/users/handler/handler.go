package handler

import (
	"backend-go/internal/platform/middleware"
	"backend-go/internal/users/domain"
	"backend-go/internal/users/service"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserHandler struct {
	service *service.Service
}

func NewUserHandler(service *service.Service) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) ListUsers(c *gin.Context) {
	users, err := h.service.ListUsers(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, users)
}

func (h *UserHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}
	user, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) GetUserByEmail(c *gin.Context) {
	email := c.Param("email")
	user, err := h.service.GetByEmail(c.Request.Context(), email)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, user)
}

type UpdateRequest struct {
	Name        *string    `json:"name"`
	Role        *string    `json:"role"`
	Gender      *string    `json:"gender"`
	DateOfBirth *time.Time `json:"date_of_birth"`
}

func (h *UserHandler) Update(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	updatedByStr, exists := c.Get(middleware.ContextUserID)
	if !exists {
		c.Error(domain.ErrInvalidInput)
		return
	}
	updatedBy, err := uuid.Parse(updatedByStr.(string))
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	// Updating user using domain method
	_, err = h.service.UpdateUser(c.Request.Context(), id, &service.UpdateInput{
		Name:        req.Name,
		Role:        req.Role,
		Gender:      req.Gender,
		DateOfBirth: req.DateOfBirth,
		UpdatedBy:   &updatedBy,
	})
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	if err := h.service.DeleteUser(c.Request.Context(), id); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusNoContent, gin.H{"message": "User deleted successfully"})
}
