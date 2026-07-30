import { describe, expect, it } from 'vitest'
import { nextAgendaExpiryDelay } from './useEarlySlotAgendaExpiry'

describe('nextAgendaExpiryDelay', () => {
  const now = Date.parse('2026-08-01T13:00:00-03:00')

  it('agenda um único refresh para a oferta que expira primeiro', () => {
    expect(
      nextAgendaExpiryDelay(
        ['2026-08-01T13:05:00-03:00', '2026-08-01T13:02:00-03:00'],
        now,
      ),
    ).toBe(120_000)
  })

  it('dispara imediatamente para prazo vencido e ignora valores ausentes', () => {
    expect(nextAgendaExpiryDelay([undefined, null, '2026-08-01T12:59:00-03:00'], now)).toBe(0)
    expect(nextAgendaExpiryDelay([undefined, null], now)).toBeNull()
  })
})
