-- AI assistant credit tokens (go_backend issue #4):
-- launch grant: every user starts with 100 credits; one chat prompt consumes
-- one credit. Admins bypass consumption entirely (unlimited) in the service.
-- ADD COLUMN with a DEFAULT backfills existing rows to 100.
ALTER TABLE users ADD COLUMN ai_credits INT NOT NULL DEFAULT 100;
