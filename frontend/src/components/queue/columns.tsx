import { type ColumnDef } from '@tanstack/react-table'
import { Clock, Download, Cpu, CheckCircle2, AlertCircle } from 'lucide-react'
import { FaSpotify } from 'react-icons/fa'

export type QueueItem = {
  id: number
  spotify_id: string
  song_name: string
  authors: string
  status: 'pending' | 'downloading' | 'fingerprinting' | 'completed' | 'failed'
  err_message: { String: string; Valid: boolean } | string | null
  added: string
  completed: { String: string; Valid: boolean } | string | null
}

export const queueColumns: ColumnDef<QueueItem>[] = [
  {
    accessorKey: 'status',
    header: 'Engine Status',
    cell: ({ row }) => {
      const status = row.getValue('status') as QueueItem['status']

      const statusConfig = {
        pending: { icon: Clock, color: 'text-slate-500', label: 'Queued' },
        downloading: {
          icon: Download,
          color: 'text-blue-500 animate-bounce',
          label: 'Downloading',
        },
        fingerprinting: {
          icon: Cpu,
          color: 'text-purple-500 animate-pulse',
          label: 'Analyzing',
        },
        completed: {
          icon: CheckCircle2,
          color: 'text-emerald-500',
          label: 'Indexed',
        },
        failed: { icon: AlertCircle, color: 'text-red-500', label: 'Failed' },
      }

      const config = statusConfig[status]
      const Icon = config.icon

      return (
        <div className="flex items-center gap-2">
          <Icon className={`h-4 w-4 ${config.color}`} />
          <span
            className={`text-[10px] uppercase font-black tracking-widest ${config.color}`}
          >
            {config.label}
          </span>
        </div>
      )
    },
  },
  {
    accessorKey: 'song_name',
    header: 'Track Info',
    cell: ({ row }) => {
      const name = row.getValue('song_name') as string
      const authors = row.original.authors
      const status = row.original.status
      const errMsg =
        typeof row.original.err_message === 'object'
          ? row.original.err_message?.String
          : row.original.err_message

      return (
        <div className="flex flex-col">
          <div className="flex items-center gap-2">
            <span className="font-bold tracking-tight text-card-foreground leading-tight">
              {name || 'Fetching metadata...'}
            </span>
          </div>
          <span className="text-[10px] text-card-foreground/50 uppercase font-medium">
            {authors || 'Unknown Artist'}
          </span>
          {status === 'failed' && errMsg && (
            <span className="text-[9px] text-red-400 mt-1 bg-red-400/10 px-1 border-l border-red-400 italic">
              Error: {errMsg}
            </span>
          )}
        </div>
      )
    },
  },
  {
    accessorKey: 'added',
    header: 'Started',
    cell: ({ row }) => {
      const date = new Date(row.getValue('added'))
      return (
        <span className="font-mono text-[11px] text-card-foreground/50">
          {date.toLocaleTimeString([], {
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
            hour12: false,
          })}
        </span>
      )
    },
  },
  {
    accessorKey: 'spotify_id',
    header: 'Source',
    cell: ({ row }) => (
      <div className="flex items-center gap-2">
        <a
          href={`https://open.spotify.com/track/${row.getValue('spotify_id')}`}
          target="_blank"
          rel="noreferrer"
          className="flex h-7 w-7 items-center justify-center rounded-none border border-primary/20 bg-secondary text-secondary-foreground hover:bg-primary hover:text-primary-foreground transition-colors"
        >
          <FaSpotify className="h-3.5 w-3.5" />
        </a>
      </div>
    ),
  },
]
