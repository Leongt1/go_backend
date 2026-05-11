package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type BudgetType string // "overall" | "category"
type BudgetKind string // "expense" | "savings"
type PeriodUnit string // "day" | "week" | "month" | "year"

const (
	BudgetTypeCategory BudgetType = "category"
	BudgetTypeOverall  BudgetType = "overall"

	BudgetKindExpense BudgetKind = "expense"
	BudgetKindSavings BudgetKind = "savings"

	PeriodUnitDay   PeriodUnit = "day"
	PeriodUnitWeek  PeriodUnit = "week"
	PeriodUnitMonth PeriodUnit = "month"
	PeriodUnitYear  PeriodUnit = "year"
)

type Budget struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Name        string
	Type        BudgetType
	Kind        BudgetKind
	Amount      int64
	PeriodUnit  PeriodUnit
	PeriodValue int
	StartDate   time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time

	// populated when fetched, not stored
	CategoryIDs []uuid.UUID
}

func NewBudget(
	userID uuid.UUID,
	name string,
	bType BudgetType, kind BudgetKind,
	amount int64,
	periodUnit PeriodUnit, periodValue int,
	startDate time.Time,
) (*Budget, error) {
	if strings.TrimSpace(name) == "" || bType == "" || kind == "" || periodUnit == "" {
		return nil, ErrInvalidInput
	}

	if amount <= 0 || periodValue <= 0 {
		return nil, ErrInvalidInput
	}

	return &Budget{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        name,
		Type:        bType,
		Kind:        kind,
		Amount:      amount,
		PeriodUnit:  periodUnit,
		PeriodValue: periodValue,
		StartDate:   startDate,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}, nil
}

func CurrentPeriod(startDate time.Time, unit PeriodUnit, value int) (from, to time.Time) {
	now := time.Now().UTC()
	start := startDate.UTC()

	n := 0
	periodStart := start
	for {
		next := addPeriod(periodStart, unit, value)
		if next.After(now) {
			break
		}
		periodStart = next
		n++
	}

	from = periodStart
	to = addPeriod(periodStart, unit, value)
	return
}

func addPeriod(t time.Time, unit PeriodUnit, value int) time.Time {
	switch unit {
	case PeriodUnitDay:
		return t.AddDate(0, 0, value)
	case PeriodUnitWeek:
		return t.AddDate(0, 0, value*7)
	case PeriodUnitMonth:
		return t.AddDate(0, value, 0)
	case PeriodUnitYear:
		return t.AddDate(value, 0, 0)
	default:
		return t
	}
}

func (b *Budget) Update(
	name *string,
	amount *int64,
	periodUnit *PeriodUnit,
	periodValue *int,
	startDate *time.Time,
	bType *BudgetType,
) error {
	if bType != nil {
		if *bType == "" {
			return ErrInvalidInput
		}
		b.Type = *bType
	}

	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return ErrInvalidInput
		}
		b.Name = trimmed
	}

	if amount != nil {
		if *amount <= 0 {
			return ErrInvalidInput
		}
		b.Amount = *amount
	}

	if periodUnit != nil {
		if *periodUnit == "" {
			return ErrInvalidInput
		}
		b.PeriodUnit = *periodUnit
	}

	if periodValue != nil {
		if *periodValue <= 0 {
			return ErrInvalidInput
		}
		b.PeriodValue = *periodValue
	}

	if startDate != nil {
		b.StartDate = startDate.UTC()
	}

	b.UpdatedAt = time.Now().UTC()

	return nil
}
