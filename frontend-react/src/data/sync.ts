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

type PublicCatalogAdicional = {
  id: string
  servico_id?: string
  nome: string
  preco_adicional: number
  duracao_adicional_minutos: number
}

type PublicCatalogServico = {
  id: string
  nome: string
  preco_base: number
  duracao_base_minutos: number
  permitir_agendamento_online?: boolean
  adicionais?: PublicCatalogAdicional[]
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
    servicos: PublicCatalogServico[]
  }>(`/api/v1/public/${slug}/catalog`)

  const tenantId = catalog.estabelecimento.id
  const prev = getStoreDb()
  const tenant = {
    id: tenantId,
    nome: catalog.estabelecimento.nome_comercial,
    slug: catalog.estabelecimento.slug,
    status: 'ATIVO' as const,
    logo_url: catalog.estabelecimento.logo_url,
    early_slot_notifications_available: catalog.estabelecimento.early_slot_notifications_available,
    plano_id: '',
    data_vencimento: '',
    criado_em: '',
  }

  const servicos = catalog.servicos.map((s) => ({
    id: s.id,
    tenant_id: tenantId,
    nome: s.nome,
    preco: s.preco_base,
    duracao_minutos: s.duracao_base_minutos,
    ativo: true,
    permitir_agendamento_online: s.permitir_agendamento_online ?? true,
  }))

  const adicionais = catalog.servicos.flatMap((s) =>
    (s.adicionais ?? []).map((a) => ({
      id: a.id,
      servico_id: a.servico_id ?? s.id,
      nome: a.nome,
      duracao_minutos: a.duracao_adicional_minutos,
      preco: a.preco_adicional,
    })),
  )

  const servicoIds = new Set(servicos.map((s) => s.id))

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
    servicos: [...prev.servicos.filter((s) => s.tenant_id !== tenantId), ...servicos],
    adicionais: [
      ...prev.adicionais.filter((a) => !servicoIds.has(a.servico_id)),
      ...adicionais,
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
  adicionais?: string[],
): Promise<string[]> {
  const params = new URLSearchParams({
    data,
    profissional_id: profissionalId,
    procedimento_id: procedimentoId,
  })
  for (const id of adicionais ?? []) {
    if (id) params.append('adicionais', id)
  }
  const slots = await apiFetch<string[]>(`/api/v1/public/${slug}/slots?${params}`)
  return slots ?? []
}
