import { Button } from '#/components/ui/button'
import { createFileRoute } from '@tanstack/react-router'
import { useRef, useState } from 'react'
import { Upload, Music, Loader2, X, Play } from 'lucide-react'
import { FaSpotify } from 'react-icons/fa'

export const Route = createFileRoute('/')({ component: App })

type Match = {
  song: {
    id: number
    spotify_id: string
    title: string
    authors: string
    duration: number
  }
  score: number
}

function App() {
  const [isRecording, setIsRecording] = useState(false)
  const [audioURL, setAudioURL] = useState<string | null>(null)
  const [audioBlob, setAudioBlob] = useState<Blob | null>(null)
  const [status, setStatus] = useState('Ready to identify')
  const [matches, setMatches] = useState<Match[]>([])
  const [isIdentifying, setIsIdentifying] = useState(false)

  const mediaRecorderRef = useRef<MediaRecorder | null>(null)
  const audioChunksRef = useRef<Blob[]>([])
  const fileInputRef = useRef<HTMLInputElement>(null)

  // 1. Handle File Upload
  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      if (file.type !== 'audio/wav' && !file.name.endsWith('.wav')) {
        setStatus('Please upload a .wav file')
        return
      }
      setAudioBlob(file)
      setAudioURL(URL.createObjectURL(file))
      setMatches([])
      setStatus('File loaded')
    }
  }

  // 2. Start/Stop Recording (Existing logic)
  const startRecording = async () => {
    setAudioURL(null)
    setAudioBlob(null)
    setMatches([])
    audioChunksRef.current = []
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      const mediaRecorder = new MediaRecorder(stream)
      mediaRecorder.ondataavailable = (event) => {
        if (event.data.size > 0) audioChunksRef.current.push(event.data)
      }
      mediaRecorder.onstop = () => {
        const blob = new Blob(audioChunksRef.current, { type: 'audio/wav' })
        setAudioBlob(blob)
        setAudioURL(URL.createObjectURL(blob))
        stream.getTracks().forEach((track) => track.stop())
      }
      mediaRecorderRef.current = mediaRecorder
      mediaRecorder.start()
      setIsRecording(true)
      setStatus('Listening...')
    } catch (err) {
      setStatus('Microphone access denied')
    }
  }

  const stopRecording = () => {
    if (mediaRecorderRef.current) {
      mediaRecorderRef.current.stop()
      setIsRecording(false)
      setStatus('Recording captured')
    }
  }

  // 3. Identification Logic
  const identifySong = async () => {
    if (!audioBlob) return
    setIsIdentifying(true)
    setStatus('Analyzing audio fingerprints...')

    const formData = new FormData()
    formData.append('audio', audioBlob, 'query.wav')

    try {
      const response = await fetch(
        'http://localhost:3333/v1/api/songs/identify',
        {
          method: 'POST',
          body: formData,
        },
      )
      const result = await response.json()

      if (result.success && result.matches.length > 0) {
        setMatches(result.matches)
        setStatus(`Found ${result.matches.length} possible matches`)
      } else {
        setMatches([])
        setStatus('No matches found in database.')
      }
    } catch (err) {
      setStatus('Server connection failed')
    } finally {
      setIsIdentifying(false)
    }
  }

  return (
    <main className="page-wrap relative px-4 pb-12 pt-24 flex h-full min-h-screen flex-col items-center bg-background">
      <div className="text-center mb-8">
        <h1 className="text-5xl font-black mb-2 tracking-tighter italic text-primary uppercase">
          ZAMZAM
        </h1>
        <p className="text-[10px] uppercase tracking-[0.4em] font-bold text-muted-foreground">
          {status}
        </p>
      </div>

      <div className="flex flex-col items-center gap-8 w-full max-w-lg">
        {/* Record Button */}
        <div className="relative">
          {isRecording && (
            <div className="absolute inset-0 bg-primary rounded-full animate-ping opacity-20" />
          )}
          <button
            onMouseDown={startRecording}
            onMouseUp={stopRecording}
            onTouchStart={startRecording}
            onTouchEnd={stopRecording}
            className={`relative z-10 w-32 h-32 rounded-full border-4 border-background transition-all flex flex-col items-center justify-center shadow-xl ${
              isRecording ? 'bg-red-500 scale-95' : 'bg-primary hover:scale-105'
            }`}
          >
            <Music
              className={`h-8 w-8 mb-1 ${isRecording ? 'text-white' : 'text-primary-foreground'}`}
            />
            <span className="text-[10px] font-black uppercase tracking-widest text-primary-foreground">
              {isRecording ? 'Release' : 'Hold'}
            </span>
          </button>
        </div>

        {/* Action Buttons */}
        <div className="flex gap-4">
          <input
            type="file"
            ref={fileInputRef}
            onChange={handleFileChange}
            accept=".wav"
            className="hidden"
          />
          <Button
            variant="outline"
            className="rounded-none border-2 border-primary/20 bg-secondary"
            onClick={() => fileInputRef.current?.click()}
          >
            <Upload className="h-4 w-4 mr-2" /> Upload .WAV
          </Button>
          {(audioURL || matches.length > 0) && (
            <Button
              variant="ghost"
              className="rounded-none text-xs uppercase font-bold"
              onClick={() => {
                setAudioURL(null)
                setMatches([])
                setStatus('Ready')
              }}
            >
              <X className="h-4 w-4 mr-2" /> Reset
            </Button>
          )}
        </div>

        {/* Identification Progress / Results */}
        {audioURL && matches.length === 0 && (
          <div className="w-full max-w-sm relative">
            <div className="relative z-10 w-full p-6 bg-secondary border-2 border-primary animate-in fade-in slide-in-from-bottom-4">
              <p className="text-[10px] uppercase tracking-widest font-black text-muted-foreground mb-4">
                Clip Captured
              </p>
              <audio
                src={audioURL}
                controls
                className="w-full mb-6 h-10 rounded-none invert dark:invert-0"
              />
              <Button
                onClick={identifySong}
                disabled={isIdentifying}
                className="rounded-none w-full h-12 text-lg font-black uppercase italic"
              >
                {isIdentifying ? (
                  <Loader2 className="animate-spin" />
                ) : (
                  'Identify Music'
                )}
              </Button>
            </div>
            <div className="absolute inset-0 z-0 translate-x-3 translate-y-3 border-2 border-primary stripe-gray-bg" />
          </div>
        )}

        {/* Top 5 Matches List */}
        {matches.map((match, idx) => (
          <div key={idx} className="relative group w-full">
            <div className="relative z-10 flex items-center justify-between p-4 bg-secondary border-2 border-primary/20 group-hover:border-primary">
              <div className="flex items-center gap-4">
                <div className="font-mono font-bold">{idx + 1}</div>
                <div>
                  <p className="font-bold text-sm leading-none mb-1">
                    {match.song.title}
                  </p>
                  <p className="text-[10px] uppercase font-bold text-muted-foreground">
                    {match.song.authors}
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-4">
                <div className="text-right">
                  <p className="text-[10px] font-black text-primary">SCORE</p>
                  {/* USE match.score instead of match.Score */}
                  <p className="font-mono text-lg font-bold leading-none">
                    {match.score}
                  </p>
                </div>
                <a
                  href={`https://open.spotify.com/track/${match.song.spotify_id}`}
                  target="_blank"
                  className="p-2 bg-primary text-primary-foreground"
                >
                  <FaSpotify className="h-5 w-5" />
                </a>
              </div>
            </div>
          </div>
        ))}
      </div>

      <footer className="mt-auto pt-12 text-[9px] text-slate-500 uppercase tracking-[0.5em] font-bold text-center">
        Powered by Suzam Fingerprinting Engine v1.0
      </footer>
    </main>
  )
}
