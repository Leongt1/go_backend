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

// caller extracts the authenticated user's ID and role from the request context.
func caller(c *gin.Context) (uuid.UUID, string, error) {
	idStr, exists := c.Get(middleware.ContextUserID)
	if !exists {
		return uuid.Nil, "", domain.ErrInvalidInput
	}
	id, err := uuid.Parse(idStr.(string))
	if err != nil {
		return uuid.Nil, "", domain.ErrInvalidInput
	}
	role, _ := c.Get(middleware.ContextRole)
	roleStr, _ := role.(string)
	return id, roleStr, nil
}

func (h *UserHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	callerID, callerRole, err := caller(c)
	if err != nil {
		c.Error(err)
		return
	}
	if callerRole != string(domain.RoleAdmin) && callerID != id {
		c.Error(domain.ErrForbidden)
		return
	}

	user, err := h.service.GetByID(c.Request.Context(), id)
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

	updatedBy, callerRole, err := caller(c)
	if err != nil {
		c.Error(err)
		return
	}

	// Non-admins may only update themselves and can never change roles
	if callerRole != string(domain.RoleAdmin) {
		if updatedBy != id {
			c.Error(domain.ErrForbidden)
			return
		}
		req.Role = nil
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

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}
