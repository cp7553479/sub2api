-- pg_trgm is installed by the database owner in the configured extension schema.
CREATE INDEX IF NOT EXISTS idx_users_email_trgm ON users USING gin (email gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_users_username_trgm ON users USING gin (username gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_users_notes_trgm ON users USING gin (notes gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_accounts_name_trgm ON accounts USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_trgm ON api_keys USING gin ("key" gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_api_keys_name_trgm ON api_keys USING gin (name gin_trgm_ops);
