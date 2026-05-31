ALTER TABLE refresh_tokens
DROP COLUMN IF EXISTS user_agent,
DROP COLUMN IF EXISTS software_name,
DROP COLUMN IF EXISTS device_name;
