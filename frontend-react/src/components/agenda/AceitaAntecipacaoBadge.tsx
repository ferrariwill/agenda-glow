import { Badge } from '../ui/Badge'

interface AceitaAntecipacaoBadgeProps {
  aceitaAdiantar?: boolean
  compact?: boolean
}

/** Sinaliza no card que o cliente optou por receber ofertas de horário mais cedo. */
export function AceitaAntecipacaoBadge({
  aceitaAdiantar,
  compact = false,
}: AceitaAntecipacaoBadgeProps) {
  if (!aceitaAdiantar) return null
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
  return <Badge variant="muted">Aceita antecipação</Badge>
}
