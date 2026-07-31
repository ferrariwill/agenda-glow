import { useEffect, useRef } from 'react'
import { CheckCircle2, X } from 'lucide-react'

export type ToastFeedbackVariant = 'success' | 'info' | 'error'

interface ToastFeedbackProps {
  message: string | null
  onDismiss: () => void
  /** Auto-dismiss in ms; 0 disables. Default 5000. */
  durationMs?: number
  variant?: ToastFeedbackVariant
  className?: string
}

const variantClass: Record<ToastFeedbackVariant, string> = {
  success: 'bg-[#7d5141] text-white',
  info: 'border border-aura-border bg-white text-aura-anthracite shadow-lg',
  error: 'bg-red-700 text-white',
}

/**
 * Feedback fixo na metade inferior da viewport.
 * Em &lt;lg fica acima da BottomNav + safe-area; em lg+ no canto inferior direito.
 * z-50: acima da BottomNav (z-40), abaixo de ActionsDropdown (z-[200]).
 */
export function ToastFeedback({
  message,
  onDismiss,
  durationMs = 5000,
  variant = 'success',
  className = '',
}: ToastFeedbackProps) {
  const onDismissRef = useRef(onDismiss)
  onDismissRef.current = onDismiss

  useEffect(() => {
    if (!message || durationMs <= 0) return
    const t = window.setTimeout(() => onDismissRef.current(), durationMs)
    return () => window.clearTimeout(t)
  }, [message, durationMs])

  if (!message) return null

  return (
    <div
      role="status"
      aria-live="polite"
      className={[
        'pointer-events-auto fixed inset-x-4 z-50 flex max-w-sm items-center gap-3 rounded-xl px-4 py-3 shadow-2xl sm:inset-x-auto sm:right-6',
        /* BottomNav (~4rem) + gap + safe-area; somece em lg */
        'bottom-[calc(5.5rem+env(safe-area-inset-bottom,0px))] lg:bottom-6',
        variantClass[variant],
        className,
      ]
        .filter(Boolean)
        .join(' ')}
    >
      {variant === 'success' && <CheckCircle2 className="h-5 w-5 shrink-0" aria-hidden />}
      <p className="flex-1 text-sm font-semibold">{message}</p>
      <button
        type="button"
        onClick={onDismiss}
        className="inline-flex min-h-touch-min min-w-touch-min shrink-0 touch-manipulation items-center justify-center opacity-70 hover:opacity-100"
        aria-label="Fechar"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  )
}
