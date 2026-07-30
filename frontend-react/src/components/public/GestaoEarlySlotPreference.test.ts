import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../../lib/api'
import { updateEarlySlotPreference } from '../../services/earlySlotOfferService'

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('GestaoEarlySlotPreference behavior', () => {
  it('ecoa early_slot_notifications_available no PATCH e envia só aceita_adiantar', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          aceita_adiantar: true,
          aceita_adiantar_em: '2026-07-30T18:00:00Z',
          early_slot_notifications_available: false,
        }),
        { status: 200, headers: { 'content-type': 'application/json' } },
      ),
    )
    vi.stubGlobal('fetch', fetchMock)

    const result = await updateEarlySlotPreference('gestao-token', true)

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/public/appointments/manage/gestao-token/early-slot-preference',
      expect.objectContaining({
        method: 'PATCH',
        body: JSON.stringify({ aceita_adiantar: true }),
      }),
    )
    expect(result.early_slot_notifications_available).toBe(false)
  })

  it('preserva ApiError 422 para o host reverter a UI', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: 'appointment_no_longer_eligible' }), {
          status: 422,
          headers: { 'content-type': 'application/json' },
        }),
      ),
    )

    await expect(updateEarlySlotPreference('gestao-token', false)).rejects.toBeInstanceOf(ApiError)
  })
})
