DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_is_active;

DROP INDEX IF EXISTS idx_password_reset_tokens_token;
DROP INDEX IF EXISTS idx_password_reset_tokens_user_id;
DROP INDEX IF EXISTS idx_password_reset_tokens_expires_at;
DROP INDEX IF EXISTS idx_password_reset_tokens_used;
DROP INDEX IF EXISTS idx_password_reset_tokens_user_id_used;