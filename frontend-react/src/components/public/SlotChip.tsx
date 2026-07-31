import { Check } from 'lucide-react'

export type SlotChipProps = {
  value: string
  label: string
  selected: boolean
  disabled?: boolean
  onSelect: (v: string) => void
}

/**
 * Chip de horário para agendamento público (e reutilizável na agenda operacional).
 * Piso de toque 44px; seleção por cor + aria-pressed + check (não só cor).
 */
export function SlotChip({ value, label, selected, disabled = false, onSelect }: SlotChipProps) {
  return (
    <button
      type="button"
      role="option"
      aria-selected={selected}
      aria-pressed={selected}
      disabled={disabled}
      onClick={() => onSelect(value)}
      className={[
        'relative inline-flex min-h-touch-min touch-manipulation items-center justify-center gap-1 rounded-lg border px-2 text-sm font-medium transition-colors',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#7d5141]/40 focus-visible:ring-offset-2',
        selected
          ? 'border-[#7d5141] bg-[#7d5141] text-white'
          : 'border-[#d6c2bd]/50 bg-white text-[#1a1c1c] hover:border-[#7d5141]',
        disabled ? 'cursor-not-allowed opacity-40' : '',
      ].join(' ')}
    >
      {selected && <Check className="h-3.5 w-3.5 shrink-0" aria-hidden />}
      <span>{label}</span>
    </button>
  )
}
