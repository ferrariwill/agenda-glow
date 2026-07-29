import type { TenantStatus } from '../types'
import { isTenantAtivo } from './tenantStatus'

export type BootstrapPhase = 'idle' | 'loading' | 'ready' | 'blocked' | 'error'
export type GateDecision = 'allow' | 'blocked' | 'unavailable'

/**
 * Decide o acesso a uma rota com `checkSubscription`. Falha fechado: sem prova de
 * assinatura ativa, o painel não abre. `tenantStatus` undefined = tenant não resolvido.
 */
export function resolveSubscriptionGate(input: {
  phase: BootstrapPhase
  tenantStatus?: TenantStatus
  mock?: boolean
}): GateDecision {
  const { phase, tenantStatus, mock } = input

  if (mock) {
    if (!tenantStatus) return 'allow'
    return isTenantAtivo(tenantStatus) ? 'allow' : 'blocked'
  }

  if (phase === 'blocked') return 'blocked'
  if (phase === 'error' || phase === 'loading' || phase === 'idle') return 'unavailable'
  if (!tenantStatus) return 'unavailable'
  return isTenantAtivo(tenantStatus) ? 'allow' : 'blocked'
}
