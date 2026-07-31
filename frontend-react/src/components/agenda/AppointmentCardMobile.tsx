import type { LucideIcon } from 'lucide-react'
import type { ReactNode } from 'react'
import { AceitaAntecipacaoBadge } from './AceitaAntecipacaoBadge'
import { EarlySlotRoundIndicator } from './EarlySlotRoundIndicator'
import type { EarlySlotOfferResumo } from '../../types/earlySlot'
import { Badge, type BadgeVariant } from '../ui/Badge'

export type AppointmentCardAction = {
  id: string
  label: string
  ariaLabel: string
  icon: LucideIcon
  onClick: () => void
  danger?: boolean
  tone?: 'default' | 'primary'
}

export type AppointmentCardMobileProps = {
  timeLabel: string
  clientName: string
  serviceLabel: string
  statusLabel: string
  statusDotClass: string
  confirmationLabel: string
  confirmationVariant: BadgeVariant
  confirmationHint?: string
  pago?: boolean
  aceitaAdiantar?: boolean
  earlySlotOffer?: EarlySlotOfferResumo | null
  actions: AppointmentCardAction[]
  cardClassName: string
  onOpenDetail?: () => void
  detailAriaLabel?: string
}

export function AppointmentCardMobile({
  timeLabel,
  clientName,
  serviceLabel,
  statusLabel,
  statusDotClass,
  confirmationLabel,
  confirmationVariant,
  confirmationHint,
  pago = false,
  aceitaAdiantar,
  earlySlotOffer,
  actions,
  cardClassName,
  onOpenDetail,
  detailAriaLabel,
}: AppointmentCardMobileProps) {
  const openLabel = detailAriaLabel ?? `Detalhe de ${clientName}, ${timeLabel}`

  return (
    <article className={['rounded-xl p-4 shadow-sm', cardClassName].join(' ')}>
      <div className="flex flex-col gap-3">
        <button
          type="button"
          className="min-w-0 flex-1 touch-manipulation text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#7d5141]/40 focus-visible:ring-offset-2"
          onClick={onOpenDetail}
          aria-label={openLabel}
          disabled={!onOpenDetail}
        >
          <p className="text-body font-bold uppercase tracking-wide text-[#7d5141]">{timeLabel}</p>
          <p className="mt-0.5 truncate text-base font-semibold text-aura-anthracite">{clientName}</p>
          <p className="truncate text-body text-aura-muted">{serviceLabel}</p>
          <div className="mt-1.5 flex flex-wrap items-center gap-1.5" title={confirmationHint}>
            <span className={`h-2 w-2 shrink-0 rounded-full ${statusDotClass}`} aria-hidden />
            <span className="text-body text-aura-muted">{statusLabel}</span>
            <Badge variant={confirmationVariant}>{confirmationLabel}</Badge>
            {pago && (
              <span className="text-body font-medium text-emerald-700" aria-label="Pago">
                · Pago
              </span>
            )}
          </div>
          {(aceitaAdiantar || earlySlotOffer) && (
            <div className="mt-2 flex flex-wrap items-center gap-1.5">
              <AceitaAntecipacaoBadge aceitaAdiantar={aceitaAdiantar} />
              <EarlySlotRoundIndicator offer={earlySlotOffer} />
            </div>
          )}
        </button>

        {actions.length > 0 && (
          <div
            role="toolbar"
            aria-label={`Ações de ${clientName}`}
            className="flex w-full gap-1 border-t border-[#d6c2bd]/40 pt-2"
          >
            {actions.map(({ id, label, ariaLabel, icon: Icon, onClick, danger, tone }) => (
              <button
                key={id}
                type="button"
                onClick={onClick}
                aria-label={ariaLabel}
                className={[
                  'inline-flex min-h-touch-min min-w-0 flex-1 touch-manipulation flex-col items-center justify-center gap-0.5 rounded-lg px-1 text-caption font-medium transition-colors',
                  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#7d5141]/40',
                  danger
                    ? 'text-red-600 hover:bg-red-50'
                    : tone === 'primary'
                      ? 'text-[#7d5141] hover:bg-[#efdcd1]/50'
                      : 'text-aura-muted hover:bg-white/60 hover:text-[#7d5141]',
                ].join(' ')}
              >
                <Icon className="h-4 w-4 shrink-0" aria-hidden />
                <span className="max-w-full truncate">{label}</span>
              </button>
            ))}
          </div>
        )}
      </div>
    </article>
  )
}

export function AppointmentCardMobileSkeleton() {
  return (
    <div
      className="animate-pulse rounded-xl border border-[#efdcd1]/40 bg-white p-4 shadow-sm"
      aria-hidden
    >
      <div className="h-4 w-28 rounded bg-[#e9e8e7]" />
      <div className="mt-2 h-5 w-40 rounded bg-[#e9e8e7]" />
      <div className="mt-2 h-4 w-32 rounded bg-[#eeeeed]" />
      <div className="mt-3 flex gap-2">
        <div className="h-3 w-16 rounded bg-[#eeeeed]" />
        <div className="h-3 w-20 rounded bg-[#eeeeed]" />
      </div>
      <div className="mt-4 flex gap-2 border-t border-[#efdcd1]/30 pt-3">
        <div className="h-11 flex-1 rounded-lg bg-[#eeeeed]" />
        <div className="h-11 flex-1 rounded-lg bg-[#eeeeed]" />
        <div className="h-11 flex-1 rounded-lg bg-[#eeeeed]" />
      </div>
    </div>
  )
}

export function AppointmentCardMobileSkeletonList({ count = 3 }: { count?: number }): ReactNode {
  return (
    <div className="space-y-3" role="status" aria-label="Carregando agenda">
      {Array.from({ length: count }, (_, i) => (
        <AppointmentCardMobileSkeleton key={i} />
      ))}
    </div>
  )
}
