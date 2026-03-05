## Planned Endpoint List

### Users
| Method | Endpoint | Description |
|---|---|---|
| POST | `/users` | Register a user |
| GET | `/users/:id` | Get user by ID |

### Artists
| Method | Endpoint | Description |
|---|---|---|
| POST | `/artists` | Create an artist |
| GET | `/artists/:id` | Get artist by ID |
| GET | `/search/artists?q=` | Search artists by name |

### Albums
| Method | Endpoint | Description |
|---|---|---|
| POST | `/albums` | Create an album (with artist_ids) |
| GET | `/albums/:id` | Get album by ID (includes artists) |
| GET | `/search/albums?q=` | Search albums by title |

### Artworks
| Method | Endpoint | Description |
|---|---|---|
| POST | `/albums/:id/artworks` | Submit artwork for an album |
| GET | `/albums/:id/artworks` | Get all artworks for an album |
| GET | `/albums/:id/artworks?status=approved` | Filter by approval status |
| GET | `/albums/:id/artworks?official=true` | Filter official only |
| GET | `/albums/:id/artworks?sort=priority` | Sort by priority score |
| PATCH | `/artworks/:id/approve` | Approve an artwork |
| PATCH | `/artworks/:id/reject` | Reject an artwork |

---