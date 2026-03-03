CREATE TABLE artists (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  musicbrainz_id TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE albums (
  id TEXT PRIMARY KEY,
  artist_id TEXT NOT NULL,
  title TEXT NOT NULL,
  release_year INTEGER,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (artist_id) REFERENCES artists(id)
);

CREATE TABLE artwork_sources (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  base_url TEXT,
  is_official BOOLEAN DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE artworks (
  id TEXT PRIMARY KEY,
  album_id TEXT NOT NULL,
  source_id TEXT NOT NULL,
  image_url TEXT NOT NULL,
  thumbnail_url TEXT,
  width INTEGER,
  height INTEGER,
  file_format TEXT,
  is_official BOOLEAN DEFAULT 0,
  submitted_by TEXT,
  approval_status TEXT DEFAULT 'pending',
  priority_score INTEGER DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE users (
  id TEXT PRIMARY KEY,
  email TEXT UNIQUE,
  role TEXT DEFAULT 'user',
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);