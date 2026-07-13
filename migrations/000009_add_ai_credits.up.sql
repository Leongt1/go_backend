-- AI assistant credit tokens (go_backend issue #4):
-- every user starts with 2 credits; one chat prompt consumes one credit.
ALTER TABLE users ADD COLUMN ai_credits INT NOT NULL DEFAULT 2;
