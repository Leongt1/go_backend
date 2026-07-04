package handler

import (
	"backend-go/internal/budgets/domain"
	"backend-go/internal/budgets/service"
	"backend-go/internal/platform/middleware"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BudgetHandler struct {
	service *service.Service
}

func NewBudgetHandler(svc *service.Service) *BudgetHandler {
	return &BudgetHandler{service: svc}
}

type CreateBudgetRequest struct {
	Name        string   `json:"name" binding:"required"`
	Type        string   `json:"type" binding:"required,oneof=category overall"`
	Kind        string   `json:"kind" binding:"required,oneof=expense savings"`
	Amount      float64  `json:"amount" binding:"required,gt=0"`
	PeriodUnit  string   `json:"period_unit" binding:"required,oneof=day week month year"`
	PeriodValue int      `json:"period_value" binding:"required,gt=0"`
	StartDate   string   `json:"start_date" binding:"required"` // ISO 8601 format
	CategoryIDs []string `json:"category_ids"`                  // Optional, category-type budgets
}

func (h *BudgetHandler) List(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		c.Error(err)
		return
	}

	budgets, err := h.service.ListByUser(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}

	response := make([]BudgetResponse, len(budgets))
	for i, b := range budgets {
		response[i] = budgetToResponse(b)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Budgets retrieved successfully", "budgets": response})
}

func (h *BudgetHandler) GetByID(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		c.Error(err)
		return
	}

	budgetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	budget, err := h.service.GetByID(c.Request.Context(), budgetID)
	if err != nil {
		c.Error(err)
		return
	}

	// Verify user owns this budget
	if budget.UserID != userID {
		c.Error(domain.ErrCannotModifyOther)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Budget retrieved successfully", "budget": budgetToResponse(*budget)})
}

func (h *BudgetHandler) Create(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		c.Error(err)
		return
	}

	var req CreateBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	// Parse start date
	startDate, err := time.Parse(time.RFC3339, req.StartDate)
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	// Create budget
	budget, err := h.service.Create(
		c.Request.Context(),
		userID,
		req.Name,
		domain.BudgetType(req.Type),
		domain.BudgetKind(req.Kind),
		rupeesToPaisa(req.Amount),
		domain.PeriodUnit(req.PeriodUnit),
		req.PeriodValue,
		startDate,
	)
	if err != nil {
		c.Error(err)
		return
	}

	// Add categories if provided
	if len(req.CategoryIDs) > 0 {
		for _, categoryIDStr := range req.CategoryIDs {
			categoryID, err := uuid.Parse(categoryIDStr)
			if err != nil {
				c.Error(domain.ErrInvalidInput)
				return
			}

			if err := h.service.AddCategoryToBudget(c.Request.Context(), budget.ID, categoryID, userID); err != nil {
				c.Error(err)
				return
			}
		}

		// Reload budget to populate categories
		budget, err = h.service.GetByID(c.Request.Context(), budget.ID)
		if err != nil {
			c.Error(err)
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Budget created successfully", "budget": budgetToResponse(*budget)})
}

type UpdateBudgetRequest struct {
	Name        *string            `json:"name"`
	Type        *domain.BudgetType `json:"type" binding:"omitempty,oneof=category overall"`
	Amount      *float64           `json:"amount" binding:"omitempty,gt=0"`
	PeriodUnit  *string            `json:"period_unit" binding:"omitempty,oneof=day week month year"`
	PeriodValue *int               `json:"period_value" binding:"omitempty,gt=0"`
	StartDate   *string            `json:"start_date"`
}

func (h *BudgetHandler) Update(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		c.Error(err)
		return
	}

	budgetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	var req UpdateBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	var bType *domain.BudgetType
	if req.Type != nil {
		bt := domain.BudgetType(*req.Type)
		bType = &bt
	}

	// Parse start date if provided
	var startDate *time.Time
	if req.StartDate != nil {
		sd, err := time.Parse(time.RFC3339, *req.StartDate)
		if err != nil {
			c.Error(domain.ErrInvalidInput)
			return
		}
		startDate = &sd
	}

	// Parse period unit if provided
	var periodUnit *domain.PeriodUnit
	if req.PeriodUnit != nil {
		pu := domain.PeriodUnit(*req.PeriodUnit)
		periodUnit = &pu
	}

	var amountPaisa *int64
	if req.Amount != nil {
		val := rupeesToPaisa(*req.Amount)
		amountPaisa = &val
	}

	if err := h.service.Update(c.Request.Context(), budgetID,
		userID, req.Name,
		amountPaisa,
		periodUnit, req.PeriodValue,
		startDate, bType,
	); err != nil {
		c.Error(err)
		return
	}

	// Fetch updated budget
	budget, err := h.service.GetByID(c.Request.Context(), budgetID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Budget updated successfully", "budget": budgetToResponse(*budget)})
}

func (h *BudgetHandler) Delete(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		c.Error(err)
		return
	}

	budgetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	if err := h.service.Delete(c.Request.Context(), budgetID, userID); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Budget deleted successfully"})
}

type BudgetStatusResponse struct {
	BudgetID        string  `json:"budget_id"`
	Name            string  `json:"name"`
	BudgetAmount    float64 `json:"budget_amount"`
	Spent           float64 `json:"spent"`
	Remaining       float64 `json:"remaining"`
	ProgressPercent float64 `json:"progress_percent"`
	Status          string  `json:"status"`
	PeriodStart     string  `json:"period_start"`
	PeriodEnd       string  `json:"period_end"`
}

func (h *BudgetHandler) GetStatus(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		c.Error(err)
		return
	}

	budgetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	status, err := h.service.GetBudgetStatus(c.Request.Context(), budgetID, userID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Budget status retrieved successfully", "status": budgetStatusToResponse(status)})
}

type AddCategoryRequest struct {
	CategoryID string `json:"category_id" binding:"required"`
}

func (h *BudgetHandler) AddCategory(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		c.Error(err)
		return
	}

	budgetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	var req AddCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	if err := h.service.AddCategoryToBudget(c.Request.Context(), budgetID, categoryID, userID); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category added to budget successfully"})
}

func (h *BudgetHandler) RemoveCategory(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		c.Error(err)
		return
	}

	budgetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	categoryID, err := uuid.Parse(c.Param("categoryId"))
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	if err := h.service.RemoveCategoryFromBudget(c.Request.Context(), budgetID, categoryID, userID); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category removed from budget successfully"})
}

// Helper functions

func getUserID(c *gin.Context) (uuid.UUID, error) {
	userIDStr, exists := c.Get(middleware.ContextUserID)
	if !exists {
		return uuid.Nil, domain.ErrInvalidInput
	}

	return uuid.Parse(userIDStr.(string))
}

type BudgetResponse struct {
	ID          string   `json:"id"`
	UserID      string   `json:"user_id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Kind        string   `json:"kind"`
	Amount      float64  `json:"amount"`
	PeriodUnit  string   `json:"period_unit"`
	PeriodValue int      `json:"period_value"`
	StartDate   string   `json:"start_date"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	CategoryIDs []string `json:"category_ids,omitempty"`
}

func budgetToResponse(b domain.Budget) BudgetResponse {
	categoryIDs := make([]string, len(b.CategoryIDs))
	for i, id := range b.CategoryIDs {
		categoryIDs[i] = id.String()
	}

	return BudgetResponse{
		ID:          b.ID.String(),
		UserID:      b.UserID.String(),
		Name:        b.Name,
		Type:        string(b.Type),
		Kind:        string(b.Kind),
		Amount:      float64(b.Amount) / 100, // convert back to rupees for response
		PeriodUnit:  string(b.PeriodUnit),
		PeriodValue: b.PeriodValue,
		StartDate:   b.StartDate.Format(time.RFC3339),
		CreatedAt:   b.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   b.UpdatedAt.Format(time.RFC3339),
		CategoryIDs: categoryIDs,
	}
}

func budgetStatusToResponse(status *service.BudgetStatus) BudgetStatusResponse {
	return BudgetStatusResponse{
		BudgetID:        status.BudgetID.String(),
		Name:            status.Name,
		BudgetAmount:    float64(status.BudgetAmount) / 100,
		Spent:           float64(status.Spent) / 100,
		Remaining:       float64(status.Remaining) / 100,
		ProgressPercent: status.ProgressPercent,
		Status:          string(status.Status),
		PeriodStart:     status.PeriodStart.Format(time.RFC3339),
		PeriodEnd:       status.PeriodEnd.Format(time.RFC3339),
	}
}

// converts rupees from frontend to backend paisa
func rupeesToPaisa(amount float64) int64 {
	return int64(math.Round(amount * 100))
}
