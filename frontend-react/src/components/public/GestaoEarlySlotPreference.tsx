import { useState } from 'react'
import { Alert } from '../ui/Alert'
import { EarlySlotPreferenceToggle } from './EarlySlotPreferenceToggle'
import { patchEarlySlotPreference } from '../../services/earlySlotOfferApi'
import {
  resolvePreferenceFailure,
  resolvePreferenceSuccess,
} from './gestaoEarlySlotPreferenceState'

interface GestaoEarlySlotPreferenceProps {
  gestaoToken: string
  /** Preferência atual devolvida pelo GET de gestão. */
  aceitaAdiantar: boolean
  /** Elegibilidade é decisão da API (AGENDADO/CONFIRMADO e futuro), nunca do client. */
  elegivel?: boolean
  profissionalNome?: string
  /** Canonical channel signal from manage GET / PATCH echo. */
  notificationsAvailable?: boolean
  onChange?: (
    aceitaAdiantar: boolean,
    aceitaAdiantarEm?: string,
    notificationsAvailable?: boolean,
  ) => void
}

/**
 * Bloco de preferência de antecipação para a página pública de gestão do
 * atendimento (`/p/agendamento/:token`, DEV-82/84). Reverte a UI se a API recusar.
 */
export function GestaoEarlySlotPreference({
  gestaoToken,
  aceitaAdiantar,
  elegivel = true,
  profissionalNome,
  notificationsAvailable,
  onChange,
}: GestaoEarlySlotPreferenceProps) {
  const [checked, setChecked] = useState(aceitaAdiantar)
  const [channelAvailable, setChannelAvailable] = useState(notificationsAvailable)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const alterar = async (value: boolean) => {
    const anterior = checked
    setChecked(value)
    setSaving(true)
    setError('')
    try {
      const result = await patchEarlySlotPreference(gestaoToken, value)
      const next = resolvePreferenceSuccess(channelAvailable, result)
      setChecked(next.checked)
      if (next.hasChannelEcho) {
        setChannelAvailable(next.channelAvailable)
      }
      onChange?.(
        next.onChange.aceitaAdiantar,
        next.onChange.aceitaAdiantarEm,
        next.onChange.notificationsAvailable,
      )
    } catch (err) {
      const next = resolvePreferenceFailure(anterior, err)
      setChecked(next.checked)
      setError(next.error)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-3">
      <EarlySlotPreferenceToggle
        checked={checked}
        onChange={(value) => void alterar(value)}
        disabled={!elegivel || saving}
        profissionalNome={profissionalNome}
        notificationsAvailable={channelAvailable}
      />
      {!elegivel && (
        <p className="text-xs text-aura-muted">
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
