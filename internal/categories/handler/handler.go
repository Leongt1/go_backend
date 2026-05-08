package handler

import (
	"backend-go/internal/categories/domain"
	"backend-go/internal/categories/service"
	"backend-go/internal/platform/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CategoryHandler struct {
	service *service.Service
}

func NewCategoryHandler(service *service.Service) *CategoryHandler {
	return &CategoryHandler{service: service}
}

func (h *CategoryHandler) List(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		c.Error(err)
		return
	}

	categories, err := h.service.ListForUser(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, categories)
}

type CreateRequest struct {
	Name string `json:"name" binding:"required"`
	Icon string `json:"icon"`
}

func (h *CategoryHandler) Create(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		c.Error(err)
		return
	}

	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	if err := h.service.Create(c.Request.Context(), userID, req.Name, req.Icon); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Category created successfully"})
}

type RenameRequest struct {
	Icon *string `json:"icon"`
	Name string  `json:"name" binding:"required"`
}

func (h *CategoryHandler) Rename(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		c.Error(err)
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	var req RenameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	if err := h.service.RenameCategory(c.Request.Context(), userID, id, req.Name, req.Icon); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category renamed successfully"})
}

func (h *CategoryHandler) Hide(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		c.Error(err)
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	if err := h.service.HideCategory(c.Request.Context(), userID, id); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category hidden successfully"})
}

func (h *CategoryHandler) Unhide(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		c.Error(err)
		return
	}

	categoryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	if err := h.service.Unhide(c.Request.Context(), userID, categoryID); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category unhidden successfully"})
}

func (h *CategoryHandler) Delete(c *gin.Context) {
	userId, err := getUserID(c)
	if err != nil {
		c.Error(err)
		return
	}

	categoryId, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	if err := h.service.DeleteCategory(c.Request.Context(), userId, categoryId); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "category deleted successfully",
	})
}

func getUserID(c *gin.Context) (uuid.UUID, error) {
	userIDStr, exists := c.Get(middleware.ContextUserID)
	if !exists {
		return uuid.Nil, domain.ErrInvalidInput
	}

	return uuid.Parse(userIDStr.(string))
}
