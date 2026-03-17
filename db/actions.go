package db

import "database/sql"

type Fingerprint struct {
	Hash       uint32
	AnchorTime int 
}

func StoreSong(db *sql.DB, title string, fingerprints []Fingerprint) error {
	res, err := db.Exec("INSERT INTO songs (title) VALUES (?)", title)

	if err != nil {
		return err
	}

	songID, _ := res.LastInsertId()

	// transaction
	tx, err := db.Begin()

	if err != nil {
		return err
	}

	stmt, _ := tx.Prepare("INSERT INTO fingerprints (hash, song_id, offset) VALUES (?, ?, ?)")

	for _, f := range fingerprints {
		_, err = stmt.Exec(int64(f.Hash), songID, f.AnchorTime)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()

}

func FindMatch(db *sql.DB, queryHashes []Fingerprint) (string, int, error) {
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

	var title string
	err := db.QueryRow("SELECT title FROM songs WHERE id = ?", bestSongID).Scan(&title)

	return title, maxScore, err

}
