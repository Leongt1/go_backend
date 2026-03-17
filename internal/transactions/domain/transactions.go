package domain

import (
	"time"

	"github.com/google/uuid"
)

type TransactionType string

const (
	TransactionTypeIncome  TransactionType = "Income"
	TransactionTypeExpense TransactionType = "Expense"
)

func ParseTransactionType(s string) (TransactionType, error) {
	switch s {
	case "Income":
		return TransactionTypeIncome, nil
	case "Expense":
		return TransactionTypeExpense, nil
	default:
		return "", ErrInvalidInput
	}
}

type Transaction struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	CategoryID  uuid.UUID
	Amount      int64
	Description *string
	Type        TransactionType
	Date        time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedBy   *uuid.UUID
	UpdatedBy   *uuid.UUID
}

func NewTransaction(
	userID, categoryID uuid.UUID,
	description *string,
	amount int64,
	txType TransactionType,
	date time.Time,
) (*Transaction, error) {
	if amount <= 0 {
		return nil, ErrInvalidInput
	}

	if date.After(time.Now().UTC()) {
		return nil, ErrInvalidInput
	}

	if txType == "" {
		return nil, ErrInvalidInput
	}

	now := time.Now().UTC()
	return &Transaction{
		ID:          uuid.New(),
		UserID:      userID,
		CategoryID:  categoryID,
		Amount:      amount,
		Description: description,
		Type:        txType,
		Date:        date,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (t *Transaction) Update(
	categoryID *uuid.UUID,
	amount *int64,
	description *string,
	txType *TransactionType,
	date *time.Time,
	updatedBy *uuid.UUID,
) error {
	if categoryID != nil {
		t.CategoryID = *categoryID
	}

	if amount != nil {
		if *amount <= 0 {
			return ErrInvalidAmount
		}
		t.Amount = *amount
	}

	if description != nil {
		t.Description = description
	}

	if txType != nil {
		if *txType != TransactionTypeIncome && *txType != TransactionTypeExpense {
			return ErrInvalidInput
		}
		t.Type = *txType
	}

	if date != nil {
		if date.After(time.Now().UTC()) {
			return ErrFutureDate
		}
		t.Date = *date
	}

	t.UpdatedAt = time.Now().UTC()
	t.UpdatedBy = updatedBy
	return nil
}
