CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login CITEXT NOT NULL UNIQUE,
    password_hash BYTEA NOT NULL,
    password_salt BYTEA NOT NULL, 
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE secrets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blind_index TEXT NOT NULL,
    data BYTEA NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    created_at TIMESTAMP NOT NULL DEFAULT now(),

    UNIQUE(user_id, blind_index)
);

CREATE INDEX idx_secrets_user_id_blind ON secrets(user_id, blind_index);