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
})
