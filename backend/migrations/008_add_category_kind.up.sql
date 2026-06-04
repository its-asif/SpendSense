-- 008_add_category_kind.up.sql

ALTER TABLE categories
    ADD COLUMN kind VARCHAR(20) NOT NULL DEFAULT 'EXPENSE';

ALTER TABLE categories
    ADD CONSTRAINT categories_kind_check CHECK (kind IN ('EXPENSE', 'INCOME'));

UPDATE categories SET kind = 'EXPENSE' WHERE kind IS NULL OR kind = '';

DROP INDEX IF EXISTS idx_categories_name_user_id;

CREATE UNIQUE INDEX idx_categories_default_kind_name
    ON categories (kind, name)
    WHERE user_id IS NULL AND is_default = TRUE;

CREATE UNIQUE INDEX idx_categories_user_kind_name
    ON categories (user_id, kind, name)
    WHERE user_id IS NOT NULL;

-- Default income categories (system-wide)
INSERT INTO categories (user_id, name, icon, color, is_default, kind)
SELECT NULL, 'Salary', '💼', '#10B981', TRUE, 'INCOME'
WHERE NOT EXISTS (
    SELECT 1 FROM categories WHERE user_id IS NULL AND name = 'Salary' AND kind = 'INCOME' AND is_default = TRUE
);

INSERT INTO categories (user_id, name, icon, color, is_default, kind)
SELECT NULL, 'Freelance', '💻', '#3B82F6', TRUE, 'INCOME'
WHERE NOT EXISTS (
    SELECT 1 FROM categories WHERE user_id IS NULL AND name = 'Freelance' AND kind = 'INCOME' AND is_default = TRUE
);

INSERT INTO categories (user_id, name, icon, color, is_default, kind)
SELECT NULL, 'Investment', '📈', '#8B5CF6', TRUE, 'INCOME'
WHERE NOT EXISTS (
    SELECT 1 FROM categories WHERE user_id IS NULL AND name = 'Investment' AND kind = 'INCOME' AND is_default = TRUE
);

INSERT INTO categories (user_id, name, icon, color, is_default, kind)
SELECT NULL, 'Gift', '🎁', '#F59E0B', TRUE, 'INCOME'
WHERE NOT EXISTS (
    SELECT 1 FROM categories WHERE user_id IS NULL AND name = 'Gift' AND kind = 'INCOME' AND is_default = TRUE
);

INSERT INTO categories (user_id, name, icon, color, is_default, kind)
SELECT NULL, 'Refund', '↩️', '#06B6D4', TRUE, 'INCOME'
WHERE NOT EXISTS (
    SELECT 1 FROM categories WHERE user_id IS NULL AND name = 'Refund' AND kind = 'INCOME' AND is_default = TRUE
);

INSERT INTO categories (user_id, name, icon, color, is_default, kind)
SELECT NULL, 'Other Income', '💰', '#64748B', TRUE, 'INCOME'
WHERE NOT EXISTS (
    SELECT 1 FROM categories WHERE user_id IS NULL AND name = 'Other Income' AND kind = 'INCOME' AND is_default = TRUE
);
