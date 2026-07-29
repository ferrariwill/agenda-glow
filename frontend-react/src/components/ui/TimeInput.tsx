import { useEffect, useState, type InputHTMLAttributes } from 'react'
import { formatTimeBR, parseTimeBR } from '../../utils/format'
import { Input } from './Input'

interface TimeInputProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'value' | 'onChange' | 'type'> {
  label?: string
  value: string
  onChange: (time: string) => void
  error?: string
}

export function TimeInput({ label, value, onChange, error, className, ...props }: TimeInputProps) {
  const [text, setText] = useState(formatTimeBR(value))
  const [focused, setFocused] = useState(false)
  const [localError, setLocalError] = useState('')

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
    <Input
      {...props}
      label={label}
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
  )
}
