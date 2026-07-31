import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from 'react'
import { createPortal } from 'react-dom'
import type { LucideIcon } from 'lucide-react'

export type ActionDropdownItem = {
  label: string
  icon?: LucideIcon
  onClick: () => void
  disabled?: boolean
  danger?: boolean
}

interface ActionsDropdownProps {
  items: ActionDropdownItem[]
  icon: LucideIcon
  ariaLabel?: string
  triggerClassName?: string
  menuClassName?: string
  align?: 'left' | 'right'
  minWidth?: number
  iconClassName?: string
}

type MenuCoords = {
  top: number
  left: number
  maxHeight: number
}

const TRIGGER_BASE =
  'inline-flex min-h-touch-min min-w-touch-min touch-manipulation items-center justify-center focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-aura-primary/40'

export function ActionsDropdown({
  items,
  icon: Icon,
  ariaLabel = 'Ações',
  triggerClassName = 'rounded-lg text-aura-muted hover:bg-aura-surface',
  menuClassName = '',
  align = 'right',
  minWidth = 176,
  iconClassName = 'h-4 w-4',
}: ActionsDropdownProps) {
  const [open, setOpen] = useState(false)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const [coords, setCoords] = useState<MenuCoords>({ top: 0, left: 0, maxHeight: 320 })

  const updatePosition = useCallback(() => {
    const anchor = triggerRef.current
    const menu = menuRef.current
    if (!anchor || !menu) return

    const rect = anchor.getBoundingClientRect()
    const gap = 4
    const pad = 8
    const menuWidth = Math.max(menu.offsetWidth, minWidth)
    const menuHeight = menu.offsetHeight

    let top = rect.bottom + gap
    let left = align === 'right' ? rect.right - menuWidth : rect.left

    if (top + menuHeight > window.innerHeight - pad) {
      const above = rect.top - menuHeight - gap
      if (above >= pad) top = above
    }

    left = Math.max(pad, Math.min(left, window.innerWidth - menuWidth - pad))
    const maxHeight = Math.max(120, window.innerHeight - top - pad)

    setCoords({ top, left, maxHeight })
  }, [align, minWidth])

  useLayoutEffect(() => {
    if (!open) return
    updatePosition()
  }, [open, items, updatePosition])

  useEffect(() => {
    if (!open) return

    const onScrollOrResize = () => updatePosition()
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    const onPointerDown = (e: MouseEvent) => {
      const target = e.target as Node
      if (triggerRef.current?.contains(target) || menuRef.current?.contains(target)) return
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

  const menu =
    open &&
    createPortal(
      <div
        ref={menuRef}
        role="menu"
        className={[
          'fixed z-[200] overflow-y-auto rounded-lg border border-aura-border bg-white py-1 shadow-lg',
          menuClassName,
        ]
          .filter(Boolean)
          .join(' ')}
        style={{
          top: coords.top,
          left: coords.left,
          minWidth,
          maxHeight: coords.maxHeight,
        }}
      >
        {items.map(({ label, icon: ItemIcon, onClick, disabled, danger }) => (
          <button
            key={label}
            type="button"
            role="menuitem"
            disabled={disabled}
            onClick={() => {
              if (disabled) return
              onClick()
              setOpen(false)
            }}
            className={[
              'flex min-h-touch-min w-full touch-manipulation items-center gap-2 px-3 text-left text-sm',
              'focus-visible:outline-none focus-visible:bg-aura-surface focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-aura-primary/40',
              disabled
                ? 'cursor-not-allowed text-aura-muted/50'
                : danger
                  ? 'text-red-600 hover:bg-red-50'
                  : 'text-aura-anthracite hover:bg-aura-surface',
            ].join(' ')}
          >
            {ItemIcon && <ItemIcon className="h-4 w-4 shrink-0 text-aura-muted" />}
            {label}
          </button>
        ))}
      </div>,
      document.body,
    )

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        onClick={() => setOpen((v) => !v)}
        className={[TRIGGER_BASE, triggerClassName].filter(Boolean).join(' ')}
        aria-label={ariaLabel}
        aria-expanded={open}
        aria-haspopup="menu"
      >
        <Icon className={iconClassName} />
      </button>
      {menu}
    </>
  )
}
