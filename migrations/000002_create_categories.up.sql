CREATE TABLE IF NOT EXISTS categories (
    id         UUID PRIMARY KEY,
    name       VARCHAR(100) NOT NULL UNIQUE,
    icon       VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS user_categories (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id UUID REFERENCES categories(id) ON DELETE CASCADE,
    custom_name VARCHAR(100),
    icon        VARCHAR(50),
    hidden      BOOLEAN NOT NULL DEFAULT false,
    deleted_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    UNIQUE(user_id, category_id)
);

CREATE INDEX idx_user_categories_user_id
    ON user_categories(user_id);

CREATE INDEX idx_user_categories_user_category
    ON user_categories(user_id, category_id);