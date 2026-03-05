Phase 1 — Core CRUD -- done

Create / Get user
Create / Get artist
Create / Get album (with multiple artists via transaction)
Submit / Get artworks for an album (with filters + sorting)


Phase 2 — Moderation

PATCH /artworks/:id/approve
PATCH /artworks/:id/reject
Only users with role = admin can approve or reject
Filter artworks by approval_status


Phase 3 — Search

GET /search/artists?q=
GET /search/albums?q=
Powered by pg_trgm trigram indexes already in the schema
Fuzzy matching (handles typos)


Phase 4 — Pagination

Add ?page= and ?limit= to all list endpoints
Primarily for GET /albums/:id/artworks
Cursor-based pagination for future scaling


Phase 5 — Authentication

Integrate Firebase or Supabase Auth
Verify JWT on protected routes (submit artwork, approve/reject)
Middleware that extracts and validates user from token
submitted_by populated automatically from token, not from request body


Phase 6 — User Roles & Permissions

Role-based access control (admin vs contributor)
Admins: approve/reject artworks, delete records
Contributors: submit artists, albums, artworks only
Promote users to admin via PATCH /users/:id/role


Phase 7 — Extended Metadata

GET /artists/:id/albums — all albums by an artist
PATCH /artists/:id — update artist details
PATCH /albums/:id — update album details
DELETE endpoints (soft delete with deleted_at column)


Phase 8 — Production Hardening

Structured logging (replace log.Println with slog or zap)
Centralized error handling middleware
Rate limiting
Request validation middleware
Environment-based config (production vs development)
Graceful shutdown


Phase 9 — Frontend Integration Prep

CORS configuration
API versioning (/v1/...)
Standardized response envelope { data, error, meta }
OpenAPI / Swagger documentation