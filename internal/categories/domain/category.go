package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// System default — seeded at startup, never modified by users
// type Category struct {
// 	ID        uuid.UUID
// 	Name      string
// 	Icon      string
// 	CreatedAt time.Time
// }

// Per-user state — overrides a system default OR a custom category
type Category struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	Name 	   string 	 `json:"name"` // always resolved name (custom or system)
	Icon       *string   `json:"icon"`
	Hidden     bool		 `json:"hidden"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// What handlers and the AI layer always work with — never the raw split
// type ResolvedCategory struct {
// 	ID     uuid.UUID `json:"id"`
// 	Name   string    `json:"name"`
// 	Icon   string    `json:"icon"`
// 	Hidden bool      `json:"hidden"`
// 	Custom bool      `json:"custom"`
// }

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
        Icon:      &icon,
        CreatedAt: now,
        UpdatedAt: now,
    }, nil
}

// NewSystemOverride creates a row that overrides a system default
// called lazily — only when the user first renames or hides a default
// func NewSystemOverride(userID, categoryID uuid.UUID) *UserCategory {
// 	now := time.Now().UTC()
// 	return &UserCategory{
// 		ID:         uuid.New(),
// 		UserID:     userID,
// 		CategoryID: &categoryID,
// 		Hidden:     false,
// 		CreatedAt:  now,
// 		UpdatedAt:  now,
// 	}
// }

// Rename sets a custom name on any user category
func (uc *Category) Rename(name string, icon *string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidInput
	}
	uc.Name = name
	if icon != nil {
		uc.Icon = icon
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

// Resolve merges a UserCategory with its system parent into a ResolvedCategory
// system can be nil if this is a custom category (no parent)
// func (uc *UserCategory) Resolve(system *Category) ResolvedCategory {
// 	resolved := ResolvedCategory{
// 		ID:     uc.ID,
// 		Hidden: uc.Hidden,
// 		Custom: uc.CategoryID == nil,
// 	}

// 	// custom_name takes priority, fall back to system name
// 	if uc.CustomName != nil && *uc.CustomName != "" {
// 		resolved.Name = *uc.CustomName
// 	} else if system != nil {
// 		resolved.Name = system.Name
// 	}

// 	// same priority for icon
// 	if uc.Icon != nil && *uc.Icon != "" {
// 		resolved.Icon = *uc.Icon
// 	} else if system != nil {
// 		resolved.Icon = system.Icon
// 	}

// 	return resolved
// }

// strPtr is a small helper — avoids &name pattern which doesn't work inline in Go
func strPtr(s string) *string {
	return &s
}
