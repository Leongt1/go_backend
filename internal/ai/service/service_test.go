package service

import (
	aiDomain "backend-go/internal/ai/domain"
	categoryDomain "backend-go/internal/categories/domain"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// fakeCreditRepo records credit operations so tests can assert whether a prompt
// spent or refunded a credit.
type fakeCreditRepo struct {
	balance      int
	consumeCalls int
	refundCalls  int
}

func (f *fakeCreditRepo) GetCredits(context.Context, uuid.UUID) (int, error) {
	return f.balance, nil
}

func (f *fakeCreditRepo) ConsumeCredit(context.Context, uuid.UUID) (int, error) {
	f.consumeCalls++
	if f.balance <= 0 {
		return 0, aiDomain.ErrNoCredits
	}
	f.balance--
	return f.balance, nil
}

func (f *fakeCreditRepo) RefundCredit(context.Context, uuid.UUID) error {
	f.refundCalls++
	f.balance++
	return nil
}

// fakeCategoryRepo embeds the interface so only ListByUser (the one method the
// no-tool chat path uses) needs implementing; other methods panic if called.
type fakeCategoryRepo struct {
	categoryDomain.CategoryRepository
	cats []categoryDomain.Category
}

func (f *fakeCategoryRepo) ListByUser(context.Context, uuid.UUID) ([]categoryDomain.Category, error) {
	return f.cats, nil
}

// stubOpenAI returns a server that replies with a plain assistant message (no
// tool calls) on 2xx, or the given status when status != 200.
func stubOpenAI(t *testing.T, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if status != http.StatusOK {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":{"message":"boom"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"Done."}}]}`))
	}))
}

// stubOpenAIContent returns a server whose assistant message content is the
// given string - used to feed the extraction path a JSON payload to parse.
func stubOpenAIContent(t *testing.T, content string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": content}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func newTestService(credits *fakeCreditRepo, baseURL string) *Service {
	// transaction/category services and the transaction repo are only touched
	// when a tool call fires; the no-tool reply path leaves them untouched, so
	// nil is safe for these tests.
	client := NewOpenAIClient("test-key", "test-model", baseURL)
	return NewService(credits, nil, nil, &fakeCategoryRepo{}, nil, client)
}

func TestChat_AdminBypassDoesNotSpendCredit(t *testing.T) {
	server := stubOpenAI(t, http.StatusOK)
	defer server.Close()

	credits := &fakeCreditRepo{balance: 5}
	svc := newTestService(credits, server.URL)

	out, err := svc.Chat(context.Background(), &ChatInput{
		UserID:  uuid.New(),
		Message: "add a 250 lunch expense",
		IsAdmin: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.CreditsRemaining != CreditsUnlimited {
		t.Errorf("admin CreditsRemaining = %d, want %d (unlimited)", out.CreditsRemaining, CreditsUnlimited)
	}
	if credits.consumeCalls != 0 {
		t.Errorf("admin consumeCalls = %d, want 0", credits.consumeCalls)
	}
	if credits.balance != 5 {
		t.Errorf("admin balance = %d, want unchanged 5", credits.balance)
	}
}

func TestChat_UserSpendsOneCredit(t *testing.T) {
	server := stubOpenAI(t, http.StatusOK)
	defer server.Close()

	credits := &fakeCreditRepo{balance: 5}
	svc := newTestService(credits, server.URL)

	out, err := svc.Chat(context.Background(), &ChatInput{
		UserID:  uuid.New(),
		Message: "hello",
		IsAdmin: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if credits.consumeCalls != 1 {
		t.Errorf("consumeCalls = %d, want 1", credits.consumeCalls)
	}
	if out.CreditsRemaining != 4 {
		t.Errorf("CreditsRemaining = %d, want 4", out.CreditsRemaining)
	}
}

func TestChat_ProviderFailureRefundsUser(t *testing.T) {
	server := stubOpenAI(t, http.StatusInternalServerError)
	defer server.Close()

	credits := &fakeCreditRepo{balance: 5}
	svc := newTestService(credits, server.URL)

	_, err := svc.Chat(context.Background(), &ChatInput{
		UserID:  uuid.New(),
		Message: "hello",
		IsAdmin: false,
	})
	if !errors.Is(err, aiDomain.ErrAIUnavailable) {
		t.Fatalf("err = %v, want ErrAIUnavailable", err)
	}
	if credits.consumeCalls != 1 {
		t.Errorf("consumeCalls = %d, want 1", credits.consumeCalls)
	}
	if credits.refundCalls != 1 {
		t.Errorf("refundCalls = %d, want 1 (credit returned on outage)", credits.refundCalls)
	}
}

func TestChat_ProviderFailureAdminNoRefund(t *testing.T) {
	server := stubOpenAI(t, http.StatusInternalServerError)
	defer server.Close()

	credits := &fakeCreditRepo{balance: 5}
	svc := newTestService(credits, server.URL)

	_, err := svc.Chat(context.Background(), &ChatInput{
		UserID:  uuid.New(),
		Message: "hello",
		IsAdmin: true,
	})
	if !errors.Is(err, aiDomain.ErrAIUnavailable) {
		t.Fatalf("err = %v, want ErrAIUnavailable", err)
	}
	if credits.consumeCalls != 0 || credits.refundCalls != 0 {
		t.Errorf("admin consume=%d refund=%d, want 0/0", credits.consumeCalls, credits.refundCalls)
	}
}

func TestExtract_ParsesAndSpendsOneCredit(t *testing.T) {
	server := stubOpenAIContent(t, `{"transactions":[
		{"amount":250,"kind":"Expense","category":"Food","date":"2026-08-01","description":"lunch"},
		{"amount":40000,"kind":"Income","category":"Salary","date":"2026-08-02","description":"july pay"}
	]}`)
	defer server.Close()

	credits := &fakeCreditRepo{balance: 5}
	svc := newTestService(credits, server.URL)

	out, err := svc.ExtractTransactions(context.Background(), &ExtractInput{
		UserID: uuid.New(),
		Text:   "spent 250 on lunch; salary 40000 credited",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Transactions) != 2 {
		t.Fatalf("got %d transactions, want 2", len(out.Transactions))
	}
	if out.Transactions[0].Amount != 250 || out.Transactions[0].Kind != "Expense" {
		t.Errorf("row 0 = %+v, want 250/Expense", out.Transactions[0])
	}
	if out.Transactions[1].Kind != "Income" {
		t.Errorf("row 1 kind = %q, want Income", out.Transactions[1].Kind)
	}
	if credits.consumeCalls != 1 {
		t.Errorf("consumeCalls = %d, want 1", credits.consumeCalls)
	}
	if out.CreditsRemaining != 4 {
		t.Errorf("CreditsRemaining = %d, want 4", out.CreditsRemaining)
	}
}

func TestExtract_DropsInvalidAmountsAndDefaultsKind(t *testing.T) {
	server := stubOpenAIContent(t, `{"transactions":[
		{"amount":0,"kind":"Expense","category":"Food"},
		{"amount":-5,"kind":"Expense","category":"Food"},
		{"amount":99,"kind":"gibberish","category":"Misc"}
	]}`)
	defer server.Close()

	credits := &fakeCreditRepo{balance: 5}
	svc := newTestService(credits, server.URL)

	out, err := svc.ExtractTransactions(context.Background(), &ExtractInput{
		UserID: uuid.New(),
		Text:   "some text",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Transactions) != 1 {
		t.Fatalf("got %d transactions, want 1 (zero/negative dropped)", len(out.Transactions))
	}
	if out.Transactions[0].Kind != "Expense" {
		t.Errorf("unknown kind = %q, want defaulted to Expense", out.Transactions[0].Kind)
	}
}

func TestExtract_AdminBypassDoesNotSpendCredit(t *testing.T) {
	server := stubOpenAIContent(t, `{"transactions":[{"amount":10,"kind":"Expense","category":"Food"}]}`)
	defer server.Close()

	credits := &fakeCreditRepo{balance: 5}
	svc := newTestService(credits, server.URL)

	out, err := svc.ExtractTransactions(context.Background(), &ExtractInput{
		UserID:  uuid.New(),
		Text:    "coffee 10",
		IsAdmin: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.CreditsRemaining != CreditsUnlimited {
		t.Errorf("admin CreditsRemaining = %d, want %d", out.CreditsRemaining, CreditsUnlimited)
	}
	if credits.consumeCalls != 0 {
		t.Errorf("admin consumeCalls = %d, want 0", credits.consumeCalls)
	}
}

func TestExtract_ProviderFailureRefundsUser(t *testing.T) {
	server := stubOpenAI(t, http.StatusInternalServerError)
	defer server.Close()

	credits := &fakeCreditRepo{balance: 5}
	svc := newTestService(credits, server.URL)

	_, err := svc.ExtractTransactions(context.Background(), &ExtractInput{
		UserID: uuid.New(),
		Text:   "spent 250 on lunch",
	})
	if !errors.Is(err, aiDomain.ErrAIUnavailable) {
		t.Fatalf("err = %v, want ErrAIUnavailable", err)
	}
	if credits.consumeCalls != 1 || credits.refundCalls != 1 {
		t.Errorf("consume=%d refund=%d, want 1/1", credits.consumeCalls, credits.refundCalls)
	}
}

func TestExtract_EmptyTextIsRejected(t *testing.T) {
	server := stubOpenAIContent(t, `{"transactions":[]}`)
	defer server.Close()

	credits := &fakeCreditRepo{balance: 5}
	svc := newTestService(credits, server.URL)

	_, err := svc.ExtractTransactions(context.Background(), &ExtractInput{
		UserID: uuid.New(),
		Text:   "   ",
	})
	if !errors.Is(err, aiDomain.ErrInvalidInput) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
	if credits.consumeCalls != 0 {
		t.Errorf("consumeCalls = %d, want 0 (rejected before spend)", credits.consumeCalls)
	}
}
