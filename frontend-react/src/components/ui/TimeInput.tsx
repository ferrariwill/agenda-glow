import { useEffect, useRef, useState, type InputHTMLAttributes } from 'react'
import { Clock } from 'lucide-react'
import { formatTimeBR, parseTimeBR } from '../../utils/format'
import { Input } from './Input'
import { TimePickerPopover } from './TimePicker'

interface TimeInputProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'value' | 'onChange' | 'type'> {
  label?: string
  value: string
  onChange: (time: string) => void
  error?: string
  stepMinutes?: number
  minTime?: string
  maxTime?: string
  /** Exibe botão com seletor visual de horários. */
  withPicker?: boolean
}

export function TimeInput({
  label,
  value,
  onChange,
  error,
  className,
  stepMinutes = 30,
  minTime,
  maxTime,
  withPicker = true,
  ...props
}: TimeInputProps) {
  const [text, setText] = useState(formatTimeBR(value))
  const [focused, setFocused] = useState(false)
  const [localError, setLocalError] = useState('')
  const [pickerOpen, setPickerOpen] = useState(false)
  const wrapRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!focused) setText(formatTimeBR(value))
  }, [value, focused])

  const handleBlur = () => {
    setFocused(false)
    const parsed = parseTimeBR(text)
    if (parsed) {
      setLocalError('')
      onChange(parsed)
      setText(formatTimeBR(parsed))
    } else if (text.trim()) {
      setLocalError('Use o formato HH:mm (24h)')
      setText(formatTimeBR(value))
    } else {
      setLocalError('')
      setText(formatTimeBR(value))
    }
  }

  return (
    <div ref={wrapRef} className="relative space-y-1.5">
      {label && (
        <label className="block text-body font-medium text-aura-anthracite">{label}</label>
      )}
      <div className="flex items-start gap-2">
        <Input
          {...props}
          label={undefined}
          value={text}
          placeholder="09:00"
          inputMode="numeric"
          autoComplete="off"
          className={className}
          error={error ?? localError}
          onFocus={(e) => {
            setFocused(true)
            setLocalError('')
            props.onFocus?.(e)
          }}
          onChange={(e) => {
            const raw = e.target.value.replace(/[^\d:]/g, '').slice(0, 5)
            setText(raw)
          }}
          onBlur={(e) => {
            handleBlur()
            props.onBlur?.(e)
          }}
        />
        {withPicker && (
          <button
            type="button"
            onClick={() => setPickerOpen((o) => !o)}
            className={[
              'mt-0 flex min-h-touch-min min-w-touch-min shrink-0 touch-manipulation items-center justify-center rounded-lg border transition-colors',
              'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-aura-primary/40',
              pickerOpen
                ? 'border-aura-primary bg-aura-primary/10 text-aura-primary'
                : 'border-aura-border bg-white text-aura-muted hover:border-aura-primary hover:text-aura-primary',
            ].join(' ')}
            aria-label="Abrir seletor de horário"
            aria-expanded={pickerOpen}
          >
            <Clock className="h-4 w-4" />
          </button>
        )}
      </div>
      {withPicker && (
        <TimePickerPopover
          open={pickerOpen}
          onClose={() => setPickerOpen(false)}
          anchorRef={wrapRef}
          value={value}
          stepMinutes={stepMinutes}
          minTime={minTime}
          maxTime={maxTime}
          onChange={onChange}
        />
      )}
    </div>
  )
}
