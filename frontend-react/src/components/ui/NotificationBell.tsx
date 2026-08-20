import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from 'react'
import { createPortal } from 'react-dom'
import { Bell } from 'lucide-react'
import { formatRelativeShort } from '../../utils/format'

export type NotificationItem = {
  id: string
  title: string
  body?: string
  created_at: string
  read_at: string | null
}

interface NotificationBellProps {
  items: NotificationItem[]
  unreadCount: number
  loading?: boolean
  onOpen?: () => void
  onMarkRead?: (id: string) => void
}

type PanelCoords = {
  top: number
  left: number
  maxHeight: number
}

const PANEL_WIDTH = 320

function badgeLabel(count: number) {
  if (count <= 0) return null
  return count > 9 ? '9+' : String(count)
}

export function NotificationBell({
  items,
  unreadCount,
  loading = false,
  onOpen,
  onMarkRead,
}: NotificationBellProps) {
  const [open, setOpen] = useState(false)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const panelRef = useRef<HTMLDivElement>(null)
  const [coords, setCoords] = useState<PanelCoords>({ top: 0, left: 0, maxHeight: 360 })

  const ariaLabel =
    unreadCount > 0
      ? `${unreadCount} notificaç${unreadCount === 1 ? 'ão' : 'ões'} não lida${unreadCount === 1 ? '' : 's'}`
      : 'Notificações'

  const updatePosition = useCallback(() => {
    const anchor = triggerRef.current
    const panel = panelRef.current
    if (!anchor || !panel) return

    const rect = anchor.getBoundingClientRect()
    const gap = 4
    const pad = 8
    const panelWidth = Math.max(panel.offsetWidth, PANEL_WIDTH)
    const panelHeight = panel.offsetHeight

    let top = rect.bottom + gap
    let left = rect.right - panelWidth

    if (top + panelHeight > window.innerHeight - pad) {
      const above = rect.top - panelHeight - gap
      if (above >= pad) top = above
    }

    left = Math.max(pad, Math.min(left, window.innerWidth - panelWidth - pad))
    const maxHeight = Math.max(160, window.innerHeight - top - pad)

    setCoords({ top, left, maxHeight })
  }, [])

  useLayoutEffect(() => {
    if (!open) return
    updatePosition()
  }, [open, items, loading, updatePosition])

  useEffect(() => {
    if (!open) return

    const onScrollOrResize = () => updatePosition()
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    const onPointerDown = (e: MouseEvent) => {
      const target = e.target as Node
      if (triggerRef.current?.contains(target) || panelRef.current?.contains(target)) return
      setOpen(false)
    }

    window.addEventListener('scroll', onScrollOrResize, true)
    window.addEventListener('resize', onScrollOrResize)
    document.addEventListener('keydown', onKeyDown)
    document.addEventListener('mousedown', onPointerDown)

    return () => {
      window.removeEventListener('scroll', onScrollOrResize, true)
      window.removeEventListener('resize', onScrollOrResize)
      document.removeEventListener('keydown', onKeyDown)
      document.removeEventListener('mousedown', onPointerDown)
    }
  }, [open, updatePosition])

  const badge = badgeLabel(unreadCount)

  const panel =
    open &&
    createPortal(
      <div
        ref={panelRef}
        role="dialog"
        aria-label="Notificações"
        className="fixed z-[200] overflow-y-auto rounded-lg border border-aura-border bg-white shadow-lg"
        style={{
          top: coords.top,
          left: coords.left,
          width: PANEL_WIDTH,
          maxWidth: `calc(100vw - 16px)`,
          maxHeight: coords.maxHeight,
        }}
      >
        <div className="border-b border-aura-border px-3 py-2.5">
          <p className="text-sm font-semibold text-aura-anthracite">Notificações</p>
        </div>
        {loading ? (
          <p className="px-3 py-6 text-center text-sm text-aura-muted">Carregando…</p>
        ) : items.length === 0 ? (
          <p className="px-3 py-6 text-center text-sm text-aura-muted">Sem notificações</p>
        ) : (
          <ul className="divide-y divide-aura-border py-1">
            {items.map((item) => {
              const unread = !item.read_at
              return (
                <li key={item.id}>
                  <button
                    type="button"
                    className={[
                      'flex min-h-touch-min w-full touch-manipulation flex-col gap-0.5 px-3 py-2.5 text-left',
                      'hover:bg-aura-surface focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-aura-primary/40',
                      unread ? 'bg-aura-primary/5' : '',
                    ].join(' ')}
                    onClick={() => onMarkRead?.(item.id)}
                  >
                    <span
                      className={[
                        'text-sm text-aura-anthracite',
                        unread ? 'font-semibold' : 'font-medium',
                      ].join(' ')}
                    >
                      {item.title}
                    </span>
                    {item.body && (
                      <span className="line-clamp-2 text-xs text-aura-muted">{item.body}</span>
                    )}
                    <span className="text-xs text-aura-muted">
                      {formatRelativeShort(item.created_at)}
                    </span>
                  </button>
                </li>
              )
            })}
          </ul>
        )}
      </div>,
      document.body,
    )

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        onClick={() => {
          setOpen((v) => {
            const next = !v
            if (next) onOpen?.()
            return next
          })
        }}
        className="relative inline-flex min-h-touch-min min-w-touch-min touch-manipulation items-center justify-center rounded-lg p-2 text-aura-muted hover:bg-aura-surface focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-aura-primary/40"
        aria-label={ariaLabel}
        aria-expanded={open}
        aria-haspopup="dialog"
      >
        <Bell className="h-5 w-5" />
        {badge && (
          <span
            aria-hidden="true"
            className="absolute right-1 top-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-semibold leading-none text-white"
          >
            {badge}
          </span>
        )}
      </button>
      {panel}
    </>
  )
}
