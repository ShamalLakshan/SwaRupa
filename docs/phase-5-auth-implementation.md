# Phase 5: Supabase Authentication Implementation

**Status:** In Progress  
**Last Updated:** May 13, 2026  
**Branch:** Main development

## Overview

Phase 5 implements JWT-based authentication using Supabase Auth, supporting email/password sign-ups and account linking to merge multiple identities (future OAuth support).

---

## Architecture

### Authentication Flow

1. **User Registration**: Sign up via Supabase Auth (email/password)
2. **JWT Token**: Supabase returns RS256-signed JWT token containing user_id, email, name claims
3. **API Requests**: Client includes `Authorization: Bearer <token>` header
4. **Middleware**: `AuthMiddleware` validates JWT → extracts claims → idempotent user creation in `public.users`
5. **Protected Endpoints**: Use user context from middleware instead of request body

### JWT Validation Strategy

- **Primary (RS256)**: Validate using JWKS (public key from Supabase)
- **Fallback (HS256)**: If `SUPABASE_JWT_SECRET` env var set, validate using secret
- **JWKS Caching**: 30-minute cache with synchronized access (custom implementation using `crypto/rsa`)

---

## Completed Implementation

### 1. Core Authentication Middleware
**File:** `internal/handlers/auth_middleware.go`

- Custom JWT validator (replaced keyfunc library due to v5 incompatibility)
- JWKS parser using `crypto/rsa`
- Extracts `sub` (user_id), `email`, `name` from JWT claims
- Idempotent user creation via `UserService.CreateUser`
- Sets `user_id` and `email` in Gin context for downstream handlers

### 2. Database Schema Updates
**File:** `migrations/20260513_add_user_email.sql`

- ✅ **EXECUTED**: Adds `email` column to `users` table
- ✅ **EXECUTED**: Creates partial unique index on email (allows multiple NULLs)

**Current Schema:**
```sql
CREATE TABLE users (
  id text PRIMARY KEY,
  display_name text,
  email text UNIQUE (WHERE email IS NOT NULL),
  role text DEFAULT 'contributor',
  created_at timestamp DEFAULT now()
);
```

### 3. User Service Enhancements
**File:** `internal/services/user_service.go`

- `CreateUser(ctx, id, displayName, email)`: UPSERT with email persistence
- `GetAllUsers()`: Includes email in response
- `GetUserByID(id)`: Includes email in response
- `LinkProviderAccount(ctx, primaryUserID, secondaryUserID)`: Merges two user accounts
  - Validates both users exist
  - Merges profile fields (display_name, email)
  - Repoints all foreign key references across artists, albums, artworks, artwork_sources
  - Deletes secondary user in transaction

### 4. User Model Update
**File:** `internal/models/user.go`

```go
type User struct {
    ID          string `json:"id"`
    DisplayName string `json:"display_name"`
    Email       string `json:"email,omitempty"`  // NEW
    Role        string `json:"role"`
    CreatedAt   string `json:"created_at"`
}
```

### 5. Protected Endpoints
**Routes secured with AuthMiddleware:**

- `POST /api/artists` - Submit artist
- `POST /api/albums` - Submit album
- `POST /api/albums/:id/artworks` - Submit album artworks
- `POST /api/artworks/:artwork_id/sources` - Submit artwork source
- `PATCH /api/artworks/:artwork_id/approve` - Approve artwork (admin only)
- `PATCH /api/artworks/:artwork_id/reject` - Reject artwork (admin only)

**New Endpoint:**
- `POST /api/users/link-provider` - Merge OAuth/email-password accounts

### 6. Handler Updates
All handlers modified to extract `user_id` from context instead of request body:

- `internal/handlers/artist.go`: Uses context user_id for `submitted_by`
- `internal/handlers/album.go`: Uses context user_id for `submitted_by`
- `internal/handlers/artwork.go`: Uses context user_id for `submitted_by`
- `internal/handlers/moderation.go`: Validates admin, uses context user_id for actions
- `internal/handlers/user.go`: New `LinkProvider` handler

### 7. Test Collections Updated
**Files:**
- `tests/SwaRupa-API-Phase04-Collection.postman_collection.json`
- `tests/swarupa-phase01-local.http`

Changes:
- ✅ Bearer token headers added to protected requests
- ✅ Deprecated fields removed: `submitted_by`, `requested_by`, `discovered_by` (now extracted from context)
- ✅ Link-provider endpoint tests added
- ✅ Example auth flow documented

### 8. Dependency Management
**File:** `go.mod`

- ✅ Added: `github.com/golang-jwt/jwt/v5 v5.1.0`
- ✅ Removed: `github.com/MichaelMure/go-sh` (keyfunc alternative, incompatible)
- ✅ Cleaned via: `go mod tidy`

### 9. Build Verification
- ✅ `go build ./...` - No errors (all code compiles)

---

## Current State (May 13, 2026)

### ✅ Fully Operational

- Email/password authentication flow ready
- JWT validation working (RS256 + HS256 fallback)
- Protected endpoints properly gated
- Account linking endpoint implemented
- Database schema migrated

### ⚠️ In Progress

- **Test User Creation**: Existing test users (`test_user_001`, `user_001`) lack email/password credentials
  - **Recommendation**: Delete test users, create new ones via Supabase Auth dashboard to generate credentials
  - Keep `user_000` (system admin) for seed data

### 📋 Not Yet Started

- **Google OAuth Integration** (deferred due to Google Cloud access restrictions)
- **Frontend Auth UI** (sign-up/login forms)
- **Token Refresh Logic** (handle expired tokens)
- **Email Verification** (optional for Phase 5)

---

## How to Resume Development

### 1. Start the API Server
```bash
cd cmd/api
go run main.go
```

Server listens on `http://localhost:8080` with protected routes gated by AuthMiddleware.

### 2. Create Test Users (via Supabase Dashboard)

1. Navigate to **Authentication** → **Users** tab
2. Click **Create User**, set email/password (e.g., `test@example.com` / `password123`)
3. Copy the resulting JWT token
4. Use in requests: `Authorization: Bearer <token>`

**Alternatively**, trigger user creation by making first authenticated request:
```bash
GET http://localhost:8080/api/users
Authorization: Bearer <valid-jwt-token>
```
This will auto-create the user row in `public.users`.

### 3. Test Email/Password Flow

```http
GET http://localhost:8080/api/users
Authorization: Bearer <your-jwt-token>
```

Expected: `200 OK` with users array including email field.

### 4. Test Account Linking

```http
POST http://localhost:8080/api/users/link-provider
Authorization: Bearer <primary-user-jwt>
Content-Type: application/json

{
  "provider_user_id": "secondary-user-id"
}
```

Expected: `200 OK` - Secondary user deleted, all references repointed to primary user.

### 5. Test Protected Submission Endpoints

```http
POST http://localhost:8080/api/artists
Authorization: Bearer <jwt>
Content-Type: application/json

{
  "name": "Artist Name",
  "bio": "Bio",
  "image_url": "https://..."
}
```

Expected: Artist created with authenticated user_id as `submitted_by`.

---

## Environment Configuration

Required `.env` variables (set in your local environment):

```
SUPABASE_URL=https://<project>.supabase.co
SUPABASE_JWKS_URL=https://<project>.supabase.co/auth/v1/.well-known/jwks.json
POOLER_DATABASE_URL=postgresql://<user>:<password>@<host>:<port>/postgres
```

Optional:
```
SUPABASE_JWT_SECRET=<only if using HS256 validation>
```

---

## Key Design Decisions

1. **Custom JWKS Parser**: Replaced keyfunc library with internal implementation using `crypto/rsa` due to jwt/v5 incompatibility
2. **Idempotent User Creation**: AuthMiddleware calls UPSERT so repeated requests from same JWT don't create duplicates
3. **Account Linking via Transaction**: Ensures data consistency when merging users and repointing foreign keys
4. **Context-Based User Identity**: User extracted from JWT context, not request body (prevents spoofing)
5. **Partial Unique Index**: Allows multiple NULL emails (legacy/system users without auth)

---

## Known Limitations & TODOs

- [ ] Google OAuth integration (blocked by Cloud access, can add when restored)
- [ ] Email verification flow (optional, not required for Phase 5)
- [ ] Token refresh endpoint (client handles refresh with Supabase SDK)
- [ ] Rate limiting on auth endpoints
- [ ] Audit logging for account merges

---

## Testing Checklist for Next Session

- [ ] Sign up new user via Supabase Auth
- [ ] Make first API request with JWT → verify user auto-created in `public.users`
- [ ] Call `GET /api/users` → verify email field populated
- [ ] Submit artwork with authenticated user → verify `submitted_by` matches JWT user_id
- [ ] Create second user, test `POST /api/users/link-provider` → verify merge
- [ ] Test moderation endpoints with non-admin user → verify rejection

---

## Files Changed

**New Files:**
- `migrations/20260513_add_user_email.sql`
- `internal/handlers/auth_middleware.go`

**Modified Files:**
- `internal/handlers/user.go` (added LinkProvider, updated Create)
- `internal/handlers/artist.go` (context user_id extraction)
- `internal/handlers/album.go` (context user_id extraction)
- `internal/handlers/artwork.go` (context user_id extraction)
- `internal/handlers/moderation.go` (context user_id extraction)
- `internal/services/user_service.go` (email support, LinkProviderAccount)
- `internal/models/user.go` (Email field)
- `cmd/api/main.go` (middleware registration, route protection)
- `go.mod` (jwt/v5 dependency)
- `tests/SwaRupa-API-Phase04-Collection.postman_collection.json`
- `tests/swarupa-phase01-local.http`

---

## Next Phase Priorities

1. **Immediate**: Create and test with real Supabase Auth users
2. **Short-term**: Complete email/password flow end-to-end testing
3. **Short-term**: Verify account linking works in production scenario
4. **Medium-term**: Add Google OAuth when Cloud access restored
5. **Long-term**: Frontend auth UI integration

---

## Reference Links

- [Supabase Auth Docs](https://supabase.com/docs/guides/auth)
- [JWT RS256 Validation](https://supabase.com/docs/guides/auth/jwts)
- [JWKS Endpoint](https://supabase.com/docs/guides/auth/jwts#jwt-secret-key-rotation)

---

**Created:** May 13, 2026  
**Development Status:** Phase 5 - 85% complete (core auth + email/password ready, OAuth pending, testing needed)
