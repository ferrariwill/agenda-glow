import { apiFetch } from '../lib/api'
import type { EarlySlotOfferStatus } from '../types'

export interface EarlySlotOfferDetail {
  status: EarlySlotOfferStatus
  salon_name: string
  professional_name: string
  service_name: string
  current_start: string
  current_end: string
  offered_start: string
  offered_end: string
  expires_at: string
  seconds_remaining: number
  actions_allowed: Array<'accept' | 'decline'>
}

export interface EarlySlotAcceptResponse {
  status: 'ACEITA'
  appointment_id: string
  new_start: string
  new_end: string
}

export interface EarlySlotDeclineResponse {
  status: 'RECUSADA'
}

function offerPath(token: string): string {
  return `/api/v1/public/early-slot-offers/${encodeURIComponent(token)}`
}

export function getEarlySlotOffer(
  token: string,
  signal?: AbortSignal,
): Promise<EarlySlotOfferDetail> {
  return apiFetch<EarlySlotOfferDetail>(offerPath(token), { signal })
}

export function acceptEarlySlotOffer(token: string): Promise<EarlySlotAcceptResponse> {
  return apiFetch<EarlySlotAcceptResponse>(`${offerPath(token)}/accept`, { method: 'POST' })
}

export function declineEarlySlotOffer(token: string): Promise<EarlySlotDeclineResponse> {
  return apiFetch<EarlySlotDeclineResponse>(`${offerPath(token)}/decline`, { method: 'POST' })
}

export interface EarlySlotPreferenceResponse {
  aceita_adiantar: boolean
  aceita_adiantar_em?: string | null
  early_slot_notifications_available?: boolean
}

export function updateEarlySlotPreference(
  managementToken: string,
  aceitaAdiantar: boolean,
): Promise<EarlySlotPreferenceResponse> {
  return apiFetch<EarlySlotPreferenceResponse>(
    `/api/v1/public/appointments/manage/${encodeURIComponent(managementToken)}/early-slot-preference`,
    {
      method: 'PATCH',
      body: JSON.stringify({ aceita_adiantar: aceitaAdiantar }),
    },
  )
}
