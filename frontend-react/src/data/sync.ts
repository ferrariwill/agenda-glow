import { apiFetch } from '../lib/api'
import { getStoreDb, setStoreDb } from './store'
import { mapAdminBootstrap, mapTenantBootstrap } from './mapBootstrap'
import type { UserRole } from '../types'

export async function syncTenantBootstrap(role?: UserRole): Promise<void> {
  const path =
    role === 'PROFISSIONAL' ? '/api/v1/professional/bootstrap' : '/api/v1/bootstrap'
  const payload = await apiFetch<Parameters<typeof mapTenantBootstrap>[0]>(path)
  setStoreDb(mapTenantBootstrap(payload, getStoreDb()))
}

export async function syncAdminBootstrap(): Promise<void> {
  const payload = await apiFetch<Parameters<typeof mapAdminBootstrap>[0]>('/api/v1/admin/bootstrap')
  setStoreDb(mapAdminBootstrap(payload, getStoreDb()))
}

export async function syncPublicCatalog(slug: string): Promise<void> {
  const catalog = await apiFetch<{
    estabelecimento: {
      id: string
      nome_comercial: string
      slug: string
      logo_url?: string
      early_slot_notifications_available?: boolean
    }
    profissionais: { id: string; nome: string; especialidade: string }[]
    servicos: {
      id: string
      nome: string
      preco_base: number
      duracao_base_minutos: number
      adicionais?: unknown[]
    }[]
  }>(`/api/v1/public/${slug}/catalog`)

  const tenantId = catalog.estabelecimento.id
  const prev = getStoreDb()
  const tenant = {
    id: tenantId,
    nome: catalog.estabelecimento.nome_comercial,
    slug: catalog.estabelecimento.slug,
    status: 'ATIVO' as const,
    logo_url: catalog.estabelecimento.logo_url,
    early_slot_notifications_available:
      catalog.estabelecimento.early_slot_notifications_available,
    plano_id: '',
    data_vencimento: '',
    criado_em: '',
  }

  setStoreDb({
    ...prev,
    tenants: [...prev.tenants.filter((t) => t.id !== tenantId), tenant],
    profissionais: [
      ...prev.profissionais.filter((p) => p.tenant_id !== tenantId),
      ...catalog.profissionais.map((p) => ({
        id: p.id,
        tenant_id: tenantId,
        nome: p.nome,
        especialidade_id: '',
        comissao_percent: 0,
        ativo: true,
        expedientes: [],
      })),
    ],
    servicos: [
      ...prev.servicos.filter((s) => s.tenant_id !== tenantId),
      ...catalog.servicos.map((s) => ({
        id: s.id,
        tenant_id: tenantId,
        nome: s.nome,
        preco: s.preco_base,
        duracao_minutos: s.duracao_base_minutos,
        ativo: true,
        permitir_agendamento_online: true,
      })),
    ],
  })
}

export async function refreshAfterMutation(role?: UserRole): Promise<void> {
  if (role === 'SUPER_ADMIN') {
    await syncAdminBootstrap()
    return
  }
  await syncTenantBootstrap(role)
}

export async function fetchPublicSlots(
  slug: string,
  profissionalId: string,
  data: string,
  procedimentoId: string,
): Promise<string[]> {
  const params = new URLSearchParams({
    data,
    profissional_id: profissionalId,
    procedimento_id: procedimentoId,
  })
  const slots = await apiFetch<string[]>(`/api/v1/public/${slug}/slots?${params}`)
  return slots ?? []
}
