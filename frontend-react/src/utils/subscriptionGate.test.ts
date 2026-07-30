import { describe, expect, it } from 'vitest'
import type { BootstrapPhase, GateDecision } from './subscriptionGate'
import { resolveSubscriptionGate } from './subscriptionGate'
import type { TenantStatus } from '../types'

describe('resolveSubscriptionGate', () => {
  it.each<{
    phase: BootstrapPhase
    tenantStatus?: TenantStatus
    mock?: boolean
    expected: GateDecision
  }>([
    { phase: 'blocked', expected: 'blocked' },
    { phase: 'error', expected: 'unavailable' },
    { phase: 'loading', expected: 'unavailable' },
    { phase: 'idle', expected: 'unavailable' },
    { phase: 'ready', expected: 'unavailable' },
    { phase: 'ready', tenantStatus: 'SUSPENSO', expected: 'blocked' },
    { phase: 'ready', tenantStatus: 'VENCIDO', expected: 'blocked' },
    { phase: 'ready', tenantStatus: 'ATIVO', expected: 'allow' },
    { phase: 'idle', mock: true, expected: 'allow' },
    { phase: 'idle', tenantStatus: 'ATIVO', mock: true, expected: 'allow' },
    { phase: 'idle', tenantStatus: 'SUSPENSO', mock: true, expected: 'blocked' },
  ])(
    'retorna $expected para phase=$phase, tenant=$tenantStatus e mock=$mock',
    ({ expected, ...input }) => {
      expect(resolveSubscriptionGate(input)).toBe(expected)
    },
  )
})
