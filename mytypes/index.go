package mytypes

type SongTask struct {
	QueueID    int64  `json:"queue_id"`
	SpotifyURL string `json:"spotify_url"`
}

type ClipFingerprint struct {
	Hash       uint32
	AnchorTime int
	Value      float64
}