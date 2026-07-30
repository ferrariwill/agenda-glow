import type { EarlySlotOfferSummary } from '../../types'
import { Badge } from '../ui/Badge'

export function EarlySlotOptInBadge({ enabled }: { enabled?: boolean }) {
  if (!enabled) return null
  return <Badge variant="default">Aceita antecipação</Badge>
}

export function EarlySlotOfferIndicator({
  offer,
}: {
  offer?: EarlySlotOfferSummary | null
}) {
  if (!offer) return null

  const variant = offer.offer_status === 'PENDENTE'
    ? 'warning'
    : offer.offer_status === 'ACEITA'
      ? 'success'
      : 'muted'

  return (
    <Badge variant={variant}>
      Oferta {offer.offer_status.toLocaleLowerCase('pt-BR')}
      {offer.offer_status === 'PENDENTE' && (
        <> · até {new Date(offer.expires_at).toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })}</>
      )}
    </Badge>
  )
}
