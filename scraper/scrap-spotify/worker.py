import pika
import psycopg2
import json
import os
import yt_dlp
import time
from datetime import datetime
from spotify_scraper import SpotifyClient
from pika.exceptions import AMQPConnectionError, ConnectionClosedByBroker

# --- CONFIGURATION ---
DATABASE_URL = os.environ.get("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/suzam?sslmode=disable")
RABBITMQ_HOST = os.environ.get("RABBITMQ_HOST","localhost")
SONG_DOWNLOAD_PROCESSING_QUEUE_NAME = "song_download_processing"
SONG_FINGERPRINTING_PROCESSING_QUEUE_NAME = "song_fingerprinting_processing"

# Use absolute path for DOWNLOAD_DIR to ensure Go worker finds it easily
DOWNLOAD_DIR = os.path.abspath("./downloads")

spotifyClient = SpotifyClient()

def calculate_video_score(video_meta, spotify_data):
    score = 0
    vid_title = video_meta.get('title', '').lower()
    vid_duration = video_meta.get('duration', 0)
    
    duration_diff = abs(vid_duration - spotify_data['duration_seconds'])
    if duration_diff <= 3:
        score += 100
    elif duration_diff <= 10:
        score += 50
    else:
        score -= (duration_diff * 2)

    if spotify_data['name'].lower() in vid_title:
        score += 60
    
    for artist in spotify_data['artists']:
        if artist.lower() in vid_title:
            score += 40

    if any(word in vid_title for word in ['cover', 'remix', 'live']):
        if not any(word in spotify_data['name'].lower() for word in ['cover', 'remix', 'live']):
            score -= 100
    return score

def update_db_status(queue_id, status, song_name=None, authors=None, err_msg=None):
    conn = psycopg2.connect(DATABASE_URL)
    cursor = conn.cursor()
    try:
        if status == 'downloading':
            cursor.execute(
                "UPDATE queue SET status=%s, song_name=%s, authors=%s WHERE id=%s",
                (status, song_name, authors, queue_id)
            )
        elif status == 'failed':
            cursor.execute(
                "UPDATE queue SET status=%s, err_message=%s WHERE id=%s",
                (status, err_msg, queue_id)
            )
        else:
            cursor.execute("UPDATE queue SET status=%s WHERE id=%s", (status, queue_id))
        conn.commit()
    finally:
        conn.close()

def process_task(ch, method, properties, body):
    data = json.loads(body)
    queue_id = data.get("queue_id")
    track_url = data.get("spotify_url")

    print(f"[*] Processing Task {queue_id}: {track_url}")

    try:
        # 1. Get Spotify Metadata
        track = spotifyClient.get_track_info(track_url)
        if not track:
            raise Exception("Track not found on Spotify")

        artists_list = [a.get('name', 'Unknown') for a in track.get('artists', [])]
        artists_str = ", ".join(artists_list)
        song_name = track.get('name', 'Unknown')
        spotify_id = track.get('id')
        
        spotify_data = {
            "id": spotify_id,
            "name": song_name,
            "artists": artists_list,
            "duration_seconds": track.get('duration_ms', 0) / 1000
        }

        # 2. Update DB: Set to Downloading
        update_db_status(queue_id, 'downloading', song_name, artists_str)

        # 3. Search YouTube
        search_query = f"{spotify_data['name']} {artists_str} official audio"
        ydl_opts = {'format': 'bestaudio/best', 'quiet': True, 'no_warnings': True}
        
        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            search_results = ydl.extract_info(f"ytsearch5:{search_query}", download=False)
            videos = search_results.get('entries', [])

        if not videos:
            raise Exception("No matching videos found on YouTube")

        # 4. Score and Find Best Match
        best_video = max(videos, key=lambda v: calculate_video_score(v, spotify_data))

        # 5. Download as WAV
        if not os.path.exists(DOWNLOAD_DIR):
            os.makedirs(DOWNLOAD_DIR)
        
        file_base = os.path.join(DOWNLOAD_DIR, spotify_id)
        # Final expected path
        abs_wav_path = os.path.abspath(f"{file_base}.wav")
        
        download_opts = {
            'format': 'bestaudio/best',
            'outtmpl': file_base,
            'postprocessors': [{
                'key': 'FFmpegExtractAudio',
                'preferredcodec': 'wav',
                'preferredquality': '192',
            }],
            'quiet': True,
        }

        with yt_dlp.YoutubeDL(download_opts) as ydl:
            ydl.download([best_video['webpage_url']])

        # 6. Update DB Status
        update_db_status(queue_id, 'fingerprinting')

        # 7. Publish to Go Fingerprinting Worker
        fingerprint_payload = {
            "queue_id": queue_id,
            "spotify_id": spotify_id,
            "file_path": abs_wav_path,
            "duration_seconds": track.get('duration_ms', 0) / 1000
        }
        
        ch.basic_publish(
            exchange='',
            routing_key=SONG_FINGERPRINTING_PROCESSING_QUEUE_NAME,
            body=json.dumps(fingerprint_payload),
            properties=pika.BasicProperties(
                delivery_mode=2,
            )
        )
        
        print(f"[+] Task {queue_id} moved to Fingerprinting queue.")
        
        # Acknowledge the download task
        ch.basic_ack(delivery_tag=method.delivery_tag)

    except Exception as e:
        print(f"[!] Error processing task {queue_id}: {str(e)}")
        update_db_status(queue_id, 'failed', err_msg=str(e))
        ch.basic_ack(delivery_tag=method.delivery_tag)

def main():
    while True:
        try:
            print(" [*] Connecting to RabbitMQ...")
            connection = pika.BlockingConnection(pika.ConnectionParameters(host=RABBITMQ_HOST))
            channel = connection.channel()
            
            # Declare both queues to ensure they exist
            channel.queue_declare(queue=SONG_DOWNLOAD_PROCESSING_QUEUE_NAME, durable=True)
            channel.queue_declare(queue=SONG_FINGERPRINTING_PROCESSING_QUEUE_NAME, durable=True)
            
            channel.basic_qos(prefetch_count=1)
            channel.basic_consume(queue=SONG_DOWNLOAD_PROCESSING_QUEUE_NAME, on_message_callback=process_task)

            print(" [*] Suzam Download Worker Running. Waiting for messages...")
            channel.start_consuming()

        except (AMQPConnectionError, ConnectionClosedByBroker):
            print(" [!] Connection lost. Retrying in 5 seconds...")
            time.sleep(5)
            continue
        except Exception as e:
            print(f" [!] Unexpected error: {e}")
            time.sleep(5)
            continue
        except KeyboardInterrupt:
            print(" [*] Stopping worker...")
            break

if __name__ == "__main__":
    main()