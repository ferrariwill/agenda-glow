import { useState } from 'react'
import { Alert } from '../ui/Alert'
import { EarlySlotPreferenceToggle } from './EarlySlotPreferenceToggle'
import { earlySlotErrorMessage, patchEarlySlotPreference } from '../../services/earlySlotOfferApi'

interface GestaoEarlySlotPreferenceProps {
  gestaoToken: string
  /** Preferência atual devolvida pelo GET de gestão. */
  aceitaAdiantar: boolean
  /** Elegibilidade é decisão da API (AGENDADO/CONFIRMADO e futuro), nunca do client. */
  elegivel?: boolean
  profissionalNome?: string
  onChange?: (aceitaAdiantar: boolean, aceitaAdiantarEm?: string) => void
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
  onChange,
}: GestaoEarlySlotPreferenceProps) {
  const [checked, setChecked] = useState(aceitaAdiantar)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const alterar = async (value: boolean) => {
    const anterior = checked
    setChecked(value)
    setSaving(true)
    setError('')
    try {
      const result = await patchEarlySlotPreference(gestaoToken, value)
      setChecked(result.aceita_adiantar)
      onChange?.(result.aceita_adiantar, result.aceita_adiantar_em)
    } catch (err) {
      setChecked(anterior)
      setError(earlySlotErrorMessage(err))
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
