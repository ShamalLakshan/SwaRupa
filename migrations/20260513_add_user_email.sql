-- Add email column to users table and unique index for non-null emails
-- Backup your database before running these statements.

BEGIN;

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS email text;

-- Create a partial unique index so multiple NULL emails are allowed
CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique_idx ON users (email) WHERE email IS NOT NULL;

COMMIT;
