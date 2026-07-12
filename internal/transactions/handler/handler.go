package handler

import (
	"backend-go/internal/platform/middleware"
	"backend-go/internal/transactions/domain"
	"backend-go/internal/transactions/service"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TransactionHandler struct {
	service *service.Service
}

func NewTransactionHandler(service *service.Service) *TransactionHandler {
	return &TransactionHandler{service: service}
}

// converts frontend rupees to backend paisa
func amountToInt64(amount float64) int64 {
	return int64(math.Round(amount * 100))
}

// converts paisa from storage to frontend rupees
func amountToFloat64(amount int64) float64 {
	return float64(amount) / 100
}

type CreateTransactionRequest struct {
	CategoryID  uuid.UUID `json:"category_id" binding:"required"`
	Amount      float64   `json:"amount" binding:"required"`
	Description *string   `json:"description"`
	Type        string    `json:"type" binding:"required"`
	Date        time.Time `json:"date" binding:"required"`
}

func (h *TransactionHandler) Create(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	var req CreateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	if req.Amount <= 0 {
		c.Error(domain.ErrInvalidInput)
		return
	}

	err = h.service.CreateTransaction(c.Request.Context(), service.CreateInput{
		UserID:      userID,
		CategoryID:  req.CategoryID,
		Amount:      amountToInt64(req.Amount),
		Description: req.Description,
		Type:        req.Type,
		Date:        req.Date,
	})
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Transaction created successfully"})
}

func (h *TransactionHandler) GetByID(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		c.Error(domain.ErrInvalidAmount)
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(domain.ErrInvalidAmount)
		return
	}

	tx, err := h.service.GetByID(c.Request.Context(), userID, id)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, toResponse(tx))
}

func (h *TransactionHandler) List(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	// parse optional query params
	input := &service.ListInput{UserID: userID}

	if cat := c.Query("category_id"); cat != "" {
		catID, err := uuid.Parse(cat)
		if err != nil {
			c.Error(domain.ErrInvalidInput)
			return
		}
		input.CategoryID = &catID
	}

	if t := c.Query("type"); t != "" {
		input.Type = &t
	}

	if from := c.Query("date_from"); from != "" {
		parsed, err := time.Parse(time.RFC3339, from)
		if err != nil {
			c.Error(domain.ErrInvalidInput)
			return
		}
		input.DateFrom = &parsed
	}

	if to := c.Query("date_to"); to != "" {
		parsed, err := time.Parse(time.RFC3339, to)
		if err != nil {
			c.Error(domain.ErrInvalidInput)
			return
		}
		input.DateTo = &parsed
	}

	if l := c.Query("limit"); l != "" {
		limit, err := strconv.Atoi(l)
		if err != nil || limit < 1 || limit > maxPageSize {
			c.Error(domain.ErrInvalidInput)
			return
		}
		input.Limit = &limit
	}

	if o := c.Query("offset"); o != "" {
		offset, err := strconv.Atoi(o)
		if err != nil || offset < 0 {
			c.Error(domain.ErrInvalidInput)
			return
		}
		input.Offset = offset
	}

	result, err := h.service.ListTransactions(c.Request.Context(), input)
	if err != nil {
		c.Error(err)
		return
	}

	// convert each transaction amount before returning
	response := make([]TransactionResponse, len(result.Transactions))
	for i, tx := range result.Transactions {
		response[i] = toResponse(&tx)
	}

	// no limit param = legacy bare-array response; with limit = paginated envelope
	if input.Limit == nil {
		c.JSON(http.StatusOK, response)
		return
	}

	c.JSON(http.StatusOK, ListTransactionsResponse{
		Transactions: response,
		Total:        result.Total,
		Limit:        *input.Limit,
		Offset:       input.Offset,
	})
}

// maxPageSize caps the page size a client may request.
const maxPageSize = 100

type ListTransactionsResponse struct {
	Transactions []TransactionResponse `json:"transactions"`
	Total        int64                 `json:"total"`
	Limit        int                   `json:"limit"`
	Offset       int                   `json:"offset"`
}

type UpdateRequest struct {
	CategoryID  *uuid.UUID `json:"category_id"`
	Amount      *float64   `json:"amount"`
	Description *string    `json:"description"`
	Type        *string    `json:"type"`
	Date        *time.Time `json:"date"`
}

func (h *TransactionHandler) Update(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	// convert amount if provided
	var amountPaise *int64
	if req.Amount != nil {
		if *req.Amount <= 0 {
			c.Error(domain.ErrInvalidAmount)
			return
		}
		converted := amountToInt64(*req.Amount)
		amountPaise = &converted
	}

	tx, err := h.service.UpdateTransaction(c.Request.Context(), userID, id, &service.UpdateInput{
		CategoryID:  req.CategoryID,
		Amount:      amountPaise,
		Description: req.Description,
		Type:        req.Type,
		Date:        req.Date,
		UpdatedBy:   &userID,
	})
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, toResponse(tx))
}

func (h *TransactionHandler) Delete(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	if err := h.service.DeleteTransaction(c.Request.Context(), userID, id); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transaction deleted successfully"})
}

// TransactionResponse is what gets returned to the frontend
// amount is in rupees (float64), never paise
type TransactionResponse struct {
	ID          uuid.UUID              `json:"id"`
	CategoryID  uuid.UUID              `json:"category_id"`
	Amount      float64                `json:"amount"` // ← rupees
	Description *string                `json:"description"`
	Type        domain.TransactionType `json:"type"`
	Date        time.Time              `json:"date"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

func toResponse(tx *domain.Transaction) TransactionResponse {
	return TransactionResponse{
		ID:          tx.ID,
		CategoryID:  tx.CategoryID,
		Amount:      amountToFloat64(tx.Amount), // ← convert back here
		Description: tx.Description,
		Type:        tx.Type,
		Date:        tx.Date,
		CreatedAt:   tx.CreatedAt,
		UpdatedAt:   tx.UpdatedAt,
	}
}

func getUserID(c *gin.Context) (uuid.UUID, error) {
	userIDStr, exists := c.Get(middleware.ContextUserID)
	if !exists {
		return uuid.Nil, domain.ErrInvalidInput
	}
	return uuid.Parse(userIDStr.(string))
}
