import { useEffect, useRef, useState } from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import {
  MONTH_NAMES_PT,
  WEEKDAYS_SHORT_PT,
  formatDateBR,
  getCalendarGrid,
  parseISOParts,
  todayISO,
} from '../../utils/format'

interface DatePickerProps {
  value: string
  onChange: (iso: string) => void
  minDate?: string
  maxDate?: string
  className?: string
}

export function DatePicker({ value, onChange, minDate, maxDate, className = '' }: DatePickerProps) {
  const parts = value ? parseISOParts(value) : parseISOParts(todayISO())
  const [viewYear, setViewYear] = useState(parts.year)
  const [viewMonth, setViewMonth] = useState(parts.month)

  useEffect(() => {
    if (!value) return
    const p = parseISOParts(value)
    setViewYear(p.year)
    setViewMonth(p.month)
  }, [value])

  const shiftMonth = (delta: number) => {
    const d = new Date(viewYear, viewMonth - 1 + delta, 1)
    setViewYear(d.getFullYear())
    setViewMonth(d.getMonth() + 1)
  }

  const isDisabled = (iso: string) => {
    if (minDate && iso < minDate) return true
    if (maxDate && iso > maxDate) return true
    return false
  }

  const grid = getCalendarGrid(viewYear, viewMonth)

  return (
    <div
      className={[
        /* Viewport 360: shell cabe na tela; grid interno não comprime abaixo de 44×44 */
        'max-w-[calc(100vw-2rem)] overflow-x-auto rounded-xl border border-aura-border bg-white shadow-lg',
        className,
      ].join(' ')}
    >
      <div className="box-border w-[382px] p-3">
        <div className="mb-3 flex items-center justify-between">
          <button
            type="button"
            onClick={() => shiftMonth(-1)}
            className="inline-flex min-h-touch-min min-w-touch-min touch-manipulation items-center justify-center rounded-lg text-aura-muted hover:bg-aura-surface hover:text-aura-anthracite focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-aura-primary/40"
            aria-label="Mês anterior"
          >
            <ChevronLeft className="h-4 w-4" />
          </button>
          <p className="text-body font-semibold text-aura-anthracite">
            {MONTH_NAMES_PT[viewMonth - 1]} {viewYear}
          </p>
          <button
            type="button"
            onClick={() => shiftMonth(1)}
            className="inline-flex min-h-touch-min min-w-touch-min touch-manipulation items-center justify-center rounded-lg text-aura-muted hover:bg-aura-surface hover:text-aura-anthracite focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-aura-primary/40"
            aria-label="Próximo mês"
          >
            <ChevronRight className="h-4 w-4" />
          </button>
        </div>

        <div className="mb-1 grid grid-cols-7 gap-2 text-center text-caption font-medium text-aura-muted">
          {WEEKDAYS_SHORT_PT.map((wd) => (
            <span key={wd} className="py-1">
              {wd}
            </span>
          ))}
        </div>

        {/* 7×44 + 6×8 gap = 356px úteis dentro de p-3 → painel 382px */}
        <div className="grid grid-cols-7 gap-2">
          {grid.map((cell, i) => {
            if (!cell.inMonth || !cell.iso) {
              return <span key={`e-${i}`} className="min-h-touch-min min-w-touch-min" aria-hidden />
            }
            const selected = cell.iso === value
            const disabled = isDisabled(cell.iso)
            return (
              <button
                key={cell.iso}
                type="button"
                disabled={disabled}
                onClick={() => onChange(cell.iso!)}
                className={[
                  'box-border flex size-touch-min shrink-0 touch-manipulation items-center justify-center rounded-lg text-sm transition-colors',
                  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-aura-primary/40',
                  selected
                    ? 'bg-aura-primary font-semibold text-white'
                    : cell.isToday
                      ? 'font-semibold text-aura-primary ring-1 ring-aura-primary/40'
                      : 'text-aura-anthracite hover:bg-aura-primary/10',
                  disabled ? 'cursor-not-allowed opacity-30' : '',
                ].join(' ')}
              >
                {cell.day}
              </button>
            )
          })}
        </div>

        <div className="mt-3 flex items-center justify-between border-t border-aura-border pt-2">
          <button
            type="button"
            onClick={() => onChange(todayISO())}
            className="min-h-touch-min touch-manipulation px-1 text-caption font-medium text-aura-primary hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-aura-primary/40"
          >
            Hoje
          </button>
          <span className="text-caption text-aura-muted">{formatDateBR(value)}</span>
        </div>
      </div>
    </div>
  )
}

interface DatePickerPopoverProps extends DatePickerProps {
  open: boolean
  onClose: () => void
  anchorRef: React.RefObject<HTMLElement | null>
}

export function DatePickerPopover({
  open,
  onClose,
  anchorRef,
  value,
  onChange,
  minDate,
  maxDate,
}: DatePickerPopoverProps) {
  const popRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const close = (e: MouseEvent) => {
      const t = e.target as Node
      if (popRef.current?.contains(t) || anchorRef.current?.contains(t)) return
      onClose()
    }
    document.addEventListener('mousedown', close)
    return () => document.removeEventListener('mousedown', close)
  }, [open, onClose, anchorRef])

  if (!open) return null

  return (
    <div ref={popRef} className="absolute left-0 top-full z-50 mt-1">
      <DatePicker
        value={value}
        minDate={minDate}
        maxDate={maxDate}
        onChange={(iso) => {
          onChange(iso)
          onClose()
        }}
      />
    </div>
  )
}
