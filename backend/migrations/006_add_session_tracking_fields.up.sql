ALTER TABLE refresh_tokens
ADD COLUMN IF NOT EXISTS device VARCHAR(255),
ADD COLUMN IF NOT EXISTS ip_address VARCHAR(64),
ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMP,
ADD COLUMN IF NOT EXISTS revoked BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMP;

UPDATE refresh_tokens
SET last_seen_at = COALESCE(last_seen_at, created_at)
WHERE last_seen_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_revoked ON refresh_tokens(revoked);
