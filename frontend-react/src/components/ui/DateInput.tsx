import { useEffect, useRef, useState, type InputHTMLAttributes } from 'react'
import { Calendar } from 'lucide-react'
import { formatDateBR, maskDateBRInput, parseDateBR } from '../../utils/format'
import { DatePickerPopover } from './DatePicker'
import { Input } from './Input'

interface DateInputProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'value' | 'onChange' | 'type'> {
  label?: string
  value: string
  onChange: (iso: string) => void
  error?: string
  minDate?: string
  maxDate?: string
  /** Exibe botão com calendário visual (datepicker). */
  withPicker?: boolean
}

export function DateInput({
  label,
  value,
  onChange,
  error,
  className,
  minDate,
  maxDate,
  withPicker = true,
  ...props
}: DateInputProps) {
  const [text, setText] = useState(formatDateBR(value))
  const [focused, setFocused] = useState(false)
  const [localError, setLocalError] = useState('')
  const [pickerOpen, setPickerOpen] = useState(false)
  const wrapRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!focused) setText(formatDateBR(value))
  }, [value, focused])

  const handleBlur = () => {
    setFocused(false)
    const parsed = parseDateBR(text)
    if (parsed) {
      setLocalError('')
      onChange(parsed)
      setText(formatDateBR(parsed))
    } else if (text.trim()) {
      setLocalError('Use o formato DD/MM/AAAA')
      setText(formatDateBR(value))
    } else {
      setLocalError('')
      setText(formatDateBR(value))
    }
  }

  return (
    <div ref={wrapRef} className="relative space-y-1.5">
      {label && (
        <label className="block text-sm font-medium text-aura-anthracite">{label}</label>
      )}
      <div className="flex items-start gap-2">
        <Input
          {...props}
          label={undefined}
          value={text}
          placeholder="DD/MM/AAAA"
          inputMode="numeric"
          autoComplete="off"
          className={className}
          error={error ?? localError}
          onFocus={(e) => {
            setFocused(true)
            setLocalError('')
            props.onFocus?.(e)
          }}
          onChange={(e) => setText(maskDateBRInput(e.target.value))}
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
              'mt-0 flex h-[42px] w-[42px] shrink-0 items-center justify-center rounded-lg border transition-colors',
              pickerOpen
                ? 'border-aura-primary bg-aura-primary/10 text-aura-primary'
                : 'border-aura-border bg-white text-aura-muted hover:border-aura-primary hover:text-aura-primary',
            ].join(' ')}
            aria-label="Abrir calendário"
          >
            <Calendar className="h-4 w-4" />
          </button>
        )}
      </div>
      {withPicker && (
        <DatePickerPopover
          open={pickerOpen}
          onClose={() => setPickerOpen(false)}
          anchorRef={wrapRef}
          value={value}
          minDate={minDate}
          maxDate={maxDate}
          onChange={onChange}
        />
      )}
    </div>
  )
}
