CREATE TABLE budgets (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    type        VARCHAR(20) NOT NULL CHECK (type IN ('overall', 'category')),
    kind        VARCHAR(20) NOT NULL CHECK (kind IN ('expense', 'savings')),
    amount      BIGINT NOT NULL CHECK (amount > 0),
    period_unit VARCHAR(10) NOT NULL CHECK (period_unit IN ('day', 'week', 'month', 'year')),
    period_value INT NOT NULL CHECK (period_value > 0),
    start_date  DATE NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL
);

CREATE TABLE budget_categories (
    id          UUID PRIMARY KEY,
    budget_id   UUID NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES user_categories(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL,
    UNIQUE (budget_id, category_id)
);

CREATE INDEX idx_budgets_user_id ON budgets(user_id);
CREATE INDEX idx_budget_categories_budget_id ON budget_categories(budget_id);