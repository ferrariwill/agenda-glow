import type {
  Agendamento,
  Cliente,
  Especialidade,
  Lancamento,
  MockDatabase,
  PlanoSaas,
  Profissional,
  Servico,
  Tenant,
  TenantStatus,
  WhatsAppIntegrationStatus,
} from '../types'
import { normalizeTenantStatus } from '../utils/tenantStatus'

interface TenantBootstrapApi {
  tenant: {
    id: string
    nome: string
    slug: string
    status: TenantStatus
    logo_url?: string
    plano_id?: string
    data_vencimento?: string
    whatsapp_enabled?: boolean
    whatsapp_status?: WhatsAppIntegrationStatus
    early_slot_queue_active?: boolean
    early_slot_queue_inactive_reason?: 'whatsapp_indisponivel' | null
  }
  early_slot_queue_active?: boolean
  early_slot_queue_inactive_reason?: 'whatsapp_indisponivel' | null
  plano?: PlanoSaas
  especialidades: { id: string; nome: string; ativo: boolean }[]
  profissionais: {
    id: string
    nome: string
    especialidade_id: string
    especialidade: string
    comissao_porcentagem: number
    ativo: boolean
    pendente_aprovacao?: boolean
    expedientes: {
      dia_semana: number
      horario_entrada: string
      inicio_almoco?: string
      fim_almoco?: string
      horario_saida: string
    }[]
  }[]
  servicos: {
    id: string
    nome: string
    preco_base: number
    duracao_base_minutos: number
    ativo: boolean
    adicionais?: {
      id: string
      servico_id: string
      nome: string
      preco_adicional: number
      duracao_adicional_minutos: number
    }[]
  }[]
  clientes: {
    id: string
    nome: string
    telefone: string
    email?: string
    ativo: boolean
    criado_em: string
  }[]
  agendamentos: {
    id: string
    tenant_id: string
    profissional_id: string
    servico_id: string
    servico_ids: string[]
    adicional_ids: string[]
    cliente_nome: string
    cliente_telefone: string
    data: string
    hora_inicio: string
    status: string
    minutos_invadidos?: number
    aceita_adiantar?: boolean
    early_slot_offer?: {
      round_id: string
      offer_status: import('../types').EarlySlotOfferStatus
      expires_at: string
    } | null
    valor_cobrado?: number
    metodo_pagamento?: string
    cobrado_em?: string
  }[]
  lancamentos: {
    id: string
    tenant_id: string
    tipo: string
    valor: number
    descricao: string
    data: string
    profissional_id?: string
  }[]
  fila_espera?: {
    id: string
    tenant_id: string
    profissional_id: string
    cliente_nome: string
    cliente_telefone: string
    status: string
    agendamento_id?: string
    data?: string
    hora_desejada?: string
  }[]
  insumos?: {
    id: string
    tenant_id: string
    nome: string
    marca?: string
    categoria: string
    quantidade: number
    estoque_minimo: number
    estoque_ideal: number
    valor_unitario: number
    unidade: string
    imagem_url?: string
    instrucoes_uso?: string
    ativo: boolean
  }[]
}

interface AdminBootstrapApi {
  tenants: {
    id: string
    nome_comercial: string
    slug: string
    ativo: boolean
    data_cadastro: string
    plano_id?: string
    plano_nome?: string
    data_vencimento?: string
    status_assinatura: TenantStatus
  }[]
  planos: PlanoSaas[]
}

export function mapTenantBootstrap(
  payload: TenantBootstrapApi,
  prev: MockDatabase,
): MockDatabase {
  const tenantId = payload.tenant.id
  const tenant: Tenant = {
    id: tenantId,
    nome: payload.tenant.nome,
    slug: payload.tenant.slug,
    status: normalizeTenantStatus(payload.tenant.status),
    logo_url: payload.tenant.logo_url,
    plano_id: payload.tenant.plano_id ?? '',
    data_vencimento: payload.tenant.data_vencimento ?? '',
    whatsapp_enabled: payload.tenant.whatsapp_enabled,
    whatsapp_status: payload.tenant.whatsapp_status,
    early_slot_queue_active:
      payload.tenant.early_slot_queue_active ?? payload.early_slot_queue_active,
    early_slot_queue_inactive_reason:
      payload.tenant.early_slot_queue_inactive_reason ?? payload.early_slot_queue_inactive_reason,
    criado_em: prev.tenants.find((t) => t.id === tenantId)?.criado_em ?? '',
  }

  const especialidades: Especialidade[] = payload.especialidades.map((e) => ({
    id: e.id,
    tenant_id: tenantId,
    nome: e.nome,
  }))

  const profissionais: Profissional[] = payload.profissionais.map((p) => ({
    id: p.id,
    tenant_id: tenantId,
    nome: p.nome,
    especialidade_id: p.especialidade_id,
    comissao_percent: p.comissao_porcentagem,
    ativo: p.ativo,
    pendente_aprovacao: p.pendente_aprovacao,
    expedientes: (p.expedientes ?? []).map((ex) => ({
      dia_semana: ex.dia_semana,
      horario_entrada: ex.horario_entrada.slice(0, 5),
      horario_saida: ex.horario_saida.slice(0, 5),
      inicio_almoco: ex.inicio_almoco?.slice(0, 5),
      fim_almoco: ex.fim_almoco?.slice(0, 5),
    })),
  }))

  const servicos: Servico[] = payload.servicos.map((s) => ({
    id: s.id,
    tenant_id: tenantId,
    nome: s.nome,
    preco: s.preco_base,
    duracao_minutos: s.duracao_base_minutos,
    ativo: s.ativo,
    permitir_agendamento_online: true,
  }))

  const adicionais = payload.servicos.flatMap((s) =>
    (s.adicionais ?? []).map((a) => ({
      id: a.id,
      servico_id: a.servico_id,
      nome: a.nome,
      duracao_minutos: a.duracao_adicional_minutos,
      preco: a.preco_adicional,
    })),
  )

  const clientes: Cliente[] = payload.clientes.map((c) => ({
    id: c.id,
    tenant_id: tenantId,
    nome: c.nome,
    telefone: c.telefone,
    email: c.email,
    ativo: c.ativo,
    criado_em: c.criado_em,
  }))

  const agendamentos: Agendamento[] = payload.agendamentos.map((a) => ({
    id: a.id,
    tenant_id: a.tenant_id,
    profissional_id: a.profissional_id,
    servico_id: a.servico_id,
    servico_ids: a.servico_ids,
    adicional_ids: a.adicional_ids ?? [],
    cliente_nome: a.cliente_nome,
    cliente_telefone: a.cliente_telefone,
    data: a.data,
    hora_inicio: a.hora_inicio,
    status: a.status as Agendamento['status'],
    minutos_invadidos: a.minutos_invadidos,
    aceita_adiantar: a.aceita_adiantar,
    early_slot_offer: a.early_slot_offer,
    valor_cobrado: a.valor_cobrado,
    metodo_pagamento: a.metodo_pagamento as Agendamento['metodo_pagamento'],
    cobrado_em: a.cobrado_em,
  }))

  const lancamentos: Lancamento[] = payload.lancamentos.map((l) => ({
    id: l.id,
    tenant_id: l.tenant_id,
    tipo: l.tipo as Lancamento['tipo'],
    valor: l.valor,
    descricao: l.descricao,
    data: l.data,
    profissional_id: l.profissional_id,
  }))

  const fila_espera = (payload.fila_espera ?? []).map((f) => ({
    id: f.id,
    tenant_id: f.tenant_id,
    profissional_id: f.profissional_id,
    cliente_nome: f.cliente_nome,
    cliente_telefone: f.cliente_telefone,
    status: f.status as import('../types').FilaStatus,
    agendamento_id: f.agendamento_id,
    data: f.data,
    hora: f.hora_desejada,
  }))

  const insumos = (payload.insumos ?? []).map((i) => ({
    id: i.id,
    tenant_id: i.tenant_id,
    nome: i.nome,
    marca: i.marca,
    categoria: i.categoria as import('../types').InsumoCategoria,
    quantidade: i.quantidade,
    estoque_minimo: i.estoque_minimo,
    estoque_ideal: i.estoque_ideal,
    valor_unitario: i.valor_unitario,
    unidade: i.unidade,
    imagem_url: i.imagem_url,
    instrucoes_uso: i.instrucoes_uso,
    ativo: i.ativo,
  }))

  const otherTenants = prev.tenants.filter((t) => t.id !== tenantId)
  const planos = payload.plano
    ? [...prev.planos.filter((p) => p.id !== payload.plano!.id), payload.plano]
    : prev.planos

  return {
    ...prev,
    tenants: [...otherTenants, tenant],
    planos,
    especialidades: [
      ...prev.especialidades.filter((e) => e.tenant_id !== tenantId),
      ...especialidades,
    ],
    profissionais: [
      ...prev.profissionais.filter((p) => p.tenant_id !== tenantId),
      ...profissionais,
    ],
    servicos: [...prev.servicos.filter((s) => s.tenant_id !== tenantId), ...servicos],
    adicionais: [...prev.adicionais.filter((a) => !servicos.some((s) => s.id === a.servico_id)), ...adicionais],
    clientes: [...prev.clientes.filter((c) => c.tenant_id !== tenantId), ...clientes],
    agendamentos: [
      ...prev.agendamentos.filter((a) => a.tenant_id !== tenantId),
      ...agendamentos,
    ],
    lancamentos: [
      ...prev.lancamentos.filter((l) => l.tenant_id !== tenantId),
      ...lancamentos,
    ],
    fila_espera: [
      ...prev.fila_espera.filter((f) => f.tenant_id !== tenantId),
      ...fila_espera,
    ],
    insumos: [...prev.insumos.filter((i) => i.tenant_id !== tenantId), ...insumos],
  }
}

export function mapAdminBootstrap(payload: AdminBootstrapApi, prev: MockDatabase): MockDatabase {
  const tenants: Tenant[] = payload.tenants.map((t) => ({
    id: t.id,
    nome: t.nome_comercial,
    slug: t.slug,
    status: normalizeTenantStatus(t.status_assinatura),
    plano_id: t.plano_id ?? '',
    data_vencimento: t.data_vencimento?.slice(0, 10) ?? '',
    criado_em: t.data_cadastro?.slice(0, 10) ?? '',
  }))
  return {
    ...prev,
    tenants,
    planos: payload.planos,
  }
}
