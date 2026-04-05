package db

import (
	"database/sql"
	"math"
	"sort"
	"suzam-example/mytypes"
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
	var songID int64
	err := db.QueryRow(
		"INSERT INTO songs (spotify_id, title, authors, duration) VALUES ($1, $2, $3, $4) RETURNING id",
		s.SpotifyID, s.Title, s.Authors, s.Duration,
	).Scan(&songID)
	if err != nil {
		return 0, err
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	stmt, _ := tx.Prepare(`INSERT INTO fingerprints (hash, song_id, "offset") VALUES ($1, $2, $3)`)
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
		rows, err := db.Query(`SELECT song_id, "offset" FROM fingerprints WHERE hash = $1`, int64(qf.Hash))
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
	err := db.QueryRow("SELECT id, spotify_id, title, authors, duration FROM songs WHERE id = $1", bestSongID).
		Scan(&s.ID, &s.SpotifyID, &s.Title, &s.Authors, &s.Duration)

	return s, maxScore, err
}

type MatchPoint struct {
	SnippetTime int
	DBTime      int
}

func FindTop5Matchs(db *sql.DB, queryHashes []mytypes.ClipFingerprint) ([]SongWithMatchScore, error) {
	matchesBySong := make(map[int][]Fingerprint)

	for _, qf := range queryHashes {
		rows, err := db.Query(`SELECT song_id, hash, "offset" FROM fingerprints WHERE hash = $1`, int64(qf.Hash))
		if err != nil {
			continue
		}
		for rows.Next() {
			var songID int
			var dbHash uint32
			var dbOffset int
			if err := rows.Scan(&songID, &dbHash, &dbOffset); err != nil {
				continue
			}

			matchesBySong[songID] = append(matchesBySong[songID], Fingerprint{
				Hash:       dbHash,
				AnchorTime: dbOffset,
			})
		}
		rows.Close()
	}

	var snippetRef mytypes.ClipFingerprint
	maxVolume := -999.0

	searchLimit := 10
	if len(queryHashes) < 10 {
		searchLimit = len(queryHashes)
	}

	for i := 0; i < searchLimit; i++ {
		if queryHashes[i].Value > maxVolume {
			maxVolume = queryHashes[i].Value
			snippetRef = queryHashes[i]
		}
	}


	snippetLookup := make(map[uint32]int)
	for _, qf := range queryHashes {
		if _, exists := snippetLookup[qf.Hash]; !exists {
			snippetLookup[qf.Hash] = qf.AnchorTime
		}
	}

	type tempScore struct {
		id    int
		score int
	}
	var results []tempScore

	for songId, fingerprints := range matchesBySong {
		var dbRef Fingerprint
		foundRef := false
		for _, fp := range fingerprints {
			if fp.Hash == snippetRef.Hash {
				dbRef = fp
				foundRef = true
				break
			}
		}

		if !foundRef {
			continue 
		}

		score := 0
		for _, fp := range fingerprints {
			snippetCurrOffset, exists := snippetLookup[fp.Hash]
			if !exists {
				continue
			}

			a := math.Abs(float64(snippetRef.AnchorTime - snippetCurrOffset))

			b := math.Abs(float64(dbRef.AnchorTime - fp.AnchorTime))

			c := math.Abs(a - b)

			if c < 120 {
				score++
			}
		}
		results = append(results, tempScore{id: songId, score: score})
	}

	// 3. Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// 4. Limit to Top 5 and Fetch Metadata
	limit := 5
	if len(results) < 5 {
		limit = len(results)
	}

	finalMatches := []SongWithMatchScore{}
	for i := 0; i < limit; i++ {
		res := results[i]
		if res.score == 0 {
			continue
		}

		var s Song
		var authors string
		err := db.QueryRow("SELECT id, spotify_id, title, authors, duration FROM songs WHERE id = $1", res.id).
			Scan(&s.ID, &s.SpotifyID, &s.Title, &authors, &s.Duration)
		if err != nil {
			continue
		}
		s.Authors = authors

		finalMatches = append(finalMatches, SongWithMatchScore{
			Song:  s,
			Score: res.score,
		})
	}

	return finalMatches, nil
}

func AddToQueue(db *sql.DB, spotifyID, songName, authors string) (int64, error) {
	var id int64
	err := db.QueryRow(
		"INSERT INTO queue (spotify_id, song_name, authors, status) VALUES ($1, $2, $3, 'pending') RETURNING id",
		spotifyID, songName, authors,
	).Scan(&id)
	return id, err
}

func UpdateQueueData(db *sql.DB, queueID int, status string, errMsg string) error {
	var err error
	if status == "completed" {
		// Set completed timestamp if finishing
		_, err = db.Exec(
			"UPDATE queue SET status = $1, err_message = $2, completed = CURRENT_TIMESTAMP WHERE id = $3",
			status, errMsg, queueID,
		)
	} else {
		// Standard status update (e.g., 'downloading', 'failed')
		_, err = db.Exec(
			"UPDATE queue SET status = $1, err_message = $2 WHERE id = $3",
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
