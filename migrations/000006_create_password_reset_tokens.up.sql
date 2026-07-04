CREATE TABLE password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL,

    token_hash TEXT NOT NULL,

    expires_at TIMESTAMPTZ NOT NULL,

    used_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_password_reset_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_password_reset_token_hash
ON password_reset_tokens(token_hash);

CREATE INDEX idx_password_reset_user
ON password_reset_tokens(user_id);