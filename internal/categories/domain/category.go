package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Per-user state — overrides a system default OR a custom category
type Category struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	Name 	   string 	 `json:"name"` // always resolved name (custom or system)
	Icon       string   `json:"icon"`
	Hidden     bool		 `json:"hidden"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// NewUserCategory creates a brand new custom category for a user
func NewCategory(userID uuid.UUID, name, icon string) (*Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidInput
	}

	now := time.Now().UTC()
    return &Category{
        ID:        uuid.New(),
        UserID:    userID,
        Name:      name,
        Icon:      icon,
        CreatedAt: now,
        UpdatedAt: now,
    }, nil
}

// Rename sets a custom name on any user category
func (uc *Category) Rename(name string, icon *string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidInput
	}
	uc.Name = name
	if icon != nil {
		uc.Icon = *icon
	}
	uc.UpdatedAt = time.Now().UTC()
	return nil
}

// Hide soft-deletes — sets hidden flag and records when it happened
func (uc *Category) Hide() {
	now := time.Now().UTC()
	uc.Hidden = true
	uc.UpdatedAt = now
}

// Unhide restores a hidden category
func (uc *Category) Unhide() {
	uc.Hidden = false
	uc.UpdatedAt = time.Now().UTC()
}
