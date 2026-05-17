-- 002_seed_default_categories.up.sql

INSERT INTO categories (user_id, name, icon, color, is_default)
SELECT NULL, 'Food', '🍔', '#FF6B6B', TRUE
WHERE NOT EXISTS (
	SELECT 1 FROM categories WHERE user_id IS NULL AND name = 'Food' AND is_default = TRUE
);

INSERT INTO categories (user_id, name, icon, color, is_default)
SELECT NULL, 'Transport', '🚗', '#4ECDC4', TRUE
WHERE NOT EXISTS (
	SELECT 1 FROM categories WHERE user_id IS NULL AND name = 'Transport' AND is_default = TRUE
);

INSERT INTO categories (user_id, name, icon, color, is_default)
SELECT NULL, 'Housing', '🏠', '#45B7D1', TRUE
WHERE NOT EXISTS (
	SELECT 1 FROM categories WHERE user_id IS NULL AND name = 'Housing' AND is_default = TRUE
);

INSERT INTO categories (user_id, name, icon, color, is_default)
SELECT NULL, 'Health', '💊', '#96CEB4', TRUE
WHERE NOT EXISTS (
	SELECT 1 FROM categories WHERE user_id IS NULL AND name = 'Health' AND is_default = TRUE
);

INSERT INTO categories (user_id, name, icon, color, is_default)
SELECT NULL, 'Entertainment', '🎬', '#FFEAA7', TRUE
WHERE NOT EXISTS (
	SELECT 1 FROM categories WHERE user_id IS NULL AND name = 'Entertainment' AND is_default = TRUE
);

INSERT INTO categories (user_id, name, icon, color, is_default)
SELECT NULL, 'Education', '📚', '#DDA15E', TRUE
WHERE NOT EXISTS (
	SELECT 1 FROM categories WHERE user_id IS NULL AND name = 'Education' AND is_default = TRUE
);

INSERT INTO categories (user_id, name, icon, color, is_default)
SELECT NULL, 'Shopping', '🛍️', '#BC6C25', TRUE
WHERE NOT EXISTS (
	SELECT 1 FROM categories WHERE user_id IS NULL AND name = 'Shopping' AND is_default = TRUE
);

INSERT INTO categories (user_id, name, icon, color, is_default)
SELECT NULL, 'Travel', '✈️', '#6A4C93', TRUE
WHERE NOT EXISTS (
	SELECT 1 FROM categories WHERE user_id IS NULL AND name = 'Travel' AND is_default = TRUE
);

INSERT INTO categories (user_id, name, icon, color, is_default)
SELECT NULL, 'Utilities', '⚡', '#FFB703', TRUE
WHERE NOT EXISTS (
	SELECT 1 FROM categories WHERE user_id IS NULL AND name = 'Utilities' AND is_default = TRUE
);

INSERT INTO categories (user_id, name, icon, color, is_default)
SELECT NULL, 'Other', '📌', '#999999', TRUE
WHERE NOT EXISTS (
	SELECT 1 FROM categories WHERE user_id IS NULL AND name = 'Other' AND is_default = TRUE
);

