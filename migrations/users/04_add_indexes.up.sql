-- Indexes for users table
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_username ON users (username);
CREATE INDEX IF NOT EXISTS idx_users_role ON users (role);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users (is_active);

-- Indexes for password_reset_tokens table
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_token ON password_reset_tokens (token);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id ON password_reset_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_expires_at ON password_reset_tokens (expires_at);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_used ON password_reset_tokens (used);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id_used ON password_reset_tokens (user_id, used) WHERE used = false;

COMMENT ON INDEX idx_users_email IS 'Index to optimize lookups by email in users table.';
COMMENT ON INDEX idx_password_reset_tokens_token IS 'Index to optimize lookups by token in password_reset_tokens table.';