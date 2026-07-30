import { AlertCircle, X } from 'lucide-react'

interface AlertProps {
  variant?: 'error' | 'warning' | 'info'
  title?: string
  children: React.ReactNode
  onDismiss?: () => void
  className?: string
  role?: React.AriaRole
}

const variants = {
  error: 'border-red-200 bg-red-50 text-red-900',
  warning: 'border-aura-warning-border bg-aura-warning text-amber-950',
  info: 'border-aura-border bg-white text-aura-anthracite',
}

export function Alert({
  variant = 'info',
  title,
  children,
  onDismiss,
  className = '',
  role,
}: AlertProps) {
  return (
    <div
      role={role ?? (variant === 'error' ? 'alert' : undefined)}
      className={['flex gap-3 rounded-lg border p-4', variants[variant], className].join(' ')}
    >
      <AlertCircle className="mt-0.5 h-5 w-5 shrink-0 opacity-70" />
      <div className="flex-1 text-sm">
        {title && <p className="mb-1 font-semibold">{title}</p>}
        <div className="text-aura-muted">{children}</div>
      </div>
      {onDismiss && (
        <button type="button" onClick={onDismiss} className="shrink-0 opacity-60 hover:opacity-100">
          <X className="h-4 w-4" />
        </button>
      )}
    </div>
  )
}
