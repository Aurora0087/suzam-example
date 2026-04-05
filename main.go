package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"suzam-example/db"
	"suzam-example/httpserver"

	amqp "github.com/rabbitmq/amqp091-go"
)

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {

	SONG_DOWNLOAD_PROCESSING_QUEUE_NAME := "song_download_processing"

	// 1. Init DB

	dsn := os.Getenv("DATABASE_URL")
	database, err := db.InitDB(dsn)
	if err != nil {
		fmt.Println("Can't connect to db, error :", err)
		panic(err)
	}

	// 2. Init RabbitMQ
	rabbitURL := os.Getenv("RABBITMQ_URL")
	conn, _ := amqp.Dial(rabbitURL)
	ch, _ := conn.Channel()
	defer ch.Close()

	// Declare the queue (ensure it exists)
	ch.QueueDeclare(SONG_DOWNLOAD_PROCESSING_QUEUE_NAME, true, false, false, false, nil)

	ctx := &httpserver.HandlerContext{
		DB:       database,
		AMQPChan: ch,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", httpserver.GetRoot)
	mux.HandleFunc("/hello", httpserver.GetHello)

	mux.HandleFunc("/v1/api/songs/import", ctx.PostImportSong)
	mux.HandleFunc("/v1/api/songs", ctx.GetStoredSongs)
	mux.HandleFunc("/v1/api/songs/queues", ctx.GetQueueedSongs)
	mux.HandleFunc("/v1/api/songs/identify", ctx.IdentifySongFromSortClip)

	fmt.Println("[  ]Server starting on :3333")

	err = http.ListenAndServe(":3333", enableCORS(mux))

	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
