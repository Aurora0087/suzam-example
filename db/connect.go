package db

import (
	"database/sql"
_ "github.com/mattn/go-sqlite3"
)


func InitDB(path string) (*sql.DB,error) {
	db, err := sql.Open("sqlite3",path)

	if err != nil {
		return nil, err
	}

	sqlStmt := `
	CREATE TABLE IF NOT EXISTS songs (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT);
	CREATE TABLE IF NOT EXISTS fingerprints (hash INTEGER, song_id INTEGER, offset INTEGER);
	CREATE INDEX IF NOT EXISTS idx_hash ON fingerprints(hash);
	`
	_, err = db.Exec(sqlStmt)
	return db, err
}