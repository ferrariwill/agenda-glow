import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type RefObject,
} from 'react'
import { createPortal } from 'react-dom'
import { formatTimeBR, minutesToTime, timeToMinutes } from '../../utils/format'

export const TIME_PICKER_POPOVER_LAYER = 'fixed z-[200]'

interface TimePickerProps {
  value: string
  onChange: (hhmm: string) => void
  stepMinutes?: number
  minTime?: string
  maxTime?: string
  className?: string
}

function buildTimeOptions(stepMinutes: number, minTime?: string, maxTime?: string): string[] {
  const step = Math.max(1, Math.min(stepMinutes, 60))
  const min = minTime ? timeToMinutes(minTime) : 0
  const max = maxTime ? timeToMinutes(maxTime) : 23 * 60 + 30
  const options: string[] = []
  for (let m = min; m <= max; m += step) {
    options.push(minutesToTime(m))
  }
  return options
}

export function TimePicker({
  value,
  onChange,
  stepMinutes = 30,
  minTime,
  maxTime,
  className = '',
}: TimePickerProps) {
  const listRef = useRef<HTMLDivElement>(null)
  const options = useMemo(
    () => buildTimeOptions(stepMinutes, minTime, maxTime),
    [stepMinutes, minTime, maxTime],
  )
  const selected = value ? formatTimeBR(value) : ''

  useEffect(() => {
    if (!selected || !listRef.current) return
    const el = listRef.current.querySelector<HTMLElement>(`[data-time="${selected}"]`)
    el?.scrollIntoView({ block: 'nearest' })
  }, [selected])

  return (
    <div
      className={[
        'max-w-[calc(100vw-2rem)] overflow-hidden rounded-xl border border-aura-border bg-white shadow-lg',
        className,
      ].join(' ')}
    >
      <div className="w-[min(100%,16rem)] p-2">
        <p className="mb-1 px-2 text-caption font-medium text-aura-muted">Horário</p>
        <div
          ref={listRef}
          role="listbox"
          aria-label="Selecionar horário"
          className="max-h-60 overflow-y-auto overscroll-contain"
        >
          {options.map((opt) => {
            const isSelected = opt === selected
            return (
              <button
                key={opt}
                type="button"
                role="option"
                data-time={opt}
                aria-selected={isSelected}
                aria-label={opt}
                onClick={() => onChange(opt)}
                className={[
                  'flex w-full min-h-touch-min touch-manipulation items-center rounded-lg px-3 text-left text-sm transition-colors',
                  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-aura-primary/40',
                  isSelected
                    ? 'bg-aura-primary font-semibold text-white'
                    : 'text-aura-anthracite hover:bg-aura-primary/10',
                ].join(' ')}
              >
                {opt}
              </button>
            )
          })}
        </div>
      </div>
    </div>
  )
}

interface TimePickerPopoverProps extends TimePickerProps {
  open: boolean
  onClose: () => void
  anchorRef: RefObject<HTMLElement | null>
}

type PopoverCoords = {
  top: number
  left: number
}

export function TimePickerPopover({
  open,
  onClose,
  anchorRef,
  value,
  onChange,
  stepMinutes,
  minTime,
  maxTime,
}: TimePickerPopoverProps) {
  const popRef = useRef<HTMLDivElement>(null)
  const [coords, setCoords] = useState<PopoverCoords>({ top: 0, left: 0 })

  const updatePosition = useCallback(() => {
    const anchor = anchorRef.current
    const pop = popRef.current
    if (!anchor || !pop) return

    const rect = anchor.getBoundingClientRect()
    const gap = 4
    const pad = 8
    const popWidth = pop.offsetWidth
    const popHeight = pop.offsetHeight

    let top = rect.bottom + gap
    let left = rect.left

    if (top + popHeight > window.innerHeight - pad) {
      const above = rect.top - popHeight - gap
      if (above >= pad) top = above
    }

    left = Math.max(pad, Math.min(left, window.innerWidth - popWidth - pad))
    setCoords({ top, left })
  }, [anchorRef])

  useLayoutEffect(() => {
    if (!open) return
    updatePosition()
  }, [open, value, updatePosition])

  useEffect(() => {
    if (!open) return

    const onScrollOrResize = () => updatePosition()
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    const onPointerDown = (e: MouseEvent) => {
      const t = e.target as Node
      if (popRef.current?.contains(t) || anchorRef.current?.contains(t)) return
      onClose()
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
  }, [open, onClose, anchorRef, updatePosition])

  if (!open) return null

  return createPortal(
    <div
      ref={popRef}
      className={TIME_PICKER_POPOVER_LAYER}
      style={{ top: coords.top, left: coords.left }}
      role="dialog"
      aria-modal="true"
      aria-label="Seletor de horário"
    >
      <TimePicker
        value={value}
        stepMinutes={stepMinutes}
        minTime={minTime}
        maxTime={maxTime}
        onChange={(hhmm) => {
          onChange(hhmm)
          onClose()
        }}
      />
    </div>,
    document.body,
  )
}
