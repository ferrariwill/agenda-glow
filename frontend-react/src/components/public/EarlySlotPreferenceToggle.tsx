interface EarlySlotPreferenceToggleProps {
  checked: boolean
  onChange: (checked: boolean) => void
  disabled?: boolean
  profissionalNome?: string
  className?: string
  /**
   * Canonical API signal. `false` shows the channel helper.
   * `undefined` = older backend — do not treat as inactive.
   */
  notificationsAvailable?: boolean
}

/**
 * Opt-in da fila de antecipação (DEV-85). Desmarcado por padrão; reutilizado no
 * agendamento online e na gestão pública do atendimento.
 */
export function EarlySlotPreferenceToggle({
  checked,
  onChange,
  disabled = false,
  profissionalNome,
  className = '',
  notificationsAvailable,
}: EarlySlotPreferenceToggleProps) {
  return (
    <label
      className={[
        'flex items-start gap-3',
        disabled ? 'cursor-not-allowed opacity-60' : 'cursor-pointer',
        className,
      ].join(' ')}
    >
      <input
        type="checkbox"
        checked={checked}
        disabled={disabled}
        onChange={(e) => onChange(e.target.checked)}
        className="mt-1 rounded border-[#d6c2bd] text-[#7d5141]"
      />
      <span>
        <span className="block text-sm font-medium text-[#1a1c1c]">
          Quero ser avisado pelo WhatsApp se surgir um horário mais cedo com este profissional.
        </span>
        <span className="mt-0.5 block text-xs text-[#514440]">
          Oferta opcional e exclusiva por 5 minutos. Seu horário atual só muda se você aceitar.
        </span>
        {notificationsAvailable === false && (
          <span className="mt-1 block text-xs font-medium text-amber-800">
            Os avisos começarão quando o salão reativar o WhatsApp.
          </span>
        )}
        {profissionalNome && (
          <span className="mt-0.5 block text-xs text-[#514440]/80">
            Profissional: {profissionalNome}
          </span>
        )}
      </span>
    </label>
  )
}
