DELETE FROM categories WHERE user_id IS NULL AND kind = 'INCOME' AND is_default = TRUE;

DROP INDEX IF EXISTS idx_categories_user_kind_name;
DROP INDEX IF EXISTS idx_categories_default_kind_name;

CREATE UNIQUE INDEX idx_categories_name_user_id ON categories(user_id, name);

ALTER TABLE categories DROP CONSTRAINT IF EXISTS categories_kind_check;
ALTER TABLE categories DROP COLUMN IF EXISTS kind;
