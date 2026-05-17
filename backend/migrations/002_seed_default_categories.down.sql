-- 002_seed_default_categories.down.sql

DELETE FROM categories
WHERE user_id IS NULL
  AND is_default = TRUE
  AND name IN (
      'Food',
      'Transport',
      'Housing',
      'Health',
      'Entertainment',
      'Education',
      'Shopping',
      'Travel',
      'Utilities',
      'Other'
  );