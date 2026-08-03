package handler

import (
	"backend-go/internal/ai/domain"
	"backend-go/internal/ai/service"
	"backend-go/internal/platform/middleware"
	usersDomain "backend-go/internal/users/domain"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AIHandler struct {
	service *service.Service
}

func NewAIHandler(service *service.Service) *AIHandler {
	return &AIHandler{service: service}
}

type ChatTurnRequest struct {
	Role    string `json:"role" binding:"required,oneof=user assistant"`
	Content string `json:"content" binding:"required"`
}

type ChatRequest struct {
	Message string            `json:"message" binding:"required"`
	History []ChatTurnRequest `json:"history"`
}

type ChatResponse struct {
	Reply            string   `json:"reply"`
	Actions          []string `json:"actions"`
	CreditsRemaining int      `json:"credits_remaining"`
}

func (h *AIHandler) Chat(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	history := make([]service.ChatTurn, len(req.History))
	for i, turn := range req.History {
		history[i] = service.ChatTurn{Role: turn.Role, Content: turn.Content}
	}

	out, err := h.service.Chat(c.Request.Context(), &service.ChatInput{
		UserID:  userID,
		Message: req.Message,
		History: history,
		IsAdmin: isAdmin(c),
	})
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, ChatResponse{
		Reply:            out.Reply,
		Actions:          out.Actions,
		CreditsRemaining: out.CreditsRemaining,
	})
}

func (h *AIHandler) Credits(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	// admins have unlimited credits; report the sentinel so the UI shows
	// "Unlimited" instead of a number.
	if isAdmin(c) {
		c.JSON(http.StatusOK, gin.H{"credits": service.CreditsUnlimited})
		return
	}

	credits, err := h.service.GetCredits(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"credits": credits})
}

func getUserID(c *gin.Context) (uuid.UUID, error) {
	userIDStr, exists := c.Get(middleware.ContextUserID)
	if !exists {
		return uuid.Nil, domain.ErrInvalidInput
	}
	return uuid.Parse(userIDStr.(string))
}

// isAdmin reports whether the authenticated caller has the Admin role, read
// from the JWT claims the auth middleware placed on the context.
func isAdmin(c *gin.Context) bool {
	role, exists := c.Get(middleware.ContextRole)
	if !exists {
		return false
	}
	roleStr, ok := role.(string)
	return ok && roleStr == string(usersDomain.RoleAdmin)
}
