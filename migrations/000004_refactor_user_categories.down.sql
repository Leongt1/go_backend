-- Drop the new index
DROP INDEX IF EXISTS idx_user_categories_user_id_name;

-- Drop the correct foreign key
ALTER TABLE transactions DROP CONSTRAINT transactions_category_id_fkey;

-- Add category_id column back
ALTER TABLE user_categories ADD COLUMN category_id UUID REFERENCES categories(id) ON DELETE CASCADE;

-- Rename name back to custom_name
ALTER TABLE user_categories RENAME COLUMN name TO custom_name;

-- Recreate the unique constraint
ALTER TABLE user_categories ADD CONSTRAINT user_categories_user_id_category_id_key UNIQUE(user_id, category_id);

-- Recreate the old index
CREATE INDEX idx_user_categories_user_category
    ON user_categories(user_id, category_id);

-- Recreate the original foreign key
ALTER TABLE transactions
    ADD CONSTRAINT transactions_category_id_fkey
    FOREIGN KEY (category_id)
    REFERENCES user_categories(id);
