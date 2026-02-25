package handler

import (
	"backend-go/internal/users/service"

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
	c.JSON(200, users)
}

func (h *UserHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	user, err := h.service.GetByID(c.Request.Context(), uuid.Must(uuid.Parse(id)))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(200, user)
}
