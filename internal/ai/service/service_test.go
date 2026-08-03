package service

import (
	aiDomain "backend-go/internal/ai/domain"
	categoryDomain "backend-go/internal/categories/domain"
	"context"
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
