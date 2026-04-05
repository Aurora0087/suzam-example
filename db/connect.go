package db

import (
	"database/sql"
	_ "github.com/lib/pq"
)

func InitDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	sqlStmt := `
-- 1. Main Songs Table
CREATE TABLE IF NOT EXISTS songs (
    id SERIAL PRIMARY KEY,
    spotify_id TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    authors TEXT,
    duration DOUBLE PRECISION,
    added TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. Fingerprints Table (Linking hashes to songs)
CREATE TABLE IF NOT EXISTS fingerprints (
    hash BIGINT NOT NULL,
    song_id INTEGER NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    "offset" INTEGER NOT NULL
);

-- 3. Queue Table (For background processing)
CREATE TABLE IF NOT EXISTS queue (
    id SERIAL PRIMARY KEY,
    spotify_id TEXT NOT NULL,
    song_name TEXT,
    authors TEXT,
    status TEXT DEFAULT 'pending',
    err_message TEXT,
    added TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed TIMESTAMP
);

-- 4. Indexes for Speed
CREATE INDEX IF NOT EXISTS idx_hash ON fingerprints(hash);
CREATE INDEX IF NOT EXISTS idx_queue_status ON queue(status);
CREATE INDEX IF NOT EXISTS idx_song_spotify_id ON songs(spotify_id);
`
	_, err = db.Exec(sqlStmt)
	return db, err
}
