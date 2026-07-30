import { describe, expect, it } from 'vitest'
import { formatCountdown, secondsUntil } from './useOfferCountdown'

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

  it('deriva os segundos restantes de expires_at, não do seconds_remaining', () => {
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
