-- Hashes cannot be reversed to plaintext: rolling back invalidates all sessions.
DELETE FROM refresh_tokens;

DROP INDEX IF EXISTS idx_refresh_tokens_family_id;
ALTER TABLE refresh_tokens DROP COLUMN family_id;

ALTER INDEX idx_refresh_tokens_token_hash RENAME TO idx_refresh_tokens_token;
ALTER TABLE refresh_tokens RENAME COLUMN token_hash TO token;
