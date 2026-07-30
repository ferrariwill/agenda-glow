import { describe, expect, it } from 'vitest'
import { ApiError } from '../../lib/api'
import {
  resolvePreferenceFailure,
  resolvePreferenceSuccess,
} from './gestaoEarlySlotPreferenceState'

describe('GestaoEarlySlotPreference state', () => {
  it.each([
    { echo: false, current: true },
    { echo: true, current: false },
  ])('aplica o eco $echo do canal e o propaga no onChange', ({ echo, current }) => {
    const next = resolvePreferenceSuccess(current, {
      aceita_adiantar: true,
      aceita_adiantar_em: '2026-07-30T22:00:00Z',
      early_slot_notifications_available: echo,
    })

    expect(next).toMatchObject({
      checked: true,
      channelAvailable: echo,
      hasChannelEcho: true,
      onChange: {
        aceitaAdiantar: true,
        aceitaAdiantarEm: '2026-07-30T22:00:00Z',
        notificationsAvailable: echo,
      },
    })
  })

  it('preserva o estado do canal quando o PATCH não envia o campo', () => {
    const next = resolvePreferenceSuccess(true, {
      aceita_adiantar: false,
      aceita_adiantar_em: '2026-07-30T22:00:00Z',
    })

    expect(next.channelAvailable).toBe(true)
    expect(next.hasChannelEcho).toBe(false)
    expect(next.onChange.notificationsAvailable).toBeUndefined()
  })

  it('reverte o toggle otimista e expõe a mensagem do ApiError 422', () => {
    const next = resolvePreferenceFailure(
      false,
      new ApiError('x', 422, 'appointment_no_longer_eligible'),
    )

    expect(next).toEqual({
      checked: false,
      error: 'Este agendamento não está mais elegível para antecipação.',
    })
  })
})
