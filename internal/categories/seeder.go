package categories

import (
	"backend-go/internal/categories/domain"
	"context"
	"log"
	"time"

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

func SeedCategories(ctx context.Context, repo domain.CategorySeeder) error {
	existing, err := repo.List(ctx)
	if err != nil {
		return err
	}

	if len(existing) > 0 {
		log.Println("Categories already seeded, skipping")
		return nil
	}

	for _, c := range defaultCategories {
		category := &domain.Category{
			ID:        uuid.New(),
			Name:      c.Name,
			Icon:      c.Icon,
			CreatedAt: time.Now().UTC(),
		}

		if err := repo.CreateCategory(ctx, category); err != nil {
			return err
		}
	}

	log.Println("Default categories seeded successfully")
	return nil
}
