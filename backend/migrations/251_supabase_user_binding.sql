ALTER TABLE users ADD COLUMN supabase_user_id UUID;
ALTER TABLE users ADD CONSTRAINT users_supabase_user_id_key UNIQUE (supabase_user_id);

-- Supabase owns credentials. Business users are created without a password.
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;
UPDATE users SET password_hash = NULL;
