DROP INDEX IF EXISTS idx_refresh_tokens_revoked;

ALTER TABLE refresh_tokens
DROP COLUMN IF EXISTS revoked_at,
DROP COLUMN IF EXISTS revoked,
DROP COLUMN IF EXISTS last_seen_at,
DROP COLUMN IF EXISTS ip_address,
DROP COLUMN IF EXISTS device;
