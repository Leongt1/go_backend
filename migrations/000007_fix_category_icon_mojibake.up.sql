-- Repair default category icons that were stored as '?' runs ('????').
-- The emoji were mangled during the original data import into Neon
-- (each UTF-8 byte was replaced by one '?'); the Go seeder itself
-- stores icons correctly, so only pre-existing rows are affected.
-- Guard: only rows whose icon is entirely question marks are touched,
-- so user-set custom icons are never overwritten.
UPDATE user_categories AS uc
SET icon = fix.icon,
    updated_at = now()
FROM (VALUES
    ('Uncategorised',   '📦'),
    ('Food & Dining',   '🍔'),
    ('Rent & Housing',  '🏠'),
    ('Transport',       '🚗'),
    ('Entertainment',   '🎬'),
    ('Healthcare',      '🏥'),
    ('Salary / Income', '💰'),
    ('Utilities',       '💡'),
    ('Shopping',        '🛍️'),
    ('Education',       '📚')
) AS fix(name, icon)
WHERE uc.name = fix.name
  AND uc.icon ~ '^[?]+$';
