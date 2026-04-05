# Suzam

A song identification system inspired by Shazam. Record or upload a short audio clip and Suzam fingerprints it against a database of indexed songs to find a match.

## How it works

1. Songs are imported via a Spotify URL
2. A Python worker finds and downloads the audio from YouTube
3. A Go worker generates acoustic fingerprints using FFT + constellation mapping
4. Fingerprints are stored in PostgreSQL
5. To identify a clip, its fingerprints are matched against the database

## Architecture

```
Frontend (React/TanStack)
        │
        ▼
Go HTTP Server (:3333)
        │
        ├──► PostgreSQL (songs, fingerprints, queue)
        │
        └──► RabbitMQ
                │
                ├──► Python Worker  (downloads audio from YouTube)
                │
                └──► Go Worker      (generates & stores fingerprints)
```

## Stack

- **Go** — HTTP API server + fingerprinting worker
- **Python** — Spotify metadata scraper + YouTube downloader (`yt-dlp`)
- **React + TanStack Start** — frontend
- **PostgreSQL** — persistent storage
- **RabbitMQ** — job queue between workers

## Running with Docker

```bash
docker compose up --build
```

| Service | URL |
|---|---|
| Frontend | http://localhost:3000 |
| API | http://localhost:3333 |
| RabbitMQ UI | http://localhost:15672 |
| PostgreSQL | localhost:5432 |

RabbitMQ default credentials: `guest` / `guest`

## Running locally

### Prerequisites

- Go 1.25+
- Python 3.14+ with `uv`
- Node.js 22+ with `pnpm`
- PostgreSQL
- RabbitMQ
- ffmpeg

### Environment variables

All services read configuration from environment variables:

| Variable | Used by | Example |
|---|---|---|
| `DATABASE_URL` | Go server, Go worker, Python worker | `postgres://suzam:suzam@localhost:5432/suzam?sslmode=disable` |
| `RABBITMQ_URL` | Go server | `amqp://guest:guest@localhost:5672/` |
| `RABBITMQ_HOST` | Go worker, Python worker | `amqp://guest:guest@localhost:5672/` |

### Start each service

```bash
# Go HTTP server
DATABASE_URL=... RABBITMQ_URL=... go run main.go

# Go fingerprinting worker
DATABASE_URL=... RABBITMQ_HOST=... go run RabbitMQ/worker.go

# Python download worker
cd scraper/scrap-spotify
DATABASE_URL=... RABBITMQ_HOST=... uv run python worker.py

# Frontend
cd frontend
pnpm install
pnpm dev
```

## License

[MIT](LICENSE)
