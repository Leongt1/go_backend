CREATE TABLE IF NOT EXISTS transactions (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES user_categories(id),
    amount      BIGINT NOT NULL,
    description VARCHAR(255),
    type        VARCHAR(10) NOT NULL,
    date        TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    created_by  UUID REFERENCES users(id),
    updated_by  UUID REFERENCES users(id)
);

CREATE INDEX idx_transactions_user_id ON transactions(user_id);
CREATE INDEX idx_transactions_date ON transactions(user_id, date DESC);
CREATE INDEX idx_transactions_category ON transactions(category_id);