ALTER TABLE users
    ADD COLUMN totp_secret text,
    ADD COLUMN totp_enabled boolean DEFAULT FALSE,
    ADD COLUMN totp_confirmed_at timestamp with time zone;
