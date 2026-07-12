-- Refresh-token hardening (go_backend issue #5):
-- 1. Store SHA-256 hashes instead of plaintext tokens. Existing rows are hashed
--    in place, so live sessions survive the migration.
-- 2. Add family_id: every login starts a token family; each rotation stays in
--    the family. Replaying a revoked token revokes the whole family.
UPDATE refresh_tokens SET token = encode(sha256(convert_to(token, 'UTF8')), 'hex');

ALTER TABLE refresh_tokens RENAME COLUMN token TO token_hash;

ALTER TABLE refresh_tokens ADD COLUMN family_id UUID;
UPDATE refresh_tokens SET family_id = id;
ALTER TABLE refresh_tokens ALTER COLUMN family_id SET NOT NULL;

ALTER INDEX idx_refresh_tokens_token RENAME TO idx_refresh_tokens_token_hash;
CREATE INDEX idx_refresh_tokens_family_id ON refresh_tokens(family_id);
