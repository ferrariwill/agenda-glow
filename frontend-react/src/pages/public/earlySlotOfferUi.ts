import { ApiError } from '../../lib/api'
import type { EarlySlotOfferDetail } from '../../services/earlySlotOfferService'

export type EarlySlotOfferViewState =
  | 'loading'
  | 'available'
  | 'submitting_accept'
  | 'submitting_decline'
  | 'accepted'
  | 'declined'
  | 'expired'
  | 'unavailable'
  | 'ineligible'
  | 'not_found'
  | 'error'

export function terminalOfferView(
  status: EarlySlotOfferDetail['status'],
): EarlySlotOfferViewState {
  switch (status) {
    case 'ACEITA':
      return 'accepted'
    case 'RECUSADA':
      return 'declined'
    case 'EXPIRADA':
      return 'expired'
    case 'INVALIDADA':
      return 'unavailable'
    default:
      return 'available'
  }
}

export function offerErrorView(error: unknown): EarlySlotOfferViewState {
  if (!(error instanceof ApiError)) return 'error'
  if (error.status === 404 || error.code === 'offer_not_found') return 'not_found'
  if (error.status === 410 || error.code === 'offer_expired') return 'expired'
  if (error.status === 409 || error.code === 'offer_no_longer_available') return 'unavailable'
  if (error.status === 422 || error.code === 'appointment_no_longer_eligible') return 'ineligible'
  return 'error'
}

export function nextExpiryRefreshState(
  alreadyRefetched: boolean,
  status: EarlySlotOfferDetail['status'],
  reconciledSeconds: number,
): boolean {
  if (status !== 'PENDENTE') return true
  if (reconciledSeconds > 0) return false
  return alreadyRefetched
}

/** Prefer POST confirmation times; for terminal GET ACEITA use offer times. */
export function acceptedScheduleStart(
  accepted: { new_start: string } | null,
  offer: Pick<EarlySlotOfferDetail, 'offered_start'> | null,
): string | null {
  if (accepted?.new_start) return accepted.new_start
  if (offer?.offered_start) return offer.offered_start
  return null
}
