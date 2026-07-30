import { describe, expect, it } from 'vitest'
import {
  formatCountdown,
  nextExpiryRefreshGate,
  reconcileSecondsRemaining,
  secondsUntil,
} from './useOfferCountdown'

describe('formatCountdown', () => {
  it('formata mm:ss com zero à esquerda', () => {
    expect(formatCountdown(300)).toBe('05:00')
    expect(formatCountdown(59)).toBe('00:59')
    expect(formatCountdown(61)).toBe('01:01')
  })

  it('nunca exibe tempo negativo', () => {
    expect(formatCountdown(-42)).toBe('00:00')
  })
})

describe('secondsUntil', () => {
  const agora = Date.parse('2026-08-01T13:00:00-03:00')

  it('deriva os segundos restantes de expires_at', () => {
    expect(secondsUntil('2026-08-01T13:05:00-03:00', agora)).toBe(300)
  })

  it('satura em zero quando o prazo já passou', () => {
    expect(secondsUntil('2026-08-01T12:59:00-03:00', agora)).toBe(0)
  })

  it('devolve null sem expires_at ou com timestamp inválido', () => {
    expect(secondsUntil(undefined, agora)).toBeNull()
    expect(secondsUntil('nao-e-data', agora)).toBeNull()
  })
})

describe('reconcileSecondsRemaining', () => {
  const agora = Date.parse('2026-08-01T13:00:00-03:00')

  it('usa seconds_remaining como semente limitada por expires_at', () => {
    expect(
      reconcileSecondsRemaining('2026-08-01T13:05:00-03:00', 187, agora),
    ).toBe(187)
    expect(
      reconcileSecondsRemaining('2026-08-01T13:01:00-03:00', 999, agora),
    ).toBe(60)
  })

  it('não inventa validade positiva quando a semente é zero', () => {
    expect(
      reconcileSecondsRemaining('2026-08-01T13:05:00-03:00', 0, agora),
    ).toBe(0)
  })
})

describe('nextExpiryRefreshGate', () => {
  it('impede novo GET em loop quando ainda há 0s após o refresh', () => {
    expect(nextExpiryRefreshGate(0)).toBe(true)
    expect(nextExpiryRefreshGate(120)).toBe(false)
  })
})
