import type { ReactNode } from 'react'
import { Button } from '../ui/Button'

export type BookingStickyBarProps = {
  summary: ReactNode
  onConfirm: () => void
  loading?: boolean
  disabled?: boolean
  confirmLabel?: string
  validationMessage?: string
}

/**
 * CTA sticky inferior do fluxo público de agendamento.
 * Safe-area + resumo curto acima do botão Confirmar.
 */
export function BookingStickyBar({
  summary,
  onConfirm,
  loading = false,
  disabled = false,
  confirmLabel = 'Confirmar agendamento',
  validationMessage,
}: BookingStickyBarProps) {
  return (
    <div
      className="fixed bottom-0 left-0 right-0 z-30 border-t border-[#e5d3c8]/40 bg-white/95 shadow-[0_-4px_20px_rgba(183,132,114,0.12)] backdrop-blur-md pb-[max(1rem,env(safe-area-inset-bottom))]"
      role="region"
      aria-label="Confirmar agendamento"
    >
      <div className="mx-auto max-w-lg px-4 pt-3">
        {validationMessage && (
          <p className="mb-2 text-sm text-red-700" role="alert">
            {validationMessage}
          </p>
        )}
        <p className="mb-2 text-sm text-[#514440]">{summary}</p>
        <Button
          fullWidth
          loading={loading}
          disabled={disabled}
          onClick={onConfirm}
          className="bg-[#7d5141] hover:bg-[#996958]"
        >
          {confirmLabel}
        </Button>
      </div>
    </div>
  )
}

/** Padding inferior do scroll container para não cobrir conteúdo com a barra. */
export const BOOKING_STICKY_BAR_SPACE = 'pb-[7.5rem]'
