DROP TABLE IF EXISTS revoked_tokens;
ALTER TABLE users DROP COLUMN IF EXISTS token_version;
