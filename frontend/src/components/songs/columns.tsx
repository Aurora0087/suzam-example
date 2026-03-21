import { type ColumnDef } from "@tanstack/react-table"
import { Music } from "lucide-react"
import { FaSpotify } from "react-icons/fa";

export type Song = {
  id: string
  title: string
  artists: string[]
  time: number
  spotify_id: string
}

export const columns: ColumnDef<Song>[] = [
  {
    accessorKey: "title",
    header: "Title",
    cell: ({ row }) => (
      <div className="flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center bg-primary/10 border border-primary/20">
          <Music className="h-5 w-5 text-primary" />
        </div>
        <span className="font-bold tracking-tight">{row.getValue("title")}</span>
      </div>
    ),
  },
  {
    accessorKey: "artists",
    header: "Artists",
    cell: ({ row }) => {
      const artists = row.getValue("artists") as string[]
      return <span className="text-slate-400 font-medium">{artists.join(", ")}</span>
    },
  },
  {
    accessorKey: "time",
    header: "Duration",
    cell: ({ row }) => {
      const ms = row.getValue("time") as number
      const minutes = Math.floor(ms / 60000)
      const seconds = ((ms % 60000) / 1000).toFixed(0)
      return <span className="font-mono text-slate-500">{minutes}:{Number(seconds) < 10 ? "0" : ""}{seconds}</span>
    },
  },
  {
    accessorKey: "spotify_id",
    header: "Link",
    cell: ({ row }) => (
      <a 
        href={`https://open.spotify.com/track/${row.getValue("spotify_id")}`}
        target="_blank"
        rel="noreferrer"
        title="Play song on Spotify"
        className="flex h-8 w-8 items-center justify-center rounded-none border border-primary/20 bg-secondary text-secondary-foreground hover:bg-primary hover:text-primary-foreground transition-colors"
      >
        <FaSpotify className="h-4 w-4" />
      </a>
    ),
  },
]