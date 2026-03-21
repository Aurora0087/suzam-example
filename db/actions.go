package db

import (
	"database/sql"
	"sort"
	"time"
)

type Fingerprint struct {
	Hash       uint32
	AnchorTime int
}

type Song struct {
	ID        int       `json:"id"`
	SpotifyID string    `json:"spotify_id"`
	Title     string    `json:"title"`
	Authors   string    `json:"authors"`
	Duration  float64   `json:"duration"`
	Added     time.Time `json:"added"`
}

type QueueItem struct {
	ID        int
	SpotifyID string
	SongName  string
	Authors   string
	Status    string // 'pending', 'downloading', 'fingerprinting', 'completed', 'failed'
}

type SongWithMatchScore struct {
	Song  Song `json:"song"`
	Score int  `json:"score"`
}

func StoreSong(db *sql.DB, s Song, fingerprints []Fingerprint) (int64, error) {
	res, err := db.Exec(
		"INSERT INTO songs (spotify_id, title, authors, duration) VALUES (?, ?, ?, ?)",
		s.SpotifyID, s.Title, s.Authors, s.Duration,
	)
	if err != nil {
		return 0, err
	}

	songID, _ := res.LastInsertId()

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	stmt, _ := tx.Prepare("INSERT INTO fingerprints (hash, song_id, offset) VALUES (?, ?, ?)")
	defer stmt.Close()

	for _, f := range fingerprints {
		_, err = stmt.Exec(int64(f.Hash), songID, f.AnchorTime)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	return songID, tx.Commit()
}

func FindMatch(db *sql.DB, queryHashes []Fingerprint) (Song, int, error) {
	hits := make(map[int]map[int]int)

	for _, qf := range queryHashes {
		rows, err := db.Query("SELECT song_id, offset FROM fingerprints WHERE hash = ?", int64(qf.Hash))
		if err != nil {
			continue
		}

		for rows.Next() {
			var songID, dbOffset int
			rows.Scan(&songID, &dbOffset)

			dif := dbOffset - qf.AnchorTime
			if hits[songID] == nil {
				hits[songID] = make(map[int]int)
			}
			hits[songID][dif]++
		}
		rows.Close()
	}

	var bestSongID, maxScore int
	for songID, diffMap := range hits {
		for _, score := range diffMap {
			if score > maxScore {
				maxScore = score
				bestSongID = songID
			}
		}
	}

	if maxScore == 0 {
		return Song{}, 0, nil
	}

	var s Song
	err := db.QueryRow("SELECT id, spotify_id, title, authors, duration FROM songs WHERE id = ?", bestSongID).
		Scan(&s.ID, &s.SpotifyID, &s.Title, &s.Authors, &s.Duration)

	return s, maxScore, err
}

func FindTop5Matchs(db *sql.DB, queryHashes []Fingerprint) ([]SongWithMatchScore, error) {
	hits := make(map[int]map[int]int)

	for _, qf := range queryHashes {
		rows, err := db.Query("SELECT song_id, offset FROM fingerprints WHERE hash = ?", int64(qf.Hash))
		if err != nil {
			continue
		}

		for rows.Next() {
			var songID, dbOffset int
			rows.Scan(&songID, &dbOffset)

			dif := dbOffset - qf.AnchorTime
			if hits[songID] == nil {
				hits[songID] = make(map[int]int)
			}
			hits[songID][dif]++
		}
		rows.Close()
	}

	type tempScore struct {
		id    int
		score int
	}
	var results []tempScore

	for songID, diffMap := range hits {
		songMax := 0
		for _, count := range diffMap {
			if count > songMax {
				songMax = count
			}
		}
		results = append(results, tempScore{id: songID, score: songMax})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	limit := 5
	if len(results) < 5 {
		limit = len(results)
	}
	topResults := results[:limit]

	finalMatches := []SongWithMatchScore{}
	for _, res := range topResults {
		var s Song
		var authors string
		err := db.QueryRow("SELECT id, spotify_id, title, authors, duration FROM songs WHERE id = ?", res.id).
			Scan(&s.ID, &s.SpotifyID, &s.Title, &authors, &s.Duration)

		if err != nil {
			continue
		}

		finalMatches = append(finalMatches, SongWithMatchScore{
			Song:  s,
			Score: res.score,
		})
	}

	return finalMatches, nil
}

func AddToQueue(db *sql.DB, spotifyID, songName, authors string) (int64, error) {
	res, err := db.Exec(
		"INSERT INTO queue (spotify_id, song_name, authors, status) VALUES (?, ?, ?, 'pending')",
		spotifyID, songName, authors,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func UpdateQueueData(db *sql.DB, queueID int, status string, errMsg string) error {
	var err error
	if status == "completed" {
		// Set completed timestamp if finishing
		_, err = db.Exec(
			"UPDATE queue SET status = ?, err_message = ?, completed = CURRENT_TIMESTAMP WHERE id = ?",
			status, errMsg, queueID,
		)
	} else {
		// Standard status update (e.g., 'downloading', 'failed')
		_, err = db.Exec(
			"UPDATE queue SET status = ?, err_message = ? WHERE id = ?",
			status, errMsg, queueID,
		)
	}
	return err
}

func GetNextInQueue(db *sql.DB) (QueueItem, error) {
	var q QueueItem
	err := db.QueryRow("SELECT id, spotify_id, song_name, authors FROM queue WHERE status = 'pending' ORDER BY added ASC LIMIT 1").
		Scan(&q.ID, &q.SpotifyID, &q.SongName, &q.Authors)
	return q, err
}
