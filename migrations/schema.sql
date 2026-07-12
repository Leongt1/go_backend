CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY,
    name            TEXT NOT NULL,
    email           TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    role            TEXT NOT NULL,
    gender          TEXT NOT NULL,
    date_of_birth   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL,
    created_by      UUID,
    updated_by      UUID
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    family_id   UUID NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS user_categories (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    icon        TEXT NOT NULL DEFAULT '',
    hidden      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS transactions (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES user_categories(id),
    amount      BIGINT NOT NULL,
    description TEXT,
    type        TEXT NOT NULL,
    date        TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    created_by  UUID,
    updated_by  UUID
);

CREATE TABLE IF NOT EXISTS budgets (
    id           UUID PRIMARY KEY,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    type         TEXT NOT NULL,
    kind         TEXT NOT NULL,
    amount       BIGINT NOT NULL,
    period_unit  TEXT NOT NULL,
    period_value INT NOT NULL,
    start_date   TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS budget_categories (
    id          UUID PRIMARY KEY,
    budget_id   UUID NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES user_categories(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL,
    UNIQUE(budget_id, category_id)
);