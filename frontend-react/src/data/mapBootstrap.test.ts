import { describe, expect, it } from 'vitest'
import { mapAdminBootstrap, mapTenantBootstrap } from './mapBootstrap'
import type { MockDatabase, TenantStatus } from '../types'

type AdminPayload = Parameters<typeof mapAdminBootstrap>[0]
type TenantPayload = Parameters<typeof mapTenantBootstrap>[0]

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

function adminPayload(
  tenants: { id: string; ativo: boolean; status_assinatura: TenantStatus }[],
): AdminPayload {
  return {
    planos: [],
    tenants: tenants.map((t) => ({
      id: t.id,
      nome_comercial: `Salão ${t.id}`,
      slug: t.id,
      ativo: t.ativo,
      data_cadastro: '2026-01-10T00:00:00Z',
      status_assinatura: t.status_assinatura,
    })),
  }
}

function tenantPayload(status: TenantStatus): TenantPayload {
  return {
    tenant: {
      id: 'tenant-1',
      nome: 'Salão Teste',
      slug: 'salao-teste',
      status,
    },
    especialidades: [],
    profissionais: [],
    servicos: [],
    clientes: [],
    agendamentos: [],
    lancamentos: [],
  }
}

describe('mapAdminBootstrap', () => {
  it('usa status_assinatura como fonte de verdade nos três estados', () => {
    const db = mapAdminBootstrap(
      adminPayload([
        { id: 'ativo', ativo: true, status_assinatura: 'ATIVO' },
        { id: 'vencido', ativo: true, status_assinatura: 'VENCIDO' },
        { id: 'suspenso', ativo: false, status_assinatura: 'SUSPENSO' },
      ]),
      emptyDb(),
    )

    expect(db.tenants.map((t) => t.status)).toEqual(['ATIVO', 'VENCIDO', 'SUSPENSO'])
  })

  it('não deixa ativo=true promover um vencido para ATIVO', () => {
    const db = mapAdminBootstrap(
      adminPayload([{ id: 'vencido', ativo: true, status_assinatura: 'VENCIDO' }]),
      emptyDb(),
    )

    expect(db.tenants[0].status).toBe('VENCIDO')
  })

  it('não deixa ativo=false rebaixar um suspenso para VENCIDO', () => {
    const db = mapAdminBootstrap(
      adminPayload([{ id: 'suspenso', ativo: false, status_assinatura: 'SUSPENSO' }]),
      emptyDb(),
    )

    expect(db.tenants[0].status).toBe('SUSPENSO')
  })

  it('ignora ativo=false quando a API informa ATIVO', () => {
    const db = mapAdminBootstrap(
      adminPayload([{ id: 'ativo', ativo: false, status_assinatura: 'ATIVO' }]),
      emptyDb(),
    )

    expect(db.tenants[0].status).toBe('ATIVO')
  })
})

function agendamentoPayload(
  extra: Partial<TenantPayload['agendamentos'][number]>,
): TenantPayload {
  return {
    ...tenantPayload('ATIVO'),
    agendamentos: [
      {
        id: 'ag-1',
        tenant_id: 'tenant-1',
        profissional_id: 'prof-1',
        servico_id: 'srv-1',
        servico_ids: ['srv-1'],
        adicional_ids: [],
        cliente_nome: 'Maria',
        cliente_telefone: '11999999999',
        data: '2026-08-01',
        hora_inicio: '16:00',
        status: 'CONFIRMADO',
        ...extra,
      },
    ],
  }
}

describe('mapTenantBootstrap', () => {
  it('preserva os três estados sem colapsar SUSPENSO em ATIVO', () => {
    const statuses: TenantStatus[] = ['ATIVO', 'VENCIDO', 'SUSPENSO']

    for (const status of statuses) {
      const db = mapTenantBootstrap(tenantPayload(status), emptyDb())
      expect(db.tenants[0].status).toBe(status)
    }
  })

  it('mapeia o opt-in e a oferta ativa de antecipação do agendamento', () => {
    const db = mapTenantBootstrap(
      agendamentoPayload({
        aceita_adiantar: true,
        early_slot_offer: {
          round_id: 'rodada-1',
          offer_status: 'PENDENTE',
          expires_at: '2026-08-01T13:05:00-03:00',
          posicao: 1,
        },
      }),
      emptyDb(),
    )

    expect(db.agendamentos[0].aceita_adiantar).toBe(true)
    expect(db.agendamentos[0].early_slot_offer).toEqual({
      round_id: 'rodada-1',
      offer_status: 'PENDENTE',
      expires_at: '2026-08-01T13:05:00-03:00',
      posicao: 1,
    })
  })

  it('não quebra quando o backend ainda não envia early_slot_offer', () => {
    const db = mapTenantBootstrap(agendamentoPayload({}), emptyDb())

    expect(db.agendamentos[0].early_slot_offer).toBeNull()
  })

  it('mapeia o sinal canônico da fila sem derivar de flags brutas', () => {
    const payload = tenantPayload('ATIVO') as TenantPayload & {
      tenant: TenantPayload['tenant'] & {
        early_slot_queue_active?: boolean
        early_slot_queue_inactive_reason?: 'whatsapp_indisponivel' | null
      }
    }
    payload.tenant.early_slot_queue_active = false
    payload.tenant.early_slot_queue_inactive_reason = 'whatsapp_indisponivel'

    const db = mapTenantBootstrap(payload, emptyDb())
    expect(db.tenants[0].early_slot_queue_active).toBe(false)
    expect(db.tenants[0].early_slot_queue_inactive_reason).toBe('whatsapp_indisponivel')
  })

  it('aceita early_slot_queue_active na raiz do bootstrap profissional', () => {
    const payload = {
      ...tenantPayload('ATIVO'),
      early_slot_queue_active: false,
      early_slot_queue_inactive_reason: 'whatsapp_indisponivel' as const,
    }
    const db = mapTenantBootstrap(payload, emptyDb())
    expect(db.tenants[0].early_slot_queue_active).toBe(false)
  })

  it('mapeia confirmacao_cliente e ultimo_lembrete_enviado_em quando a API envia', () => {
    const db = mapTenantBootstrap(
      agendamentoPayload({
        confirmacao_cliente: 'CONFIRMADO_CLIENTE',
        ultimo_lembrete_enviado_em: '2026-08-01T13:00:00-03:00',
      }),
      emptyDb(),
    )

    expect(db.agendamentos[0]).toMatchObject({
      confirmacao_cliente: 'CONFIRMADO_CLIENTE',
      ultimo_lembrete_enviado_em: '2026-08-01T13:00:00-03:00',
    })
  })

  it('assume PENDENTE e lembrete nulo quando a API ainda n�o envia os campos', () => {
    const db = mapTenantBootstrap(agendamentoPayload({}), emptyDb())

    expect(db.agendamentos[0]).toMatchObject({
      confirmacao_cliente: 'PENDENTE',
      ultimo_lembrete_enviado_em: null,
    })
  })

  it('mapeia profissional_ids, eh_dona e dona_atua_como_profissional do bootstrap', () => {
    const payload: TenantPayload = {
      ...tenantPayload('ATIVO'),
      tenant: {
        ...tenantPayload('ATIVO').tenant,
        dona_atua_como_profissional: true,
      },
      profissionais: [
        {
          id: 'prof-dona',
          nome: 'Maria Dona',
          especialidade_id: 'esp-1',
          especialidade: 'Geral',
          comissao_porcentagem: 0,
          ativo: true,
          eh_dona: true,
          user_id: 'user-dona',
          expedientes: [],
        },
        {
          id: 'prof-1',
          nome: 'Ana',
          especialidade_id: 'esp-1',
          especialidade: 'Geral',
          comissao_porcentagem: 40,
          ativo: true,
          eh_dona: false,
          expedientes: [],
        },
      ],
      servicos: [
        {
          id: 'srv-1',
          nome: 'Corte',
          preco_base: 80,
          duracao_base_minutos: 45,
          ativo: true,
          categoria_id: 'cat-1',
          profissional_ids: ['prof-dona', 'prof-1'],
        },
        {
          id: 'srv-2',
          nome: 'Escova',
          preco_base: 50,
          duracao_base_minutos: 30,
          ativo: true,
          profissional_ids: [],
        },
      ],
    }

    const db = mapTenantBootstrap(payload, emptyDb())

    expect(db.tenants[0].dona_atua_como_profissional).toBe(true)
    expect(db.profissionais.find((p) => p.id === 'prof-dona')).toMatchObject({
      eh_dona: true,
      user_id: 'user-dona',
    })
    expect(db.servicos.find((s) => s.id === 'srv-1')).toMatchObject({
      categoria_id: 'cat-1',
      profissional_ids: ['prof-dona', 'prof-1'],
    })
    expect(db.servicos.find((s) => s.id === 'srv-2')?.profissional_ids).toEqual([])
  })

  it('usa profissional_ids vazio quando o campo falta no JSON (legado)', () => {
    const payload: TenantPayload = {
      ...tenantPayload('ATIVO'),
      servicos: [
        {
          id: 'srv-legado',
          nome: 'Legado',
          preco_base: 10,
          duracao_base_minutos: 15,
          ativo: true,
        },
      ],
    }

    const db = mapTenantBootstrap(payload, emptyDb())
    expect(db.servicos[0].profissional_ids).toEqual([])
  })
})
