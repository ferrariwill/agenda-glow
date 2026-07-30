import { useState } from 'react'
import { Alert } from '../ui/Alert'
import {
  updateEarlySlotPreference,
  type EarlySlotPreferenceResponse,
} from '../../services/earlySlotOfferService'
import { ApiError } from '../../lib/api'

interface GestaoEarlySlotPreferenceProps {
  gestaoToken: string
  aceitaAdiantar: boolean
  /** Elegibilidade vem da API de gestão; o client não decide. */
  elegivel?: boolean
  notificationsAvailable?: boolean
  profissionalNome?: string
  onChange?: (result: EarlySlotPreferenceResponse) => void
}

/**
 * Preferência de antecipação pronta para embed em `/p/agendamento/:token` (DEV-84).
 * Controla loading, reverte em erro (inclui 422) e ecoa o sinal canônico do PATCH.
 */
export function GestaoEarlySlotPreference({
  gestaoToken,
  aceitaAdiantar,
  elegivel = true,
  notificationsAvailable,
  profissionalNome,
  onChange,
}: GestaoEarlySlotPreferenceProps) {
  const [checked, setChecked] = useState(aceitaAdiantar)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [available, setAvailable] = useState(notificationsAvailable)

  const alterar = async (value: boolean) => {
    const anterior = checked
    setChecked(value)
    setSaving(true)
    setError('')
    try {
      const result = await updateEarlySlotPreference(gestaoToken, value)
      setChecked(result.aceita_adiantar)
      if (result.early_slot_notifications_available !== undefined) {
        setAvailable(result.early_slot_notifications_available)
      }
      onChange?.(result)
    } catch (err) {
      setChecked(anterior)
      if (err instanceof ApiError && (err.status === 422 || err.code === 'appointment_no_longer_eligible')) {
        setError('Este atendimento não aceita mais alterações de preferência.')
      } else {
        setError('Não foi possível atualizar a preferência. Tente novamente.')
      }
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-3">
      <label htmlFor="gestao-aceita-adiantar" className="flex cursor-pointer items-start gap-3">
        <input
          id="gestao-aceita-adiantar"
          type="checkbox"
          checked={checked}
          disabled={!elegivel || saving}
          onChange={(e) => void alterar(e.target.checked)}
          className="mt-1 rounded border-[#d6c2bd] text-[#7d5141]"
        />
        <span>
          <span className="block text-sm font-medium text-[#1a1c1c]">
            Quero ser avisado pelo WhatsApp se surgir um horário mais cedo
            {profissionalNome ? ` com ${profissionalNome}` : ''}.
          </span>
          <span className="mt-0.5 block text-xs text-[#514440]">
            A oferta é opcional e exclusiva por 5 minutos. Seu horário atual só muda se você aceitar.
          </span>
          {available === false && (
            <span className="mt-1 block text-xs font-medium text-amber-800">
              Os avisos começarão quando o salão reativar o WhatsApp.
            </span>
          )}
        </span>
      </label>
      {!elegivel && (
        <p className="text-xs text-[#514440]">
          Este atendimento não aceita mais alterações de preferência.
        </p>
      )}
      {error && (
        <Alert variant="error" onDismiss={() => setError('')}>
          {error}
        </Alert>
      )}
    </div>
  )
}
