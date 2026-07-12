package service

import (
	"backend-go/internal/ai/domain"
	categoryDomain "backend-go/internal/categories/domain"
	categoryService "backend-go/internal/categories/service"
	transactionDomain "backend-go/internal/transactions/domain"
	transactionService "backend-go/internal/transactions/service"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

// maxToolRounds bounds the model<->tool loop for one prompt.
const maxToolRounds = 5

// maxHistoryTurns caps how much client-supplied history is forwarded.
const maxHistoryTurns = 20

type Service struct {
	credits         domain.CreditRepository
	transactions    *transactionService.Service
	categories      *categoryService.Service
	categoryRepo    categoryDomain.CategoryRepository
	transactionRepo transactionDomain.TransactionRepository
	client          *OpenAIClient
}

func NewService(
	credits domain.CreditRepository,
	transactions *transactionService.Service,
	categories *categoryService.Service,
	categoryRepo categoryDomain.CategoryRepository,
	transactionRepo transactionDomain.TransactionRepository,
	client *OpenAIClient,
) *Service {
	return &Service{
		credits:         credits,
		transactions:    transactions,
		categories:      categories,
		categoryRepo:    categoryRepo,
		transactionRepo: transactionRepo,
		client:          client,
	}
}

type ChatTurn struct {
	Role    string
	Content string
}

type ChatInput struct {
	UserID  uuid.UUID
	Message string
	History []ChatTurn
}

type ChatOutput struct {
	Reply            string
	Actions          []string
	CreditsRemaining int
}

func (s *Service) GetCredits(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.credits.GetCredits(ctx, userID)
}

func (s *Service) Chat(ctx context.Context, input *ChatInput) (*ChatOutput, error) {
	if strings.TrimSpace(input.Message) == "" {
		return nil, domain.ErrInvalidInput
	}
	if !s.client.configured() {
		return nil, domain.ErrAINotConfigured
	}

	// one prompt = one credit, spent up front (atomic); refunded if the
	// provider is unreachable
	remaining, err := s.credits.ConsumeCredit(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	categories, err := s.categoryRepo.ListByUser(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	messages := []chatMessage{{Role: "system", Content: buildSystemPrompt(categories)}}
	history := input.History
	if len(history) > maxHistoryTurns {
		history = history[len(history)-maxHistoryTurns:]
	}
	for _, turn := range history {
		if turn.Role != "user" && turn.Role != "assistant" {
			continue
		}
		messages = append(messages, chatMessage{Role: turn.Role, Content: turn.Content})
	}
	messages = append(messages, chatMessage{Role: "user", Content: input.Message})

	actions := []string{}
	reply := ""

	for round := 0; round < maxToolRounds; round++ {
		msg, err := s.client.chat(ctx, messages, toolDefs())
		if err != nil {
			log.Printf("ai: provider call failed: %v", err)
			if refundErr := s.credits.RefundCredit(ctx, input.UserID); refundErr != nil {
				log.Printf("ai: credit refund failed: %v", refundErr)
			}
			return nil, domain.ErrAIUnavailable
		}

		if len(msg.ToolCalls) == 0 {
			reply = msg.Content
			break
		}

		messages = append(messages, *msg)
		for _, tc := range msg.ToolCalls {
			result, action := s.executeTool(ctx, input.UserID, tc)
			if action != "" {
				actions = append(actions, action)
			}
			messages = append(messages, chatMessage{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    result,
			})
		}
	}

	if reply == "" {
		reply = "I did what I could with that request - check the actions below."
	}

	return &ChatOutput{
		Reply:            reply,
		Actions:          actions,
		CreditsRemaining: remaining,
	}, nil
}

func buildSystemPrompt(categories []categoryDomain.Category) string {
	names := make([]string, 0, len(categories))
	for _, c := range categories {
		if !c.Hidden {
			names = append(names, c.Name)
		}
	}
	today := time.Now().UTC().Format("2006-01-02")
	return fmt.Sprintf(
		"You are FinAI, a personal finance assistant inside a money-tracking app. "+
			"All amounts are Indian Rupees (INR). Today is %s. "+
			"The user's categories are: %s. "+
			"You can record transactions, create categories and read spending summaries "+
			"using the provided tools. Use get_spending_summary before answering questions "+
			"about the user's spending. When recording a transaction, pick the best matching "+
			"existing category unless the user asks for a new one. Be concise and practical; "+
			"answer money-management questions with the user's actual data when available.",
		today, strings.Join(names, ", "),
	)
}

func toolDefs() []toolDef {
	return []toolDef{
		{
			Type: "function",
			Function: functionDef{
				Name:        "create_transaction",
				Description: "Record an income or expense transaction for the user.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"category":    map[string]any{"type": "string", "description": "Name of an existing category"},
						"amount":      map[string]any{"type": "number", "description": "Amount in rupees, > 0"},
						"kind":        map[string]any{"type": "string", "enum": []string{"Income", "Expense"}},
						"date":        map[string]any{"type": "string", "description": "YYYY-MM-DD, defaults to today; must not be in the future"},
						"description": map[string]any{"type": "string", "description": "Short optional note"},
					},
					"required": []string{"category", "amount", "kind"},
				},
			},
		},
		{
			Type: "function",
			Function: functionDef{
				Name:        "create_category",
				Description: "Create a new spending category for the user.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string"},
						"icon": map[string]any{"type": "string", "description": "A single emoji, optional"},
					},
					"required": []string{"name"},
				},
			},
		},
		{
			Type: "function",
			Function: functionDef{
				Name:        "get_spending_summary",
				Description: "Summarize the user's transactions: totals and per-category breakdown, optionally filtered by date range or category.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"date_from": map[string]any{"type": "string", "description": "YYYY-MM-DD inclusive, optional"},
						"date_to":   map[string]any{"type": "string", "description": "YYYY-MM-DD exclusive, optional"},
						"category":  map[string]any{"type": "string", "description": "Restrict to one category name, optional"},
					},
				},
			},
		},
	}
}

// executeTool runs one tool call. It returns the JSON result for the model
// and, for state-changing tools, a human-readable action confirmation for the
// UI. Tool failures are reported back to the model, never abort the chat.
func (s *Service) executeTool(ctx context.Context, userID uuid.UUID, tc toolCall) (string, string) {
	switch tc.Function.Name {
	case "create_transaction":
		return s.toolCreateTransaction(ctx, userID, tc.Function.Arguments)
	case "create_category":
		return s.toolCreateCategory(ctx, userID, tc.Function.Arguments)
	case "get_spending_summary":
		return s.toolSpendingSummary(ctx, userID, tc.Function.Arguments), ""
	default:
		return toolError("unknown tool"), ""
	}
}

func toolError(msg string) string {
	out, _ := json.Marshal(map[string]string{"error": msg})
	return string(out)
}

func toolOK(payload map[string]any) string {
	payload["ok"] = true
	out, _ := json.Marshal(payload)
	return string(out)
}

func (s *Service) resolveCategory(ctx context.Context, userID uuid.UUID, name string) (*categoryDomain.Category, error) {
	all, err := s.categoryRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range all {
		if strings.EqualFold(all[i].Name, name) && !all[i].Hidden {
			return &all[i], nil
		}
	}
	return nil, nil
}

func (s *Service) toolCreateTransaction(ctx context.Context, userID uuid.UUID, rawArgs string) (string, string) {
	var args struct {
		Category    string  `json:"category"`
		Amount      float64 `json:"amount"`
		Kind        string  `json:"kind"`
		Date        string  `json:"date"`
		Description string  `json:"description"`
	}
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return toolError("invalid arguments"), ""
	}
	if args.Amount <= 0 {
		return toolError("amount must be greater than zero"), ""
	}

	category, err := s.resolveCategory(ctx, userID, args.Category)
	if err != nil {
		return toolError("could not look up categories"), ""
	}
	if category == nil {
		return toolError(fmt.Sprintf("category %q not found; ask the user or create it first", args.Category)), ""
	}

	date := time.Now().UTC()
	if args.Date != "" {
		parsed, err := time.Parse("2006-01-02", args.Date)
		if err != nil {
			return toolError("date must be YYYY-MM-DD"), ""
		}
		date = parsed
	}

	var description *string
	if strings.TrimSpace(args.Description) != "" {
		d := strings.TrimSpace(args.Description)
		description = &d
	}

	err = s.transactions.CreateTransaction(ctx, transactionService.CreateInput{
		UserID:      userID,
		CategoryID:  category.ID,
		Amount:      int64(math.Round(args.Amount * 100)), // rupees -> paisa
		Description: description,
		Type:        args.Kind,
		Date:        date,
	})
	if err != nil {
		return toolError(fmt.Sprintf("could not create transaction: %v", err)), ""
	}

	label := category.Name
	if description != nil {
		label = *description
	}
	action := fmt.Sprintf("Added %s ₹%.2f - %s (%s)",
		strings.ToLower(args.Kind), args.Amount, label, category.Name)
	return toolOK(map[string]any{
		"category": category.Name,
		"amount":   args.Amount,
		"kind":     args.Kind,
		"date":     date.Format("2006-01-02"),
	}), action
}

func (s *Service) toolCreateCategory(ctx context.Context, userID uuid.UUID, rawArgs string) (string, string) {
	var args struct {
		Name string `json:"name"`
		Icon string `json:"icon"`
	}
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return toolError("invalid arguments"), ""
	}
	if strings.TrimSpace(args.Name) == "" {
		return toolError("name is required"), ""
	}

	if err := s.categories.Create(ctx, userID, strings.TrimSpace(args.Name), args.Icon); err != nil {
		return toolError(fmt.Sprintf("could not create category: %v", err)), ""
	}

	action := fmt.Sprintf("Created category %s", strings.TrimSpace(args.Name))
	if args.Icon != "" {
		action = fmt.Sprintf("Created category %s %s", args.Icon, strings.TrimSpace(args.Name))
	}
	return toolOK(map[string]any{"name": strings.TrimSpace(args.Name)}), action
}

func (s *Service) toolSpendingSummary(ctx context.Context, userID uuid.UUID, rawArgs string) string {
	var args struct {
		DateFrom string `json:"date_from"`
		DateTo   string `json:"date_to"`
		Category string `json:"category"`
	}
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return toolError("invalid arguments")
	}

	filter := transactionDomain.TransactionFilter{}
	if args.DateFrom != "" {
		parsed, err := time.Parse("2006-01-02", args.DateFrom)
		if err != nil {
			return toolError("date_from must be YYYY-MM-DD")
		}
		filter.DateFrom = &parsed
	}
	if args.DateTo != "" {
		parsed, err := time.Parse("2006-01-02", args.DateTo)
		if err != nil {
			return toolError("date_to must be YYYY-MM-DD")
		}
		filter.DateTo = &parsed
	}

	categories, err := s.categoryRepo.ListByUser(ctx, userID)
	if err != nil {
		return toolError("could not look up categories")
	}
	names := make(map[uuid.UUID]string, len(categories))
	for _, c := range categories {
		names[c.ID] = c.Name
	}
	if args.Category != "" {
		match, err := s.resolveCategory(ctx, userID, args.Category)
		if err != nil || match == nil {
			return toolError(fmt.Sprintf("category %q not found", args.Category))
		}
		filter.CategoryID = &match.ID
	}

	transactions, err := s.transactionRepo.List(ctx, userID, filter)
	if err != nil {
		return toolError("could not list transactions")
	}

	type catTotals struct {
		Income  float64 `json:"income"`
		Expense float64 `json:"expense"`
	}
	var totalIncome, totalExpense float64
	byCategory := map[string]*catTotals{}
	for _, tx := range transactions {
		amount := float64(tx.Amount) / 100 // paisa -> rupees
		name := names[tx.CategoryID]
		if byCategory[name] == nil {
			byCategory[name] = &catTotals{}
		}
		if tx.Type == transactionDomain.TransactionTypeIncome {
			totalIncome += amount
			byCategory[name].Income += amount
		} else {
			totalExpense += amount
			byCategory[name].Expense += amount
		}
	}

	out, _ := json.Marshal(map[string]any{
		"currency":          "INR",
		"transaction_count": len(transactions),
		"total_income":      totalIncome,
		"total_expense":     totalExpense,
		"net":               totalIncome - totalExpense,
		"by_category":       byCategory,
	})
	return string(out)
}
