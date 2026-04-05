import { useEffect, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { SongTable } from '#/components/songs/SongTable'
import { columns, type Song } from '@/components/songs/columns'
import { InputGroup, InputGroupAddon, InputGroupInput } from '@/components/ui/input-group'
import { Link2, Plus, Search, Loader2 } from 'lucide-react'
import { Button } from '#/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { toast } from "sonner"

export const Route = createFileRoute('/songs')({
  component: RouteComponent,
})

type SongsApiResponse = {
  total: number;
  songs: Song[];
}

// 1. Fetcher Function
const fetchSongs = async (title: string): Promise<SongsApiResponse> => {
  const response = await fetch(`http://localhost:3333/v1/api/songs?limit=100&ignore=0&songTitle=${encodeURIComponent(title)}`)
  if (!response.ok) {
    throw new Error('Failed to fetch songs from database')
  }
  return response.json()
}

function RouteComponent() {
  const queryClient = useQueryClient()
  
  const [searchInput, setSearchInput] = useState("") // Raw input value
  const [debouncedSearch, setDebouncedSearch] = useState("") // Value used for API call
  const [spotifyUrl, setSpotifyUrl] = useState("")
  const [isProcessing, setIsProcessing] = useState(false)
  const [isOpen, setIsOpen] = useState(false)

   useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedSearch(searchInput)
    }, 500)

    return () => clearTimeout(handler)
  }, [searchInput])
  
  const { data, isLoading, isError, isPlaceholderData } = useQuery({
    queryKey: ['songs', debouncedSearch],
    queryFn: () => fetchSongs(debouncedSearch),
    placeholderData: (previousData) => previousData, // Keeps old data visible while loading new results
  })

  async function sendUrlForProcess() {
    if (!spotifyUrl.startsWith("https://open.spotify.com/track/")) {
      toast.error("Invalid URL. Must start with https://open.spotify.com/track/")
      return
    }

    setIsProcessing(true)
    
    const processPromise = fetch('http://localhost:3333/v1/api/songs/import', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ url: spotifyUrl }),
    })

    toast.promise(processPromise, {
      loading: 'Pushing to Suzam Engine...',
      success: () => {
        setIsProcessing(false)
        setSpotifyUrl("")
        setIsOpen(false)
        // 3. Invalidate Query: This tells React Query the data is old 
        // and it will automatically re-fetch the table data
        queryClient.invalidateQueries({ queryKey: ['songs'] })
        return 'Song queued for processing!'
      },
      error: (err) => {
        setIsProcessing(false)
        return 'Failed to start process.'
      }
    })
  }

  const songs = data?.songs || []
  const totalCount = data?.total || 0

  return (
    <main className="page-wrap relative px-4 pb-8 pt-24 flex h-full min-h-screen flex-col gap-8 items-center">
      <h1 className="text-4xl font-bold mb-2 tracking-tighter italic text-primary uppercase">
        Songs in DB
      </h1>
      
      <div className="w-full max-w-6xl">
        <div className="w-full mb-4 flex items-center gap-2">
          <InputGroup>
            <InputGroupInput placeholder="Search indexed music..."  value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)} />
            <InputGroupAddon>
            {isLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Search />}
            </InputGroupAddon>
            <InputGroupAddon align="inline-end">
              {totalCount} songs
            </InputGroupAddon>
          </InputGroup>

          <Dialog open={isOpen} onOpenChange={setIsOpen}>
            <DialogTrigger asChild>
              <Button className="rounded-none border-2 border-primary/20 font-bold">
                <Plus className="mr-2 h-4 w-4" />
                Add New Song
              </Button>
            </DialogTrigger>
            <DialogContent className="rounded-none border-2 border-primary/20 bg-secondary">
              <DialogHeader>
                <DialogTitle className="text-2xl font-black italic tracking-tighter uppercase">
                  Import from Spotify
                </DialogTitle>
                <DialogDescription className="text-slate-400 text-xs uppercase tracking-widest font-bold">
                  Suzam Audio Pipeline v1.0
                </DialogDescription>
              </DialogHeader>
              <div className="py-4">
                <InputGroup>
                  <InputGroupInput  
                    placeholder="https://open.spotify.com/track/..." 
                    value={spotifyUrl}
                    disabled={isProcessing}
                    onChange={(e) => setSpotifyUrl(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") sendUrlForProcess()
                    }}
                  />
                  <InputGroupAddon><Link2 className="h-4 w-4" /></InputGroupAddon>
                </InputGroup>
                
                <Button 
                  onClick={sendUrlForProcess} 
                  disabled={isProcessing || !spotifyUrl}
                  className="w-full mt-4 uppercase tracking-widest"
                >
                  {isProcessing && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  {isProcessing ? "Processing..." : "Import Track"}
                </Button>
              </div>
            </DialogContent>
          </Dialog>
        </div>

        {/* 4. Handle Loading and Error States */}
        {isLoading ? (
          <div className="h-64 flex flex-col items-center justify-center border-2 border-dashed border-primary/10">
             <Loader2 className="h-10 w-10 animate-spin text-primary/40" />
             <p className="mt-4 text-[10px] uppercase font-black tracking-[0.3em] text-slate-500">Retrieving Database...</p>
          </div>
        ) : isError ? (
          <div className="h-64 flex items-center justify-center text-red-500 font-bold uppercase tracking-tighter">
            Error: Could not connect to Suzam Backend
          </div>
        ) : (
          <SongTable columns={columns} data={songs || []} />
        )}
      </div>
    </main>
  )
}