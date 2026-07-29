import { describe, expect, it } from 'vitest'
import { statusTenantBadge } from './Badge'

describe('statusTenantBadge', () => {
  it('define variante explícita para os três estados de assinatura', () => {
    expect(statusTenantBadge('ATIVO')).toBe('success')
    expect(statusTenantBadge('VENCIDO')).toBe('danger')
    expect(statusTenantBadge('SUSPENSO')).toBe('warning')
  })
})
