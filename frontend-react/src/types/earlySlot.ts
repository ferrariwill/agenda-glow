/**
 * DTOs da fila automática de antecipação (DEV-85).
 * Espelham o contrato HTTP do backend — o client não calcula elegibilidade,
 * ordem da fila nem validade: só renderiza o que a API devolve.
 */

export type EarlySlotOfferStatus =
  | 'PENDENTE'
  | 'ACEITA'
  | 'RECUSADA'
  | 'EXPIRADA'
  | 'INVALIDADA'
  | (string & {})

export type EarlySlotOfferAction = 'accept' | 'decline'

/** `GET /api/v1/public/early-slot-offers/{token}` */
export interface EarlySlotOffer {
  status: EarlySlotOfferStatus
  salon_name?: string
  professional_name?: string
  service_name?: string
  current_start?: string
  current_end?: string
  offered_start?: string
  offered_end?: string
  expires_at?: string
  /** Semente inicial do contador; a validade real vem de `expires_at`. */
  seconds_remaining?: number
  actions_allowed?: EarlySlotOfferAction[]
}

/** `POST /api/v1/public/early-slot-offers/{token}/accept` */
export interface EarlySlotAcceptResult {
  status: EarlySlotOfferStatus
  appointment_id?: string
  new_start?: string
  new_end?: string
}

/** `POST /api/v1/public/early-slot-offers/{token}/decline` */
export interface EarlySlotDeclineResult {
  status: EarlySlotOfferStatus
}

/** `PATCH /api/v1/public/appointments/manage/{gestao_token}/early-slot-preference` */
export interface EarlySlotPreferenceResult {
  aceita_adiantar: boolean
  aceita_adiantar_em?: string
}

/** Resumo da oferta ativa embutido no agendamento (bootstrap / listagens). */
export interface EarlySlotOfferResumo {
  round_id: string
  offer_status: EarlySlotOfferStatus
  expires_at?: string
  posicao?: number
}

export const EARLY_SLOT_ERROR_CODES = [
  'offer_not_found',
  'offer_expired',
  'offer_no_longer_available',
  'appointment_no_longer_eligible',
] as const

export type EarlySlotErrorCode = (typeof EARLY_SLOT_ERROR_CODES)[number]
