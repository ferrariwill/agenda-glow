import { forwardRef, useId, type InputHTMLAttributes } from 'react'

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string
  error?: string
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ label, error, className = '', id, ...props }, ref) => {
    const autoId = useId()
    const inputId = id ?? (label ? label.toLowerCase().replace(/\s/g, '-') : autoId)
    const errorId = error ? `${inputId}-error` : undefined

    return (
      <div className="space-y-1.5">
        {label && (
          <label htmlFor={inputId} className="block text-body font-medium text-aura-anthracite">
            {label}
          </label>
        )}
        <input
          ref={ref}
          id={inputId}
          {...props}
          aria-invalid={error ? true : props['aria-invalid']}
          aria-describedby={errorId ?? props['aria-describedby']}
          className={[
            'min-h-touch-min w-full touch-manipulation rounded-lg border border-aura-border bg-white px-3 py-2.5 text-control text-aura-anthracite',
            'placeholder:text-aura-muted focus:border-aura-primary focus:outline-none focus:ring-2 focus:ring-aura-primary/20',
            'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-aura-primary/40',
            error ? 'border-red-400' : '',
            className,
          ].join(' ')}
        />
        {error && (
          <p id={errorId} className="text-caption text-red-600" role="alert">
            {error}
          </p>
        )}
      </div>
    )
  },
)
Input.displayName = 'Input'
