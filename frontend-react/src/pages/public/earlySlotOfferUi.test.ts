import { describe, expect, it } from 'vitest'
import { ApiError } from '../../lib/api'
import {
  nextExpiryRefreshState,
  offerErrorView,
  terminalOfferView,
} from './earlySlotOfferUi'

describe('earlySlotOfferUi', () => {
  it.each([
    [404, 'offer_not_found', 'not_found'],
    [410, 'offer_expired', 'expired'],
    [409, 'offer_no_longer_available', 'unavailable'],
    [422, 'appointment_no_longer_eligible', 'ineligible'],
  ] as const)('mapeia %i/%s sem analisar a mensagem', (status, code, expected) => {
    expect(offerErrorView(new ApiError('mensagem variável', status, code))).toBe(expected)
  })

  it.each([
    ['ACEITA', 'accepted'],
    ['RECUSADA', 'declined'],
    ['EXPIRADA', 'expired'],
    ['INVALIDADA', 'unavailable'],
    ['PENDENTE', 'available'],
  ] as const)('mapeia o estado terminal %s', (status, expected) => {
    expect(terminalOfferView(status)).toBe(expected)
  })

  it('mantém o refetch consumido quando PENDENTE volta com zero', () => {
    expect(nextExpiryRefreshState(false, 'PENDENTE', 0)).toBe(false)
    expect(nextExpiryRefreshState(true, 'PENDENTE', 0)).toBe(true)
  })

  it('libera uma nova confirmação somente após prazo positivo novo', () => {
    expect(nextExpiryRefreshState(true, 'PENDENTE', 45)).toBe(false)
    expect(nextExpiryRefreshState(false, 'EXPIRADA', 0)).toBe(true)
  })
})
