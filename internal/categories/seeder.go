package categories

import (
	"backend-go/internal/categories/domain"
	"context"
	"log"

	"github.com/google/uuid"
)

var defaultCategories = []struct {
	Name string
	Icon string
}{
	{Name: "Uncategorised", Icon: "📦"},
	{Name: "Food & Dining", Icon: "🍔"},
	{Name: "Rent & Housing", Icon: "🏠"},
	{Name: "Transport", Icon: "🚗"},
	{Name: "Entertainment", Icon: "🎬"},
	{Name: "Healthcare", Icon: "🏥"},
	{Name: "Salary / Income", Icon: "💰"},
	{Name: "Utilities", Icon: "💡"},
	{Name: "Shopping", Icon: "🛍️"},
	{Name: "Education", Icon: "📚"},
}

func SeedCategoriesForUser(ctx context.Context, userID uuid.UUID, repo domain.CategoryRepository) error {
	for _, c := range defaultCategories {
		category, err := domain.NewCategory(userID, c.Name, c.Icon)
		if err != nil {
			return err
		}
		if err := repo.Create(ctx, category); err != nil {
			return err
		}
	}

	log.Printf("Default categories seeded for user %s", userID)
	return nil
}
