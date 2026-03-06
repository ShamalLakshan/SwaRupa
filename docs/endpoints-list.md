## Planned Endpoint List

### Users
| Method | Endpoint | Description |
|---|---|---|
| POST | `/api/users` | Register a user |
| GET | `/api/users/:id` | Get user by ID |

### Artists
| Method | Endpoint | Description |
|---|---|---|
| POST | `/api/artists` | Create an artist |
| GET | `/api/artists/:id` | Get artist by ID |
| GET | `/api/search/artists?q=` | Search artists by name |

### Albums
| Method | Endpoint | Description |
|---|---|---|
| POST | `/api/albums` | Create an album (with artist_ids) |
| GET | `/api/albums/:id` | Get album by ID (includes artists) |
| GET | `/api/search/albums?q=` | Search albums by title |

### Artworks
| Method | Endpoint | Description |
|---|---|---|
| POST | `/api/albums/:id/artworks` | Submit artwork for an album |
| GET | `/api/albums/:id/artworks` | Get all artworks for an album |
| GET | `/api/albums/:id/artworks?status=approved` | Filter by approval status |
| GET | `/api/albums/:id/artworks?official=true` | Filter official only |
| GET | `/api/albums/:id/artworks?sort=priority` | Sort by priority score |
| PATCH | `/api/artworks/:id/approve` | Approve an artwork |
| PATCH | `/api/artworks/:id/reject` | Reject an artwork |

---