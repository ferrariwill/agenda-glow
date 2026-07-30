import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../lib/api'
import {
  acceptOffer,
  declineOffer,
  earlySlotErrorMessage,
  getOffer,
  isEarlySlotTerminalError,
  patchEarlySlotPreference,
} from './earlySlotOfferApi'

const jsonResponse = (body: unknown) =>
  new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'content-type': 'application/json' },
  })

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('earlySlotOfferApi paths', () => {
  it('codifica o token e usa os métodos do contrato público', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ status: 'PENDENTE' }))
      .mockResolvedValueOnce(jsonResponse({ status: 'ACEITA', new_start: '2026-08-01T14:00:00-03:00' }))
      .mockResolvedValueOnce(jsonResponse({ status: 'RECUSADA' }))
    vi.stubGlobal('fetch', fetchMock)

    await getOffer('token/com espaço')
    await acceptOffer('token/com espaço')
    await declineOffer('token/com espaço')

    expect(fetchMock.mock.calls.map(([url, init]) => [url, init?.method])).toEqual([
      ['/api/v1/public/early-slot-offers/token%2Fcom%20espa%C3%A7o', undefined],
      ['/api/v1/public/early-slot-offers/token%2Fcom%20espa%C3%A7o/accept', 'POST'],
      ['/api/v1/public/early-slot-offers/token%2Fcom%20espa%C3%A7o/decline', 'POST'],
    ])
  })

  it('envia somente aceita_adiantar no PATCH de preferência', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ aceita_adiantar: true, early_slot_notifications_available: false }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await patchEarlySlotPreference('manage-token', true)

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/public/appointments/manage/manage-token/early-slot-preference',
      expect.objectContaining({
        method: 'PATCH',
        body: JSON.stringify({ aceita_adiantar: true }),
      }),
    )
  })
})

describe('earlySlotErrorMessage', () => {
  it('traduz os códigos estáveis do contrato', () => {
    expect(earlySlotErrorMessage(new ApiError('x', 404, 'offer_not_found'))).toBe(
      'Oferta não encontrada.',
    )
    expect(earlySlotErrorMessage(new ApiError('x', 410, 'offer_expired'))).toBe(
      'Esta oferta expirou.',
    )
    expect(earlySlotErrorMessage(new ApiError('x', 409, 'offer_no_longer_available'))).toBe(
      'Este horário não está mais disponível.',
    )
    expect(
      earlySlotErrorMessage(new ApiError('x', 422, 'appointment_no_longer_eligible')),
    ).toBe('Este agendamento não está mais elegível para antecipação.')
  })

  it('usa o status quando a API não manda code', () => {
    expect(earlySlotErrorMessage(new ApiError('x', 410))).toBe('Esta oferta expirou.')
  })

  it('não vaza mensagem técnica em erro genérico', () => {
    const generico = earlySlotErrorMessage(new ApiError('pq: duplicate key value', 500))
    expect(generico).toBe('Não foi possível carregar a oferta. Tente novamente em instantes.')
    expect(earlySlotErrorMessage(new Error('boom'))).toBe(generico)
  })
})

describe('isEarlySlotTerminalError', () => {
  it('reconhece os estados finais do contrato', () => {
    expect(isEarlySlotTerminalError(new ApiError('x', 404, 'offer_not_found'))).toBe(true)
    expect(isEarlySlotTerminalError(new ApiError('x', 409, 'offer_no_longer_available'))).toBe(true)
  })

  it('não trata falha genérica como estado final', () => {
    expect(isEarlySlotTerminalError(new ApiError('x', 500))).toBe(false)
    expect(isEarlySlotTerminalError(new Error('offline'))).toBe(false)
  })
})
