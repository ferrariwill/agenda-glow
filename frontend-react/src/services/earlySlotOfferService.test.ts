import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  acceptEarlySlotOffer,
  declineEarlySlotOffer,
  getEarlySlotOffer,
  updateEarlySlotPreference,
} from './earlySlotOfferService'

const jsonResponse = (body: unknown) => new Response(JSON.stringify(body), {
  status: 200,
  headers: { 'content-type': 'application/json' },
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('earlySlotOfferService', () => {
  it('codifica o token e usa os métodos do contrato público', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({ status: 'PENDENTE' }))
      .mockResolvedValueOnce(jsonResponse({ status: 'ACEITA' }))
      .mockResolvedValueOnce(jsonResponse({ status: 'RECUSADA' }))
    vi.stubGlobal('fetch', fetchMock)

    await getEarlySlotOffer('token/com espaço')
    await acceptEarlySlotOffer('token/com espaço')
    await declineEarlySlotOffer('token/com espaço')

    expect(fetchMock.mock.calls.map(([url, init]) => [url, init.method])).toEqual([
      ['/api/v1/public/early-slot-offers/token%2Fcom%20espa%C3%A7o', undefined],
      ['/api/v1/public/early-slot-offers/token%2Fcom%20espa%C3%A7o/accept', 'POST'],
      ['/api/v1/public/early-slot-offers/token%2Fcom%20espa%C3%A7o/decline', 'POST'],
    ])
  })

  it('envia somente aceita_adiantar ao atualizar a preferência', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ aceita_adiantar: true }))
    vi.stubGlobal('fetch', fetchMock)

    await updateEarlySlotPreference('manage-token', true)

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/public/appointments/manage/manage-token/early-slot-preference',
      expect.objectContaining({
        method: 'PATCH',
        body: JSON.stringify({ aceita_adiantar: true }),
      }),
    )
  })
})
