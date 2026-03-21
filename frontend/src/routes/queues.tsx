import { createFileRoute } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { QueueTable } from '@/components/queue/QueueTable'
import { Loader2, RefreshCcw, AlertTriangle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { queueColumns, type QueueItem } from '@/components/queue/columns'

export const Route = createFileRoute('/queues')({
  component: RouteComponent,
})


const fetchQueues = async (): Promise<QueueItem[]> => {
  const response = await fetch('http://localhost:3333/v1/api/songs/queues?limit=50&status=all')
  if (!response.ok) {
    throw new Error('Failed to fetch queue data')
  }
  return response.json()
}

function RouteComponent() {
  const { data, isLoading, isError, refetch, isRefetching } = useQuery({
    queryKey: ['queues'],
    queryFn: fetchQueues,
    refetchInterval: 3000, //3sec
  })

  return (
    <main className="page-wrap relative px-4 pb-8 pt-24 flex h-full min-h-screen flex-col gap-8 items-center">
      <div className="text-center space-y-2">
        <h1 className="text-4xl font-bold tracking-tighter italic text-primary uppercase">
          Added music Queue
        </h1>
        <p className="text-xs text-slate-500 uppercase tracking-[0.3em] font-bold">
          Real-time Engine Status
        </p>
      </div>

      <div className="w-full max-w-6xl">
        <div className="flex justify-end mb-4">
          <Button 
            variant="ghost" 
            size="sm" 
            onClick={() => refetch()}
            disabled={isLoading || isRefetching}
            className="text-[10px] uppercase tracking-widest font-black"
          >
            {isRefetching ? (
              <Loader2 className="mr-2 h-3 w-3 animate-spin" />
            ) : (
              <RefreshCcw className="mr-2 h-3 w-3" />
            )}
            Force Refresh
          </Button>
        </div>

        {isLoading ? (
          <div className="h-64 flex flex-col items-center justify-center space-y-4 border-2 border-dashed border-primary/10 bg-secondary/20">
            <Loader2 className="h-8 w-8 animate-spin text-primary" />
            <p className="text-[10px] uppercase font-black tracking-widest text-slate-500">
              Synchronizing with Suzam Core...
            </p>
          </div>
        ) : isError ? (
          <div className="h-64 flex flex-col items-center justify-center space-y-4 border-2 border-destructive/20 bg-destructive/5">
            <AlertTriangle className="h-8 w-8 text-destructive" />
            <p className="text-sm font-bold text-destructive uppercase tracking-tight">
              Connection to Backend Severed
            </p>
            <Button onClick={() => refetch()} variant="outline" size="sm">Retry Connection</Button>
          </div>
        ) : (
          <QueueTable columns={queueColumns} data={data || []} />
        )}
      </div>
    </main>
  )
}