-- Drop trigger
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop table
DROP TABLE IF EXISTS users;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop type
DROP TYPE IF EXISTS user_role;

