-- Migration 004: Add GitHub OAuth and Submission Queue Support
-- Date: 2026-05-23
-- Purpose: Replace email auth with GitHub OAuth, add approval workflow for artists/albums

-- ============================================================================
-- STEP 1: Update users table with GitHub OAuth fields
-- ============================================================================

-- Rename existing email to contact_email (preserve old data)
ALTER TABLE users RENAME COLUMN email TO contact_email;

-- Add GitHub OAuth fields
ALTER TABLE users ADD COLUMN github_id text UNIQUE;
ALTER TABLE users ADD COLUMN github_username text;
ALTER TABLE users ADD COLUMN github_profile_url text;
ALTER TABLE users ADD COLUMN oauth_provider text DEFAULT 'github';
ALTER TABLE users ADD COLUMN last_login timestamp;

-- Create unique indexes for GitHub and contact email (partial, allowing multiple NULLs)
CREATE UNIQUE INDEX idx_users_github_id ON users(github_id) WHERE github_id IS NOT NULL;
CREATE UNIQUE INDEX idx_users_contact_email ON users(contact_email) WHERE contact_email IS NOT NULL;

-- ============================================================================
-- STEP 2: Add approval workflow to artists table
-- ============================================================================

ALTER TABLE artists ADD COLUMN approval_status text DEFAULT 'pending';
ALTER TABLE artists ADD COLUMN approved_by text;
ALTER TABLE artists ADD COLUMN approved_at timestamp;
ALTER TABLE artists ADD COLUMN rejection_reason text;

-- Create indexes for filtering and queries
CREATE INDEX IF NOT EXISTS idx_artists_approval_status ON artists(approval_status);
CREATE INDEX IF NOT EXISTS idx_artists_submitted_by ON artists(submitted_by);

-- ============================================================================
-- STEP 3: Add approval workflow to albums table
-- ============================================================================

ALTER TABLE albums ADD COLUMN approval_status text DEFAULT 'pending';
ALTER TABLE albums ADD COLUMN approved_by text;
ALTER TABLE albums ADD COLUMN approved_at timestamp;
ALTER TABLE albums ADD COLUMN rejection_reason text;

-- Create indexes for filtering and queries
CREATE INDEX IF NOT EXISTS idx_albums_approval_status ON albums(approval_status);
CREATE INDEX IF NOT EXISTS idx_albums_submitted_by ON albums(submitted_by);

-- ============================================================================
-- STEP 4: Create submission queue table (optional but useful for admin dashboard)
-- ============================================================================

CREATE TABLE submission_queue (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  content_type text NOT NULL CHECK (content_type IN ('artist', 'album', 'artwork')),
  content_id UUID NOT NULL,
  submitted_by text NOT NULL,
  submitted_at timestamp DEFAULT now(),
  status text DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'withdrawn')),
  reviewed_by text,
  reviewed_at timestamp,
  review_notes text
);

CREATE INDEX IF NOT EXISTS idx_submission_queue_status ON submission_queue(status);
CREATE INDEX IF NOT EXISTS idx_submission_queue_content_type ON submission_queue(content_type);
CREATE INDEX IF NOT EXISTS idx_submission_queue_submitted_by ON submission_queue(submitted_by);
CREATE INDEX IF NOT EXISTS idx_submission_queue_created_at ON submission_queue(submitted_at DESC);

-- ============================================================================
-- STEP 5: Verify migration (run these after applying migration)
-- ============================================================================
-- SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'users' AND column_name IN ('github_id', 'github_username', 'github_profile_url');
-- SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'artists' AND column_name IN ('approval_status', 'approved_by', 'approved_at', 'rejection_reason');
-- SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'albums' AND column_name IN ('approval_status', 'approved_by', 'approved_at', 'rejection_reason');
