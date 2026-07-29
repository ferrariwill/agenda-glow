import { describe, expect, it } from 'vitest'
import { mapAdminBootstrap } from './mapBootstrap'
import { adminBootstrapLive } from './__fixtures__/adminBootstrapLive'
import { statusTenantBadge } from '../components/ui/Badge'
import { matchesTenantStatusFilter } from '../utils/tenantStatus'
import type { MockDatabase } from '../types'

// Payload real da API (8081) com os três estados de assinatura — guarda o
// contrato API → painel Super Admin (BUG-S4-01).
const payload = adminBootstrapLive as unknown as Parameters<typeof mapAdminBootstrap>[0]

function emptyDb(): MockDatabase {
  return {
    planos: [],
    tenants: [],
    faturas_saas: [],
    categorias: [],
    clientes: [],
    especialidades: [],
    profissionais: [],
    servicos: [],
    insumos: [],
    adicionais: [],
    agendamentos: [],
    fila_espera: [],
    lancamentos: [],
    users: [],
    contas_cliente: [],
    cliente_preferencias: [],
    cliente_notas: [],
    cliente_galeria: [],
  }
}

const tenants = mapAdminBootstrap(payload, emptyDb()).tenants
const bySlug = (slug: string) => {
  const tenant = tenants.find((t) => t.slug === slug)
  if (!tenant) throw new Error(`fixture sem o salão ${slug}`)
  return tenant
}

describe('painel Super Admin sobre payload real da API', () => {
  it('exibe exatamente o status_assinatura devolvido pela API', () => {
    for (const origem of payload.tenants) {
      expect(bySlug(origem.slug).status).toBe(origem.status_assinatura)
    }
  })

  it('mantém VENCIDO mesmo com ativo=true no payload', () => {
    const origem = payload.tenants.find((t) => t.slug === 'qa-retest-vencido')
    expect(origem?.ativo).toBe(true)
    expect(bySlug('qa-retest-vencido').status).toBe('VENCIDO')
  })

  it('mantém SUSPENSO mesmo com vencimento futuro e ativo=false', () => {
    const origem = payload.tenants.find((t) => t.slug === 'qa-retest-suspenso')
    expect(origem?.ativo).toBe(false)
    expect(bySlug('qa-retest-suspenso').status).toBe('SUSPENSO')
  })

  it('agrupa vencido e suspenso no filtro de inativos e distingue os badges', () => {
    expect(matchesTenantStatusFilter(bySlug('qa-retest-vencido').status, 'INATIVOS')).toBe(true)
    expect(matchesTenantStatusFilter(bySlug('qa-retest-suspenso').status, 'INATIVOS')).toBe(true)
    expect(matchesTenantStatusFilter(bySlug('qa-retest-ativo').status, 'INATIVOS')).toBe(false)

    expect(statusTenantBadge(bySlug('qa-retest-ativo').status)).toBe('success')
    expect(statusTenantBadge(bySlug('qa-retest-vencido').status)).toBe('danger')
    expect(statusTenantBadge(bySlug('qa-retest-suspenso').status)).toBe('warning')
  })
})
