package db

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

func InitDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)

	if err != nil {
		return nil, err
	}

	sqlStmt := `
-- 1. Main Songs Table
CREATE TABLE IF NOT EXISTS songs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    spotify_id TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    authors TEXT,          -- Store as a comma-separated string or JSON
    duration REAL,         -- Store in seconds
    added DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 2. Fingerprints Table (Linking hashes to songs)
CREATE TABLE IF NOT EXISTS fingerprints (
    hash INTEGER NOT NULL,
    song_id INTEGER NOT NULL,
    offset INTEGER NOT NULL,
    FOREIGN KEY(song_id) REFERENCES songs(id) ON DELETE CASCADE
);

-- 3. Queue Table (For background processing)
CREATE TABLE IF NOT EXISTS queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    spotify_id TEXT NOT NULL,
    song_name TEXT,
    authors TEXT,
    status TEXT DEFAULT 'pending', -- 'pending', 'downloading', 'fingerprinting', 'completed', 'failed'
    err_message TEXT,
    added DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed DATETIME
);

-- 4. Indexes for Speed
CREATE INDEX IF NOT EXISTS idx_hash ON fingerprints(hash);
CREATE INDEX IF NOT EXISTS idx_queue_status ON queue(status);
CREATE INDEX IF NOT EXISTS idx_song_spotify_id ON songs(spotify_id);
`
	_, err = db.Exec(sqlStmt)
	return db, err
}
