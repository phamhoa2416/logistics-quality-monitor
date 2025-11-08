CREATE TYPE user_role AS ENUM (
    'customer',
    'provider',
    'shipper',
    'admin'
    );

CREATE TABLE users
(
    id              UUID PRIMARY KEY                  DEFAULT gen_random_uuid(),
    username        VARCHAR(100) UNIQUE      NOT NULL,
    email           VARCHAR(255) UNIQUE      NOT NULL,
    password_hashed VARCHAR(255)             NOT NULL,
    full_name       VARCHAR(255)             NOT NULL,
    phone_number    VARCHAR(20) UNIQUE,
    role            user_role                NOT NULL DEFAULT 'customer',

    address         TEXT,

    is_active       BOOLEAN                  NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE OR REPLACE FUNCTION update_updated_at_column()
    RETURNS TRIGGER AS
$$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE
    ON users
    FOR EACH ROW
EXECUTE PROCEDURE update_updated_at_column();

COMMENT ON TABLE users IS 'Table storing user information with 4 roles: customer, provider, shipper, admin.';