package httpserver

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"suzam-example/db"
	"suzam-example/mytypes"
	"suzam-example/suzam"
	"suzam-example/utils"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	amqp "github.com/rabbitmq/amqp091-go"
)


func GetRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("got / request\n")
	io.WriteString(w, "This is my website!\n")
}
func GetHello(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("got /hello request\n")
	io.WriteString(w, "Hello, HTTP!\n")
}

type ImportRequest struct {
	URL string `json:"url"`
}

type QueueItem struct {
	ID         int            `json:"id"`
	SpotifyID  string         `json:"spotify_id"`
	SongName   string         `json:"song_name"`
	Authors    string         `json:"authors"`
	Status     string         `json:"status"`
	ErrMessage sql.NullString `json:"err_message"`
	Added      string         `json:"added"`
	Completed  sql.NullString `json:"completed"`
}

type HandlerContext struct {
	DB       *sql.DB
	AMQPChan *amqp.Channel
}



func (h *HandlerContext) PostImportSong(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ImportRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || !strings.HasPrefix(req.URL, "https://open.spotify.com/track/") {
		http.Error(w, "Invalid Spotify URL", http.StatusBadRequest)
		return
	}

	spotifyID := utils.ExtractSpotifyID(req.URL)
	if spotifyID == "" {
		http.Error(w, "Could not parse Spotify ID", http.StatusBadRequest)
		return
	}

	var exists bool
	err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM songs WHERE spotify_id = ?)", spotifyID).Scan(&exists)
	if exists {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Song is already indexed in the database.",
		})
		return
	}

	var queueExists bool
	err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM queue WHERE spotify_id = ? AND status != 'failed')", spotifyID).Scan(&queueExists)
	if queueExists {
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Song is currently being processed. Please wait.",
		})
		return
	}
	queueID, err := db.AddToQueue(h.DB, spotifyID, "Pending Metadata", "Unknown")
	if err != nil {
		 w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]any{"error": "Database error"})
		return
	}

	task := mytypes.SongTask{
		QueueID:    queueID,
		SpotifyURL: req.URL,
	}
	body, _ := json.Marshal(task)

	err = h.AMQPChan.Publish(
		"",
		"song_download_processing",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, "Failed to queue task", http.StatusInternalServerError)
		return
	}

	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"queue_id": queueID,
		"message":  "Song added to processing queue",
	})
}

func (h *HandlerContext) GetQueueedSongs(w http.ResponseWriter, r *http.Request) {
	// 1. Parse Query Parameters
	query := r.URL.Query()

	// Defaults: ignore=0, limit=10, status='all'
	ignore, _ := strconv.Atoi(query.Get("ignore"))
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit <= 0 {
		limit = 10
	}
	status := query.Get("status")
	if status == "" {
		status = "all"
	}

	// 2. Build Dynamic SQL
	// Start with the base query
	sqlQuery := "SELECT id, spotify_id, song_name, authors, status, err_message, added, completed FROM queue"
	var args []any

	// Add WHERE clause if filtering by status
	if status != "all" {
		sqlQuery += " WHERE status = ?"
		args = append(args, status)
	}

	// Add Ordering and Pagination
	sqlQuery += " ORDER BY added DESC LIMIT ? OFFSET ?"
	args = append(args, limit, ignore)

	// 3. Execute Query
	rows, err := h.DB.Query(sqlQuery, args...)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// 4. Scan Rows
	queues := []QueueItem{}
	for rows.Next() {
		var q QueueItem
		err := rows.Scan(
			&q.ID,
			&q.SpotifyID,
			&q.SongName,
			&q.Authors,
			&q.Status,
			&q.ErrMessage,
			&q.Added,
			&q.Completed,
		)
		if err != nil {
			continue
		}
		queues = append(queues, q)
	}

	// 5. Send JSON Response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(queues)
}


type SongResponse struct {
	ID        int      `json:"id"`
	SpotifyID string   `json:"spotify_id"`
	Title     string   `json:"title"`
	Artists   []string `json:"artists"` // Converted from comma-separated string
	Time      float64  `json:"time"`    // Duration in ms (as expected by your frontend)
	Added     string   `json:"added"`
}

func (h *HandlerContext) GetStoredSongs(w http.ResponseWriter, r *http.Request) {
	// 1. Parse Query Parameters
	query := r.URL.Query()
	ignore, _ := strconv.Atoi(query.Get("ignore"))
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit <= 0 { limit = 10 }

	// 2. Get TOTAL COUNT from DB (New Step)
	var totalCount int
	err := h.DB.QueryRow("SELECT COUNT(*) FROM songs").Scan(&totalCount)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. Execute SQL Query for paginated data
	rows, err := h.DB.Query(
		"SELECT id, spotify_id, title, authors, duration, added FROM songs ORDER BY added DESC LIMIT ? OFFSET ?",
		limit, ignore,
	)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// 4. Process Rows
	songs := []SongResponse{}
	for rows.Next() {
		var id int
		var spotifyID, title, authors, added string
		var durationSec float64

		err := rows.Scan(&id, &spotifyID, &title, &authors, &durationSec, &added)
		if err != nil {
			continue
		}

		songs = append(songs, SongResponse{
			ID:        id,
			SpotifyID: spotifyID,
			Title:     title,
			Artists:   strings.Split(authors, ", "),
			Time:      durationSec * 1000,
			Added:     added,
		})
	}

	// 5. Send Wrapper Object (New Step)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := struct {
		Total int            `json:"total"`
		Songs []SongResponse `json:"songs"`
	}{
		Total: totalCount,
		Songs: songs,
	}

	json.NewEncoder(w).Encode(response)
}


func (h *HandlerContext) IdentifySongFromSortClip(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "File too large"})
		return
	}

	file, header, err := r.FormFile("audio")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Could not read audio file"})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".wav" {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		json.NewEncoder(w).Encode(map[string]string{"error": "Only .wav allowed"})
		return
	}

	clipDir := "./clip"
	os.MkdirAll(clipDir, os.ModePerm)
	os.MkdirAll("./clip-identify", os.ModePerm)


	savePath := filepath.Join(clipDir, uuid.New().String()+ext)
	dst, err := os.Create(savePath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Internal save error"})
		return
	}
	
	_, err = io.Copy(dst, file)
	dst.Close()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Write error"})
		return
	}

	top5Matches, err := suzam.FindSongFromClip("./clip-identify", savePath, h.DB)

	defer os.Remove(savePath)

	if err != nil {
		fmt.Printf("Identification error: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"success": false, 
			"error": "Failed to analyze audio",
		})
		return
	}

	if len(top5Matches) == 0 {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "No matching songs found in database",
			"matches": []interface{}{},
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"count":   len(top5Matches),
		"matches": top5Matches,
	})
}