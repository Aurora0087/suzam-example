import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
} from '@/components/ui/drawer'
import { Button } from '../ui/button'
import { useCallback, useEffect, useRef, useState } from 'react'

interface SelectedSong {
  id: string
  title: string
}

interface SongTableProps<TData, TValue> {
  columns: ColumnDef<TData, TValue>[]
  data: TData[]
}

const API = 'http://localhost:3333'
const MIN_SCALE = 0.25
const MAX_SCALE = 20

interface Transform {
  x: number
  y: number
  scale: number
}

function SyncImageViewer({ images }: { images: { src: string; label: string }[] }) {
  const [transform, setTransform] = useState<Transform>({ x: 0, y: 0, scale: 1 })
  const [missing, setMissing] = useState<Record<number, boolean>>({})
  const dragging = useRef(false)
  const last = useRef({ x: 0, y: 0 })
  const containerRef = useRef<HTMLDivElement>(null)

  // reset when images change (new song opened)
  useEffect(() => {
    setTransform({ x: 0, y: 0, scale: 0.5 })
    setMissing({})
  }, [images[0]?.src])

  const onMouseDown = useCallback((e: React.MouseEvent) => {
    if (e.button !== 0) return
    dragging.current = true
    last.current = { x: e.clientX, y: e.clientY }
    e.preventDefault()
    e.stopPropagation()
  }, [])

  const onMouseMove = useCallback((e: React.MouseEvent) => {
    if (!dragging.current) return
    e.preventDefault();
    const dx = e.clientX - last.current.x
    const dy = e.clientY - last.current.y
    last.current = { x: e.clientX, y: e.clientY }
    setTransform(t => ({ ...t, x: t.x + dx, y: t.y + dy }))
  }, [])

  const stopDrag = useCallback(() => { dragging.current = false }, [])

  const onWheel = useCallback((e: React.WheelEvent) => {
    e.preventDefault()
    const rect = containerRef.current?.getBoundingClientRect()
    if (!rect) return
    // cursor position relative to container
    const cx = e.clientX - rect.left
    const cy = e.clientY - rect.top
    setTransform(t => {
      const factor = e.deltaY < 0 ? 1.1 : 0.9
      const next = Math.min(MAX_SCALE, Math.max(MIN_SCALE, t.scale * factor))
      const ratio = next / t.scale
      return {
        scale: next,
        x: cx - (cx - t.x) * ratio,
        y: cy - (cy - t.y) * ratio,
      }
    })
  }, [])

  const style = {
    transform: `translate(${transform.x}px, ${transform.y}px) scale(${transform.scale})`,
    transformOrigin: '0 0',
  }

  return (
    <div
      ref={containerRef}
      className="flex flex-col gap-0 select-none overflow-hidden border border-primary/10 bg-black/20"
      style={{ cursor: dragging.current ? 'grabbing' : 'grab' }}
      onPointerDown={e => e.stopPropagation()}
      onMouseDown={onMouseDown}
      onMouseMove={onMouseMove}
      onMouseUp={stopDrag}
      onMouseLeave={stopDrag}
      onWheel={onWheel}
    >
      {images.map((img, i) => (
        <div key={img.src} className="flex flex-col">
          <span className="text-[10px] uppercase tracking-widest text-muted-foreground font-bold px-2 py-1 bg-background/60 border-b border-primary/10 shrink-0">
            {img.label}
          </span>
          <div className="overflow-hidden" style={{ height: 180 }}>
            {missing[i] ? (
              <div className="flex items-center justify-center h-full text-muted-foreground text-xs border-b border-dashed border-primary/20">
                Not available
              </div>
            ) : (
              <div style={style}>
                <img
                  src={img.src}
                  alt={img.label}
                  draggable={false}
                  onError={() => setMissing(m => ({ ...m, [i]: true }))}
                  style={{ display: 'block', maxWidth: 'none' }}
                />
              </div>
            )}
          </div>
        </div>
      ))}
      <p className="text-[10px] text-center text-muted-foreground/50 py-1">
        scroll to zoom · drag to pan
      </p>
    </div>
  )
}

export function SongTable<TData, TValue>({
  columns,
  data,
}: SongTableProps<TData, TValue>) {
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [selected, setSelected] = useState<SelectedSong>({ id: '', title: '' })

  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
  })

  return (
    <div className="relative w-full mx-auto mt-4">
      <Drawer  open={drawerOpen} onOpenChange={setDrawerOpen}>
        <DrawerContent >
          <DrawerHeader>
            <DrawerTitle>{selected.title || `Song #${selected.id}`}</DrawerTitle>
            <DrawerDescription>
              Spectrogram and Constellation Map — scroll to zoom, drag to pan
            </DrawerDescription>
          </DrawerHeader>
          <div className="px-4 pb-2">
            <SyncImageViewer
              images={[
                { src: `${API}/images/${selected.id}/spectrogram.png`, label: 'Spectrogram' },
                { src: `${API}/images/${selected.id}/peaks.png`,       label: 'Constellation Map' },
              ]}
            />
          </div>
          <DrawerFooter>
            <DrawerClose>
              <Button variant="outline">Close</Button>
            </DrawerClose>
          </DrawerFooter>
        </DrawerContent>
      </Drawer>
      <div className="absolute inset-0 z-0 translate-x-2 translate-y-2 border-2 border-primary/20 stripe-gray-bg pointer-events-none" />
      <div className="relative z-10 overflow-hidden bg-secondary border-2">
        <Table>
          <TableHeader className="bg-secondary">
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow
                key={headerGroup.id}
                className="border-b-2 border-primary/10 hover:bg-transparent"
              >
                {headerGroup.headers.map((header) => {
                  return (
                    <TableHead
                      key={header.id}
                      className="h-10 px-4 text-[10px] uppercase tracking-[0.25em] font-black text-secondary-foreground"
                    >
                      {header.isPlaceholder
                        ? null
                        : flexRender(
                            header.column.columnDef.header,
                            header.getContext(),
                          )}
                    </TableHead>
                  )
                })}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows?.length ? (
              table.getRowModel().rows.map((row) => (
                <TableRow
                  key={row.id}
                  data-state={row.getIsSelected() && 'selected'}
                  className="group border-b border-primary/50 hover:bg-card/5 bg-card transition-colors cursor-pointer"
                  onClick={() => {
                    setDrawerOpen(true)
                    setSelected({ id: String(row.getValue('id')), title: row.getValue('title') })
                  }}
                >
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      {flexRender(
                        cell.column.columnDef.cell,
                        cell.getContext(),
                      )}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className="h-24 text-center text-xs uppercase tracking-widest text-slate-600 italic font-bold"
                >
                  No results.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}
