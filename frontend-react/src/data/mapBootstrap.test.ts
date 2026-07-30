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

describe('mapTenantBootstrap', () => {
  it('preserva os três estados sem colapsar SUSPENSO em ATIVO', () => {
    const statuses: TenantStatus[] = ['ATIVO', 'VENCIDO', 'SUSPENSO']

    for (const status of statuses) {
      const db = mapTenantBootstrap(tenantPayload(status), emptyDb())
      expect(db.tenants[0].status).toBe(status)
    }
  })

  it('preserva opt-in e resumo da oferta sem exigir os novos campos', () => {
    const payload = tenantPayload('ATIVO')
    payload.agendamentos = [{
      id: 'ag-1',
      tenant_id: 'tenant-1',
      profissional_id: 'prof-1',
      servico_id: 'serv-1',
      servico_ids: ['serv-1'],
      adicional_ids: [],
      cliente_nome: 'Cliente',
      cliente_telefone: '5511999999999',
      data: '2026-08-01',
      hora_inicio: '16:00',
      status: 'AGENDADO',
      aceita_adiantar: true,
      early_slot_offer: {
        round_id: 'round-1',
        offer_status: 'PENDENTE',
        expires_at: '2026-08-01T13:05:00-03:00',
      },
    }]

    const mapped = mapTenantBootstrap(payload, emptyDb())

    expect(mapped.agendamentos[0].aceita_adiantar).toBe(true)
    expect(mapped.agendamentos[0].early_slot_offer?.round_id).toBe('round-1')
    expect(mapTenantBootstrap(tenantPayload('ATIVO'), emptyDb()).agendamentos).toEqual([])
  })

  it('mapeia apenas o sinal canônico da fila nos formatos staff', () => {
    const staff = tenantPayload('ATIVO')
    staff.tenant.early_slot_queue_active = false
    staff.tenant.early_slot_queue_inactive_reason = 'whatsapp_indisponivel'

    const professional = tenantPayload('ATIVO')
    professional.early_slot_queue_active = true

    expect(mapTenantBootstrap(staff, emptyDb()).tenants[0]).toMatchObject({
      early_slot_queue_active: false,
      early_slot_queue_inactive_reason: 'whatsapp_indisponivel',
    })
    expect(mapTenantBootstrap(professional, emptyDb()).tenants[0].early_slot_queue_active).toBe(true)
  })
})
