# User Auth And API Key Plan

This document turns the current Supabase Auth setup into the flow you described:

- users sign up and sign in with Supabase email/password
- the API stores a local profile row for each authenticated Supabase user
- each user gets exactly one API key that can be reset
- the API key is used to identify the user when a request is not using a Supabase JWT
- local roles remain the source of authorization for admin/contributor/moderator checks

## Target flow

1. A user is created in Supabase Auth with email/password.
2. The first authenticated API request upserts a matching row in the local `users` table.
3. The app stores the Supabase user id as the stable identity reference.
4. The app issues one API key for the user and only stores a hash of that key.
5. Requests can authenticate with either:
   - `Authorization: Bearer <supabase_jwt>`
   - `X-API-Key: <user_api_key>`
6. The local users table decides role-based authorization.
7. Resetting the API key invalidates the old one and returns a new secret once.

## Recommended backend changes

### 1. Keep Supabase user id as the primary identity

Use the Supabase Auth `sub` claim as the local `users.id` value. Do not accept client-supplied ids for authenticated creation.

### 2. Add one API key per user

Use one of these two designs:

- simplest: add `api_key_hash`, `api_key_prefix`, `api_key_created_at`, `api_key_rotated_at` columns to `users`
- richer history: create a `user_api_keys` table with one active key per user and an audit trail

For your requirement, the first option is enough and easier to keep reliable.

Store only the hash, never the plaintext key.

### 3. Add API-key lifecycle endpoints

Add endpoints like:

- `GET /api/users/me` to return the authenticated user profile
- `GET /api/users/:id` for admin or owner lookup
- `PATCH /api/users/:id` for allowed profile updates
- `PATCH /api/users/:id/role` for admin-only role changes
- `POST /api/users/:id/api-key/reset` for admin or owner reset
- `GET /api/users/:id/api-key` only if you want a one-time create flow; otherwise fold this into reset

On create/reset, return the plaintext key only once in the response.

### 4. Support API-key auth in middleware

Update the auth middleware to accept either JWT or API key.

- If a Supabase JWT is present, validate it and resolve the user by `sub`.
- If an API key is present, hash it, compare it to the stored hash, and load the matching user row.
- Put the resolved user id and role into request context for downstream handlers.

### 5. Make user creation server-driven

For the Supabase email/password flow, user creation should happen in one of two places:

- automatically on the first authenticated request, by upserting the local row from the JWT claims
- or explicitly from an admin-only provisioning endpoint that creates the Supabase Auth user and the local row together

Do not rely on the public request body to supply the user id for authenticated users.

### 6. Add role checks around CRUD

Suggested policy:

- user can read their own profile
- admin can list all users
- admin can create or delete users in the local table
- admin can change roles
- owner or admin can reset an API key

## Database changes

Add a migration for the local `users` table:

- `role text not null default 'contributor'`
- `api_key_hash text null`
- `api_key_prefix text null`
- `api_key_created_at timestamptz null`
- `api_key_rotated_at timestamptz null`
- keep `email` as nullable and unique only when present

If you want stricter auditing later, add an append-only `user_api_key_events` table.

## Supabase setup instructions

Use the current Supabase dashboard navigation as follows.

### A. Enable email/password authentication

1. Open your Supabase project dashboard.
2. Go to **Authentication**.
3. Open **Providers**.
4. Select **Email**.
5. Enable email signups and password sign-ins.
6. If you want manual user provisioning, decide whether email confirmation is required.

### B. Configure auth URLs

1. Go to **Authentication**.
2. Open **URL Configuration**.
3. Set your local and production site URLs.
4. Add redirect URLs for any frontend auth callback routes you will use.

### C. Create or manage users

1. Go to **Authentication**.
2. Open **Users**.
3. Use **Add user** or **Create user** to seed a user with email/password.
4. If you create a user manually, capture the Supabase user id and the initial credentials.
5. Use that user id only as the auth identity reference in your local database.

### D. Get the keys needed by the API server

1. Go to **Project Settings**.
2. Open **API**.
3. Copy the project URL.
4. Copy the `anon` key only for client-side usage.
5. Copy the `service_role` key only for trusted server-side admin operations.

### E. Confirm JWT verification settings

1. Keep `SUPABASE_URL` or `SUPABASE_JWKS_URL` available in the API server environment.
2. Prefer JWKS-based RS256 validation for production.
3. Use `SUPABASE_JWT_SECRET` only if you intentionally want HS256 validation.

## Backend implementation order

1. Add the user API key fields and migration.
2. Add the API-key generation and hashing helpers.
3. Add middleware support for `X-API-Key` alongside JWT.
4. Add owner/admin user CRUD and role endpoints.
5. Add reset-key endpoint that rotates the single active key.
6. Lock down any user-listing route that should not be public.
7. Update tests and example requests.

## Validation checklist

Run these checks after implementation:

1. `go build ./...`
2. authenticate with a real Supabase email/password user
3. confirm the first authenticated request creates the local user row
4. confirm the user can be identified by JWT and by API key
5. reset the API key and verify the old key no longer works
6. verify admin-only role changes are rejected for non-admin users

## Notes on the current codebase

- The current middleware already validates Supabase JWTs and upserts a local user row.
- The current user endpoints still accept client-provided ids for creation, which should be tightened for the Supabase flow.
- The current codebase does not yet have an API-key storage or verification path, so that is the next structural change.
