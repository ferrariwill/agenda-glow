import { forwardRef, type InputHTMLAttributes } from 'react'
import { maskBRLInput } from '../../utils/format'
import { Input } from './Input'

interface MoneyInputProps
  extends Omit<InputHTMLAttributes<HTMLInputElement>, 'value' | 'onChange' | 'type'> {
  label?: string
  value: string
  onChange: (masked: string) => void
  error?: string
  /** Prefixo visual "R$" (default true). */
  showPrefix?: boolean
  /**
   * Input cru (sem wrapper `Input` do design system) — útil nos forms glass da Dona
   * que já têm label/estilo próprio.
   */
  bare?: boolean
}

export const MoneyInput = forwardRef<HTMLInputElement, MoneyInputProps>(
  (
    {
      label,
      value,
      onChange,
      error,
      showPrefix = true,
      bare = false,
      className = '',
      placeholder = '0,00',
      ...props
    },
    ref,
  ) => {
    const handleChange = (raw: string) => {
      onChange(maskBRLInput(raw))
    }

    if (bare) {
      return (
        <div className="relative">
          {showPrefix && (
            <span className="pointer-events-none absolute left-0 top-1/2 -translate-y-1/2 text-[#514440]">
              R$
            </span>
          )}
          <input
            ref={ref}
            type="text"
            inputMode="decimal"
            autoComplete="off"
            placeholder={placeholder}
            {...props}
            value={value}
            onChange={(e) => handleChange(e.target.value)}
            className={[showPrefix ? 'pl-8' : '', className].filter(Boolean).join(' ')}
          />
        </div>
      )
    }

    return (
      <div className="space-y-1.5">
        {label && (
          <label className="block text-body font-medium text-aura-anthracite">{label}</label>
        )}
        <div className="relative">
          {showPrefix && (
            <span className="pointer-events-none absolute left-3 top-1/2 z-10 -translate-y-1/2 text-sm text-aura-muted">
              R$
            </span>
          )}
          <Input
            ref={ref}
            error={error}
            type="text"
            inputMode="decimal"
            autoComplete="off"
            placeholder={placeholder}
            {...props}
            value={value}
            onChange={(e) => handleChange(e.target.value)}
            className={[showPrefix ? 'pl-9' : '', className].filter(Boolean).join(' ')}
          />
        </div>
      </div>
    )
  },
)
MoneyInput.displayName = 'MoneyInput'
