package mytypes

type SongTask struct {
	QueueID    int64  `json:"queue_id"`
	SpotifyURL string `json:"spotify_url"`
}