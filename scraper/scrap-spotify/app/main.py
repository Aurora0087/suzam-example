import os

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import List
import yt_dlp
from spotify_scraper import SpotifyClient


app = FastAPI()


@app.get("/")
async def root():
    return {"message": "Hello Bro!"}


spotifyClient = SpotifyClient()


class TrackRequest(BaseModel):
    url: str

class TrackMetadata(BaseModel):
    id: str
    name: str
    artists: List[str]
    duration_seconds: float


@app.post("/getSongMetadata", response_model=TrackMetadata)
async def get_song_details(request: TrackRequest):
    track_url = request.url

    # Validation
    if not track_url.startswith("https://open.spotify.com/track/"):
        raise HTTPException(status_code=400, detail="Invalid Spotify track URL")

    try:
        # Assuming spotifyClient is already initialized in your scope
        track = spotifyClient.get_track_info(track_url)
        
        if not track:
            raise HTTPException(status_code=404, detail="Track not found on Spotify")

        # Extracting the list of artist names
        artists = [artist.get('name', 'Unknown') for artist in track.get('artists', [])]
        
        # Duration calculation
        duration_ms = track.get('duration_ms', 0)
        duration_seconds = duration_ms / 1000

        return {
            "id": track.get('id', 'Unknown'),
            "name": track.get('name', 'Unknown'),
            "artists": artists,
            "duration_seconds": duration_seconds
        }

    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Spotify Scraper Error: {str(e)}")



def calculate_video_score(video_meta, spotify_data):
    """
    Scores a YouTube video based on how well it matches Spotify metadata.
    Higher is better.
    """
    score = 0
    vid_title = video_meta.get('title', '').lower()
    vid_duration = video_meta.get('duration', 0)
    
    # 1. Duration Check (The most reliable metric)
    # If the duration is within 3 seconds of the Spotify track, +100 points
    duration_diff = abs(vid_duration - spotify_data['duration_seconds'])
    if duration_diff <= 3:
        score += 100
    elif duration_diff <= 10:
        score += 50
    else:
        # Subtract points for huge duration mismatches (prevents "10 hour loop" or "Live" versions)
        score -= duration_diff 

    # 2. Title Match
    # Does the title contain the song name?
    if spotify_data['name'].lower() in vid_title:
        score += 50
    
    # 3. Artist Match
    for artist in spotify_data['artists']:
        if artist.lower() in vid_title:
            score += 30

    # 4. Keyword Penalties (Filter out "Cover", "Live", "Remix" if Spotify doesn't mention them)
    clean_keywords = ['cover', 'live', 'remix', 'instrumental']
    for word in clean_keywords:
        if word in vid_title and word not in spotify_data['name'].lower():
            score -= 40

    return score

@app.post("/downloadSong")
async def get_song_details(request: TrackRequest):
    track_url = request.url

    # Validation
    if not track_url.startswith("https://open.spotify.com/track/"):
        raise HTTPException(status_code=400, detail="Invalid Spotify track URL")

    try:
        # Assuming spotifyClient is already initialized in your scope
        track = spotifyClient.get_track_info(track_url)
        
        if not track:
            raise HTTPException(status_code=404, detail="Track not found on Spotify")

        # Extracting the list of artist names
        artists = [artist.get('name', 'Unknown') for artist in track.get('artists', [])]

        spotify_data = {
            "id": track.get('id'),
            "name": track.get('name'),
            "artists": artists,
            "duration_seconds": track.get('duration_ms', 0) / 1000
        }
        
        # search to 5 video on youtube

        search_query = f"{spotify_data['name']} {' '.join(artists)} official audio"
        ydl_opts = {
            'format': 'bestaudio/best',
            'quiet': True,
            'no_warnings': True,
        }

        videos = []

        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            search_results = ydl.extract_info(f"ytsearch5:{search_query}", download=False)
            videos = search_results.get('entries', [])
        
        if not videos:
            raise HTTPException(status_code=404, detail="No matching videos found on YouTube")
        
        

        # see which video closely resable with spotifys data [duration, name, artists]

        best_video = None
        max_score = -999

        for video in videos:
            score = calculate_video_score(video, spotify_data)
            if score > max_score:
                max_score = score
                best_video = video

        # download mp3 most scroed video

        output_dir = "downloads"
        if not os.path.exists(output_dir):
            os.makedirs(output_dir)
        
        file_path = os.path.join(output_dir, f"{spotify_data['id']}")

        download_opts = {
            'format': 'bestaudio/best',
            'outtmpl': file_path,
            'postprocessors': [{
                'key': 'FFmpegExtractAudio',
                'preferredcodec': 'wav',
                'preferredquality': '192',
            }],
            'quiet': True,
        }

        with yt_dlp.YoutubeDL(download_opts) as ydl:
            ydl.download([best_video['webpage_url']])

        return {
            "success": True,
            "id": spotify_data['id'],
            "name": spotify_data['name'],
            "artists": spotify_data['artists'],
            "file_path": f"{file_path}.wav",
            "match_score": max_score,
            "youtube_url": best_video['webpage_url']
        }

    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Spotify Scraper Error: {str(e)}")