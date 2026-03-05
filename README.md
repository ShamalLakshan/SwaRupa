<div align="center">

# SwaRupa

![](./docs/banner.png)

A collaborative music artwork database API for Sri Lankan Audiophiles.

SwaRupa allows users to submit, browse, and moderate album artwork. It is not a streaming service. The goal is to build a structured, crowdsourced metadata and artwork database for music albums for enthusiasts of Sri Lankan music and to whom metadata matters.

</div>

---

<div align="center">

## Tech Stack

</div>

- Go
- Gin (HTTP router)
- pgx v5 + pgxpool (PostgreSQL driver)
- Supabase (hosted PostgreSQL)

---

<div align="center">

## Features

</div>

- Artists and albums with many-to-many relationships
- Multiple artworks per album
- Artwork moderation workflow (pending, approved, rejected)
- Submission tracking per user
- Fuzzy search by artist name and album title

---

## Getting Started

1. Clone the repository
2. Create a `.env` file in the project root:

```
DATABASE_URL=your_supabase_connection_string
```

3. Run the server:

```bash
go run ./cmd/api
```

The server starts on `http://localhost:8080`.

---

## Environment Variables

| Variable | Description |
|---|---|
| `DATABASE_URL` | Full Supabase PostgreSQL connection string |

---

## License

MIT