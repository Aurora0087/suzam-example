package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"suzam-example/db"
	"suzam-example/suzam"

	_ "github.com/mattn/go-sqlite3"
	amqp "github.com/rabbitmq/amqp091-go"
)

type FingerprintTask struct {
	QueueID   int    `json:"queue_id"`
	SpotifyID string `json:"spotify_id"`
	FilePath  string `json:"file_path"`
	DurationSeconds float64 `json:"duration_seconds"`
}

func main() {
	database, err := sql.Open("sqlite3", "./suzam.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	qName := "song_fingerprinting_processing"
	_, err = ch.QueueDeclare(qName, true, false, false, false, nil)

	msgs, err := ch.Consume(qName, "", false, false, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(" [] Go Fingerprinting Worker waiting for messages...")

	for d := range msgs {
		var task FingerprintTask
		err := json.Unmarshal(d.Body, &task)
		if err != nil {
			log.Printf("Error decoding task: %s", err)
			d.Ack(false)
			continue
		}

		fmt.Printf(" [] Fingerprinting Song: %s (ID: %d)\n", task.SpotifyID, task.QueueID)

		var title, authors string
		err = database.QueryRow("SELECT song_name, authors FROM queue WHERE id = ?", task.QueueID).Scan(&title, &authors)
		
		 suzam.MakefingarprintFromSong(task.QueueID, "./output-data", task.FilePath, title, task.SpotifyID, authors, task.DurationSeconds, database)

		if err != nil {
			log.Printf(" [ ] Fingerprinting Failed: %v", err)
			db.UpdateQueueData(database, task.QueueID, "failed", err.Error())
		} else {
			fmt.Printf(" [+] Fingerprinting Completed: %s\n", title)
			db.UpdateQueueData(database, task.QueueID, "completed", "")
		}

		d.Ack(false)
	}
}

