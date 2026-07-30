import { ArrowDown } from 'lucide-react'
import { formatTimestampBR, formatTimestampTimeBR } from '../../utils/format'

interface EarlySlotOfferSummaryProps {
  currentStart?: string
  currentEnd?: string
  offeredStart?: string
  offeredEnd?: string
}

function range(start?: string, end?: string) {
  if (!start) return '—'
  const inicio = formatTimestampBR(start)
  if (!end) return inicio
  return `${inicio} – ${formatTimestampTimeBR(end)}`
}

export function EarlySlotOfferSummary({
  currentStart,
  currentEnd,
  offeredStart,
  offeredEnd,
}: EarlySlotOfferSummaryProps) {
  return (
    <div className="space-y-2">
      <div className="rounded-xl border border-aura-border bg-aura-surface p-4">
        <p className="text-[11px] font-bold uppercase tracking-widest text-aura-muted">
          Horário atual
        </p>
        <p className="mt-1 text-sm font-medium text-aura-anthracite line-through decoration-aura-muted/50">
          {range(currentStart, currentEnd)}
        </p>
      </div>

      <div className="flex justify-center">
        <ArrowDown className="h-4 w-4 text-aura-primary" />
      </div>

      <div className="rounded-xl border border-aura-primary/30 bg-aura-primary/10 p-4">
        <p className="text-[11px] font-bold uppercase tracking-widest text-aura-primary-dark">
          Horário oferecido
        </p>
        <p className="mt-1 font-display text-lg font-semibold text-aura-primary-dark">
          {range(offeredStart, offeredEnd)}
        </p>
      </div>
    </div>
  )
}
