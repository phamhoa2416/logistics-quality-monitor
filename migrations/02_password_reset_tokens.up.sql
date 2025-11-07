CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE password_reset_tokens
(
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID      NOT NULL,
    token      TEXT      NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    used       BOOLEAN          DEFAULT FALSE,
    created_at TIMESTAMP        DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
