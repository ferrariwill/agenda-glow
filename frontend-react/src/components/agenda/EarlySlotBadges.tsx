import type { EarlySlotOfferSummary } from '../../types'
import { Badge } from '../ui/Badge'

export function EarlySlotOptInBadge({
  enabled,
  compact = false,
}: {
  enabled?: boolean
  compact?: boolean
}) {
  if (!enabled) return null
  if (compact) {
    return (
      <span
        className="inline-flex h-4 w-4 shrink-0 items-center justify-center rounded bg-[#e9e8e7] text-[10px] font-bold text-[#7d5141]"
        title="Aceita antecipação"
        aria-label="Aceita antecipação"
      >
        ↑
      </span>
    )
  }
  return <Badge variant="default">Aceita antecipação</Badge>
}

export function EarlySlotOfferIndicator({
  offer,
  compact = false,
}: {
  offer?: EarlySlotOfferSummary | null
  compact?: boolean
}) {
  if (!offer) return null

  const prazo =
    offer.offer_status === 'PENDENTE'
      ? `até ${new Date(offer.expires_at).toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })}`
      : ''
  const label = `Oferta ${offer.offer_status.toLocaleLowerCase('pt-BR')}${prazo ? ` · ${prazo}` : ''}`

  if (compact) {
    if (offer.offer_status !== 'PENDENTE') return null
    return (
      <span
        className="inline-flex h-4 shrink-0 items-center rounded bg-aura-warning px-1 text-[10px] font-medium text-amber-900"
        title={label}
        aria-label={label}
      >
        Oferta
      </span>
    )
  }

  const variant = offer.offer_status === 'PENDENTE'
    ? 'warning'
    : offer.offer_status === 'ACEITA'
      ? 'success'
      : 'muted'

  return <Badge variant={variant}>{label}</Badge>
}
