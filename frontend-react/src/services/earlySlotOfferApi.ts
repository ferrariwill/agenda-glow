import { apiFetch, ApiError } from '../lib/api'
import type {
  EarlySlotAcceptResult,
  EarlySlotDeclineResult,
  EarlySlotOffer,
  EarlySlotPreferenceResult,
} from '../types/earlySlot'

const OFFERS_BASE = '/api/v1/public/early-slot-offers'

export function getOffer(token: string): Promise<EarlySlotOffer> {
  return apiFetch<EarlySlotOffer>(`${OFFERS_BASE}/${encodeURIComponent(token)}`)
}

export function acceptOffer(token: string): Promise<EarlySlotAcceptResult> {
  return apiFetch<EarlySlotAcceptResult>(`${OFFERS_BASE}/${encodeURIComponent(token)}/accept`, {
    method: 'POST',
  })
}

export function declineOffer(token: string): Promise<EarlySlotDeclineResult> {
  return apiFetch<EarlySlotDeclineResult>(`${OFFERS_BASE}/${encodeURIComponent(token)}/decline`, {
    method: 'POST',
  })
}

export function patchEarlySlotPreference(
  gestaoToken: string,
  aceitaAdiantar: boolean,
): Promise<EarlySlotPreferenceResult> {
  return apiFetch<EarlySlotPreferenceResult>(
    `/api/v1/public/appointments/manage/${encodeURIComponent(gestaoToken)}/early-slot-preference`,
    {
      method: 'PATCH',
      body: JSON.stringify({ aceita_adiantar: aceitaAdiantar }),
    },
  )
}

/** Mensagem de UI para os códigos estáveis do contrato; nunca expõe detalhe técnico. */
export function earlySlotErrorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    switch (err.code) {
      case 'offer_not_found':
        return 'Oferta não encontrada.'
      case 'offer_expired':
        return 'Esta oferta expirou.'
      case 'offer_no_longer_available':
        return 'Este horário não está mais disponível.'
      case 'appointment_no_longer_eligible':
        return 'Este agendamento não está mais elegível para antecipação.'
    }
    if (err.status === 404) return 'Oferta não encontrada.'
    if (err.status === 410) return 'Esta oferta expirou.'
    if (err.status === 409) return 'Este horário não está mais disponível.'
    if (err.status === 422) return 'Este agendamento não está mais elegível para antecipação.'
  }
  return 'Não foi possível carregar a oferta. Tente novamente em instantes.'
}

/** `true` quando o erro descreve um estado final da oferta (sem CTAs). */
export function isEarlySlotTerminalError(err: unknown): boolean {
  if (!(err instanceof ApiError)) return false
  return (
    err.code === 'offer_not_found' ||
    err.code === 'offer_expired' ||
    err.code === 'offer_no_longer_available' ||
    err.code === 'appointment_no_longer_eligible' ||
    err.status === 404 ||
    err.status === 410 ||
    err.status === 409 ||
    err.status === 422
  )
}
