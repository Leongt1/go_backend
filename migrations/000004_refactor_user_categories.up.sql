-- Drop the unique constraint that references category_id
ALTER TABLE user_categories DROP CONSTRAINT user_categories_user_id_category_id_key;

-- Drop the unused index
DROP INDEX IF EXISTS idx_user_categories_user_category;

-- Drop the existing foreign key
ALTER TABLE transactions DROP CONSTRAINT transactions_category_id_fkey;

-- Rename custom_name to name
ALTER TABLE user_categories RENAME COLUMN custom_name TO name;

-- Drop category_id column
ALTER TABLE user_categories DROP COLUMN category_id;

-- Add the correct foreign key pointing to user_categories
ALTER TABLE transactions
    ADD CONSTRAINT transactions_category_id_fkey
    FOREIGN KEY (category_id)
    REFERENCES user_categories(id)
    ON DELETE SET NULL;

-- Create a new index for the updated structure
CREATE INDEX idx_user_categories_user_id_name
    ON user_categories(user_id, name)
    WHERE hidden = false;
