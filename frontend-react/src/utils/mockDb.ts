import type {
  Agendamento,
  AgendamentoStatus,
  Cliente,
  ClienteGaleriaItem,
  ClienteNotaInterna,
  ContaCliente,
  FaturaSaas,
  FaturaSaasStatus,
  FilaEspera,
  Lancamento,
  LancamentoTipo,
  MetodoPagamento,
  MockDatabase,
  PlanoSaas,
  Profissional,
  Tenant,
} from '../types'
import { PlanLimitExceededError } from '../types'
import { addDaysISO, formatMonthYearBR, formatNowBR, recentYearMonths } from './format'
import { IS_MOCK } from '../lib/config'
import { getStoreDb, setStoreDb } from '../data/store'
import { refreshAfterMutation, syncAdminBootstrap } from '../data/sync'
import { apiFetch } from '../lib/api'

const STORAGE_KEY = 'agendaglow_mock_db'
const SEED_VERSION = 13
const FECHAMENTO_NOTA_KEY = 'agendaglow_fechamento_nota'
const SEED_VERSION_KEY = 'agendaglow_mock_db_seed_v'

const uid = () => crypto.randomUUID()

const today = () => new Date().toISOString().slice(0, 10)

function addMonths(isoDate: string, months: number): string {
  const d = new Date(isoDate + 'T12:00:00')
  d.setMonth(d.getMonth() + months)
  return d.toISOString().slice(0, 10)
}

function buildFaturasSaas(
  tenants: Tenant[],
  planos: PlanoSaas[],
): FaturaSaas[] {
  const statusByTenant: Record<string, FaturaSaasStatus[]> = {
    'tenant-glow': ['SUCESSO', 'SUCESSO', 'SUCESSO', 'SUCESSO'],
    'tenant-vencido': ['PENDENTE', 'FALHA', 'SUCESSO', 'FALHA'],
  }
  const faturas: FaturaSaas[] = []
  let seq = 0
  for (const tenant of tenants) {
    const plano = planos.find((p) => p.id === tenant.plano_id)
    if (!plano) continue
    const statuses = statusByTenant[tenant.id] ?? ['SUCESSO', 'SUCESSO', 'PENDENTE']
    statuses.forEach((status, i) => {
      const data = addMonths(today(), -(i + 1))
      faturas.push({
        id: `fat-${tenant.id}-${++seq}`,
        tenant_id: tenant.id,
        valor: plano.preco_mensal,
        data,
        status,
        referencia_mes: yearMonthFromISO(data),
      })
    })
  }
  return faturas.sort((a, b) => b.data.localeCompare(a.data))
}

const EXP_SEG_SEX: import('../types').Expediente[] = [1, 2, 3, 4, 5].map((dia) => ({
  dia_semana: dia,
  horario_entrada: '09:00',
  horario_saida: '18:00',
  inicio_almoco: '12:00',
  fim_almoco: '13:00',
}))

const EXP_TER_SAB: import('../types').Expediente[] = [2, 3, 4, 5, 6].map((dia) => ({
  dia_semana: dia,
  horario_entrada: '10:00',
  horario_saida: '20:00',
  inicio_almoco: '13:00',
  fim_almoco: '14:00',
}))

function buildEquipeCatalog(
  tenantId: string,
  espCabelo: string,
  espUnhas: string,
  espEstetica: string,
): Profissional[] {
  const imgHair =
    'https://images.unsplash.com/photo-1560066984-138dadb4c035?w=400&q=80'
  const imgBarber =
    'https://images.unsplash.com/photo-1503951914875-452162b0f3f1?w=400&q=80'
  const imgEstetica =
    'https://images.unsplash.com/photo-1570172619644-dfd03ed5d881?w=400&q=80'
  const imgNail =
    'https://images.unsplash.com/photo-1604654894610-df63bc536371?w=400&q=80'

  return [
    {
      id: 'prof-helena',
      tenant_id: tenantId,
      nome: 'Helena Oliveira',
      email: 'helena.o@glow.local',
      foto_url: imgHair,
      especialidade_id: espCabelo,
      comissao_percent: 42,
      ativo: true,
      expedientes: EXP_SEG_SEX,
    },
    {
      id: 'prof-ricardo',
      tenant_id: tenantId,
      nome: 'Ricardo Santos',
      email: 'ricardo.s@glow.local',
      foto_url: imgBarber,
      especialidade_id: espCabelo,
      comissao_percent: 38,
      ativo: true,
      expedientes: EXP_TER_SAB,
    },
    {
      id: 'prof-mariana',
      tenant_id: tenantId,
      nome: 'Mariana Luz',
      email: 'mariana.l@glow.local',
      foto_url: imgEstetica,
      especialidade_id: espEstetica,
      comissao_percent: 40,
      ativo: false,
      expedientes: [],
    },
    {
      id: 'prof-carla',
      tenant_id: tenantId,
      nome: 'Carla Ribeiro',
      email: 'carla.r@glow.local',
      foto_url: imgHair,
      especialidade_id: espCabelo,
      comissao_percent: 35,
      ativo: true,
      expedientes: EXP_SEG_SEX,
    },
    {
      id: 'prof-bruno',
      tenant_id: tenantId,
      nome: 'Bruno Ferreira',
      email: 'bruno.f@glow.local',
      foto_url: imgBarber,
      especialidade_id: espCabelo,
      comissao_percent: 36,
      ativo: true,
      expedientes: EXP_TER_SAB,
    },
    {
      id: 'prof-leticia',
      tenant_id: tenantId,
      nome: 'Letícia Moura',
      email: 'leticia.m@glow.local',
      foto_url: imgNail,
      especialidade_id: espUnhas,
      comissao_percent: 45,
      ativo: true,
      expedientes: [
        {
          dia_semana: 1,
          horario_entrada: '10:00',
          horario_saida: '19:00',
          inicio_almoco: '13:00',
          fim_almoco: '14:00',
        },
        {
          dia_semana: 3,
          horario_entrada: '10:00',
          horario_saida: '19:00',
          inicio_almoco: '13:00',
          fim_almoco: '14:00',
        },
        {
          dia_semana: 5,
          horario_entrada: '10:00',
          horario_saida: '19:00',
          inicio_almoco: '13:00',
          fim_almoco: '14:00',
        },
      ],
    },
    {
      id: 'prof-julia',
      tenant_id: tenantId,
      nome: 'Julia Costa',
      email: 'julia.c@glow.local',
      foto_url: imgEstetica,
      especialidade_id: espEstetica,
      comissao_percent: 40,
      ativo: false,
      pendente_aprovacao: true,
      expedientes: EXP_SEG_SEX,
    },
    {
      id: 'prof-pedro',
      tenant_id: tenantId,
      nome: 'Pedro Alves',
      email: 'pedro.a@glow.local',
      foto_url: imgBarber,
      especialidade_id: espCabelo,
      comissao_percent: 35,
      ativo: false,
      pendente_aprovacao: true,
      expedientes: EXP_TER_SAB,
    },
    {
      id: 'prof-sofia',
      tenant_id: tenantId,
      nome: 'Sofia Martins',
      email: 'sofia.m@glow.local',
      foto_url: imgNail,
      especialidade_id: espUnhas,
      comissao_percent: 42,
      ativo: false,
      pendente_aprovacao: true,
      expedientes: [],
    },
    {
      id: 'prof-fernanda',
      tenant_id: tenantId,
      nome: 'Fernanda Dias',
      email: 'fernanda.d@glow.local',
      foto_url: imgEstetica,
      especialidade_id: espEstetica,
      comissao_percent: 38,
      ativo: true,
      expedientes: EXP_SEG_SEX,
    },
  ]
}

function buildFinanceiroLancamentos(
  tenantId: string,
  dataHoje: string,
  profClaudiaId: string,
): Lancamento[] {
  const d = (offset: number) => addDaysISO(dataHoje, offset)
  const items: Omit<Lancamento, 'id'>[] = [
    {
      tenant_id: tenantId,
      tipo: 'ENTRADA',
      valor: 850,
      descricao: 'Pacote Mechas Aura Glow',
      subtitulo: 'Cliente: Ana Clara Mendes',
      categoria: 'SERVICO',
      status_transacao: 'PAGO',
      data: d(-5),
    },
    {
      tenant_id: tenantId,
      tipo: 'CUSTO_FIXO',
      valor: 4200,
      descricao: 'Aluguel do Espaço - Shopping Crystal',
      subtitulo: 'Vencimento: 20/04',
      categoria: 'ALUGUEL',
      status_transacao: 'PENDENTE',
      vencimento: d(5),
      data: d(-6),
    },
    {
      tenant_id: tenantId,
      tipo: 'ENTRADA',
      valor: 1240,
      descricao: "Venda de Produtos L'Oréal",
      subtitulo: 'Kit Manutenção Home Care',
      categoria: 'PRODUTO',
      status_transacao: 'PAGO',
      data: d(-7),
    },
    {
      tenant_id: tenantId,
      tipo: 'CUSTO_FIXO',
      valor: 9800,
      descricao: 'Folha de Pagamento - Estilistas',
      subtitulo: 'Referência: mês anterior',
      categoria: 'SALARIOS',
      status_transacao: 'PAGO',
      data: d(-8),
    },
    {
      tenant_id: tenantId,
      tipo: 'ENTRADA',
      valor: 450,
      descricao: 'Mechas Californianas',
      subtitulo: 'Cliente: Mariana Silva',
      categoria: 'SERVICO',
      status_transacao: 'PAGO',
      data: d(-2),
    },
    {
      tenant_id: tenantId,
      tipo: 'ENTRADA',
      valor: 180,
      descricao: 'Corte & Finalização',
      subtitulo: 'Cliente: Beatriz Alves',
      categoria: 'SERVICO',
      status_transacao: 'PAGO',
      data: d(-1),
    },
    {
      tenant_id: tenantId,
      tipo: 'ENTRADA',
      valor: 120,
      descricao: 'Manicure Spa Gel',
      subtitulo: 'Cliente: Juliana Costa',
      categoria: 'SERVICO',
      status_transacao: 'PAGO',
      data: dataHoje,
    },
    {
      tenant_id: tenantId,
      tipo: 'CUSTO_VARIAVEL',
      valor: 72,
      descricao: 'Comissão Cláudia — Corte Mariana',
      categoria: 'SALARIOS',
      status_transacao: 'PENDENTE',
      data: dataHoje,
      profissional_id: profClaudiaId,
      status_pagamento: 'PENDENTE',
    },
    {
      tenant_id: tenantId,
      tipo: 'CUSTO_FIXO',
      valor: 890,
      descricao: 'Conta de energia elétrica',
      categoria: 'OUTRO',
      status_transacao: 'PAGO',
      data: d(-10),
    },
    {
      tenant_id: tenantId,
      tipo: 'CUSTO_VARIAVEL',
      valor: 340,
      descricao: 'Reposição insumos coloração',
      categoria: 'PRODUTO',
      status_transacao: 'PAGO',
      data: d(-12),
    },
    {
      tenant_id: tenantId,
      tipo: 'ENTRADA',
      valor: 680,
      descricao: 'Design de Sobrancelhas + Henna',
      subtitulo: 'Cliente: Camila Rocha',
      categoria: 'SERVICO',
      status_transacao: 'PAGO',
      data: d(-14),
    },
    {
      tenant_id: tenantId,
      tipo: 'CUSTO_FIXO',
      valor: 2500,
      descricao: 'Aluguel do espaço',
      categoria: 'ALUGUEL',
      status_transacao: 'PAGO',
      data: d(-35),
    },
    {
      tenant_id: tenantId,
      tipo: 'ENTRADA',
      valor: 920,
      descricao: 'Combo Noiva Premium',
      subtitulo: 'Cliente: Fernanda Lima',
      categoria: 'SERVICO',
      status_transacao: 'PAGO',
      data: d(-38),
    },
    {
      tenant_id: tenantId,
      tipo: 'CUSTO_FIXO',
      valor: 1200,
      descricao: 'Marketing digital — Instagram Ads',
      categoria: 'OUTRO',
      status_transacao: 'PENDENTE',
      data: d(-40),
    },
    {
      tenant_id: tenantId,
      tipo: 'ENTRADA',
      valor: 560,
      descricao: 'Hidratação Profunda Kérastase',
      subtitulo: 'Cliente: Patrícia Souza',
      categoria: 'SERVICO',
      status_transacao: 'PAGO',
      data: d(-42),
    },
    {
      tenant_id: tenantId,
      tipo: 'ENTRADA',
      valor: 310,
      descricao: 'Venda Shampoo Mythic Oil',
      categoria: 'PRODUTO',
      status_transacao: 'PAGO',
      data: d(-45),
    },
    {
      tenant_id: tenantId,
      tipo: 'CUSTO_FIXO',
      valor: 6500,
      descricao: 'Folha de Pagamento — Equipe',
      categoria: 'SALARIOS',
      status_transacao: 'PAGO',
      data: d(-48),
    },
    {
      tenant_id: tenantId,
      tipo: 'CUSTO_FIXO',
      valor: 1800,
      descricao: 'Internet + telefonia',
      categoria: 'OUTRO',
      status_transacao: 'PAGO',
      data: d(-50),
    },
  ]

  return items.map((item, i) => ({
    ...item,
    id: `lanc-fin-${i + 1}`,
  }))
}

function buildInsumosCatalog(tenantId: string): import('../types').InsumoEstoque[] {
  const imgHair =
    'https://images.unsplash.com/photo-1527799820374-dcf8d9a4e388?w=400&q=80'
  const imgNail =
    'https://images.unsplash.com/photo-1604654894610-df63bc536371?w=400&q=80'
  const imgEstetica =
    'https://images.unsplash.com/photo-1570172619644-dfd03ed5d881?w=400&q=80'
  const imgTools =
    'https://images.unsplash.com/photo-1560066984-138dadb4c035?w=400&q=80'

  const featured: import('../types').InsumoEstoque[] = [
    {
      id: 'ins-mythic-oil',
      tenant_id: tenantId,
      nome: 'Mythic Oil Professional',
      marca: "L'Oréal Professionnel",
      categoria: 'CUIDADOS_CAPILARES',
      quantidade: 24,
      estoque_minimo: 8,
      estoque_ideal: 30,
      valor_unitario: 185.9,
      unidade: 'un',
      ativo: true,
      imagem_url: imgHair,
    },
    {
      id: 'ins-acrylic-powder',
      tenant_id: tenantId,
      nome: 'Acrylic Powder Luxe Clear',
      marca: 'Tammy Taylor',
      categoria: 'TECNICA_UNHAS',
      quantidade: 3,
      estoque_minimo: 6,
      estoque_ideal: 12,
      valor_unitario: 342,
      unidade: 'un',
      ativo: true,
      imagem_url: imgNail,
    },
    {
      id: 'ins-hyaluronic-serum',
      tenant_id: tenantId,
      nome: 'Hyaluronic Glow Serum',
      marca: 'Bioage Professional',
      categoria: 'ESTETICA',
      quantidade: 15,
      estoque_minimo: 6,
      estoque_ideal: 25,
      valor_unitario: 128.5,
      unidade: 'un',
      ativo: true,
      imagem_url: imgEstetica,
    },
    {
      id: 'ins-carbon-comb',
      tenant_id: tenantId,
      nome: 'Carbon Pro Comb Set',
      marca: 'YS Park Japan',
      categoria: 'CUIDADOS_CAPILARES',
      quantidade: 6,
      estoque_minimo: 8,
      estoque_ideal: 18,
      valor_unitario: 54,
      unidade: 'un',
      ativo: true,
      imagem_url: imgTools,
    },
    {
      id: 'ins-color-1',
      tenant_id: tenantId,
      nome: 'Pó Descolorante Premium (50g)',
      marca: 'Wella Professionals',
      categoria: 'CUIDADOS_CAPILARES',
      quantidade: 18,
      estoque_minimo: 10,
      estoque_ideal: 24,
      valor_unitario: 45,
      unidade: 'un',
      ativo: true,
      imagem_url: imgHair,
    },
    {
      id: 'ins-color-2',
      tenant_id: tenantId,
      nome: 'Oxidante 20vol (100ml)',
      marca: 'Wella Professionals',
      categoria: 'CUIDADOS_CAPILARES',
      quantidade: 22,
      estoque_minimo: 12,
      estoque_ideal: 28,
      valor_unitario: 12,
      unidade: 'un',
      ativo: true,
      imagem_url: imgHair,
    },
    {
      id: 'ins-color-3',
      tenant_id: tenantId,
      nome: 'Tonalizante Gloss (1un)',
      marca: 'Redken',
      categoria: 'CUIDADOS_CAPILARES',
      quantidade: 14,
      estoque_minimo: 8,
      estoque_ideal: 20,
      valor_unitario: 38,
      unidade: 'un',
      ativo: true,
      imagem_url: imgHair,
    },
  ]

  const extras: Omit<import('../types').InsumoEstoque, 'id' | 'tenant_id'>[] = [
    { nome: 'Shampoo Metal Detox', marca: 'L\'Oréal', categoria: 'CUIDADOS_CAPILARES', quantidade: 11, estoque_minimo: 8, estoque_ideal: 20, valor_unitario: 89, unidade: 'un', ativo: true, imagem_url: imgHair },
    { nome: 'Máscara Absolut Repair', marca: 'L\'Oréal', categoria: 'CUIDADOS_CAPILARES', quantidade: 9, estoque_minimo: 6, estoque_ideal: 16, valor_unitario: 112, unidade: 'un', ativo: true, imagem_url: imgHair },
    { nome: 'Ampola Discipline', marca: 'Kérastase', categoria: 'CUIDADOS_CAPILARES', quantidade: 32, estoque_minimo: 15, estoque_ideal: 40, valor_unitario: 28, unidade: 'un', ativo: true, imagem_url: imgHair },
    { nome: 'Gel Builder Rose', marca: 'CND', categoria: 'TECNICA_UNHAS', quantidade: 7, estoque_minimo: 8, estoque_ideal: 15, valor_unitario: 98, unidade: 'un', ativo: true, imagem_url: imgNail },
    { nome: 'Base Rubber Coat', marca: 'Bluesky', categoria: 'TECNICA_UNHAS', quantidade: 5, estoque_minimo: 6, estoque_ideal: 14, valor_unitario: 42, unidade: 'un', ativo: true, imagem_url: imgNail },
    { nome: 'Lixa Buffer 180', marca: ' Mundial', categoria: 'TECNICA_UNHAS', quantidade: 40, estoque_minimo: 20, estoque_ideal: 50, valor_unitario: 8.5, unidade: 'un', ativo: true, imagem_url: imgNail },
    { nome: 'Ácido Hialurônico 2%', marca: 'Bioage', categoria: 'ESTETICA', quantidade: 4, estoque_minimo: 5, estoque_ideal: 12, valor_unitario: 156, unidade: 'un', ativo: true, imagem_url: imgEstetica },
    { nome: 'Máscara de Argila Verde', marca: 'Bioage', categoria: 'ESTETICA', quantidade: 8, estoque_minimo: 6, estoque_ideal: 14, valor_unitario: 74, unidade: 'un', ativo: true, imagem_url: imgEstetica },
    { nome: 'Peeling Enzimático', marca: 'SkinCeuticals', categoria: 'ESTETICA', quantidade: 3, estoque_minimo: 4, estoque_ideal: 10, valor_unitario: 210, unidade: 'un', ativo: true, imagem_url: imgEstetica },
    { nome: 'Toalha Descartável Premium', marca: 'Soft Touch', categoria: 'GERAL', quantidade: 120, estoque_minimo: 50, estoque_ideal: 150, valor_unitario: 0.85, unidade: 'un', ativo: true, imagem_url: imgTools },
    { nome: 'Luvas Nitrílicas (cx)', marca: 'Medix', categoria: 'GERAL', quantidade: 16, estoque_minimo: 10, estoque_ideal: 24, valor_unitario: 38, unidade: 'cx', ativo: true, imagem_url: imgTools },
    { nome: 'Algodão Hidrófilo 500g', marca: ' Cremer', categoria: 'GERAL', quantidade: 12, estoque_minimo: 8, estoque_ideal: 18, valor_unitario: 22, unidade: 'un', ativo: true, imagem_url: imgTools },
    { nome: 'Óleo Essencial Lavanda', marca: 'Aroma Spa', categoria: 'GERAL', quantidade: 6, estoque_minimo: 4, estoque_ideal: 12, valor_unitario: 48, unidade: 'un', ativo: true, imagem_url: imgTools },
    { nome: 'Cera Quente Depilatória', marca: ' Depil Bella', categoria: 'ESTETICA', quantidade: 10, estoque_minimo: 6, estoque_ideal: 16, valor_unitario: 65, unidade: 'un', ativo: true, imagem_url: imgEstetica },
    { nome: 'Pinça Profissional', marca: ' Rubis', categoria: 'TECNICA_UNHAS', quantidade: 2, estoque_minimo: 3, estoque_ideal: 8, valor_unitario: 185, unidade: 'un', ativo: true, imagem_url: imgNail },
    { nome: 'Spray Fixador Diamond', marca: ' L\'Oréal', categoria: 'CUIDADOS_CAPILARES', quantidade: 13, estoque_minimo: 8, estoque_ideal: 20, valor_unitario: 67, unidade: 'un', ativo: true, imagem_url: imgHair },
    { nome: 'Sérum Reparador de Pontas', marca: ' Kérastase', categoria: 'CUIDADOS_CAPILARES', quantidade: 7, estoque_minimo: 6, estoque_ideal: 15, valor_unitario: 142, unidade: 'un', ativo: true, imagem_url: imgHair },
  ]

  return [
    ...featured,
    ...extras.map((e, i) => ({
      ...e,
      id: `ins-extra-${i + 1}`,
      tenant_id: tenantId,
    })),
    ...Array.from({ length: 224 }, (_, i) => {
      const n = i + 25
      const cats: import('../types').InsumoCategoria[] = [
        'CUIDADOS_CAPILARES',
        'TECNICA_UNHAS',
        'ESTETICA',
        'GERAL',
      ]
      return {
        id: `ins-fill-${n}`,
        tenant_id: tenantId,
        nome: `Suprimento Profissional ${String(n).padStart(3, '0')}`,
        marca: 'Parceiro Premium',
        categoria: cats[n % 4],
        quantidade: 4 + (n % 14),
        estoque_minimo: 6,
        estoque_ideal: 18,
        valor_unitario: 18 + (n % 75),
        unidade: 'un',
        ativo: true,
        imagem_url: n % 3 === 0 ? imgNail : n % 3 === 1 ? imgEstetica : imgHair,
      } satisfies import('../types').InsumoEstoque
    }),
  ]
}

function seedDatabase(): MockDatabase {
  const planoEssencial: PlanoSaas = {
    id: 'plano-essencial',
    nome: 'Essencial',
    preco_mensal: 149.9,
    limite_profissionais: 3,
    ativo: true,
  }
  const planoProfissional: PlanoSaas = {
    id: 'plano-profissional',
    nome: 'Profissional',
    preco_mensal: 299.9,
    limite_profissionais: 8,
    ativo: true,
  }
  const planoElite: PlanoSaas = {
    id: 'plano-elite',
    nome: 'Elite',
    preco_mensal: 499.9,
    limite_profissionais: 20,
    ativo: true,
  }

  const tenantGlow: Tenant = {
    id: 'tenant-glow',
    nome: 'Estúdio Glow Salto',
    slug: 'estudio-glow-salto',
    status: 'ATIVO',
    plano_id: planoProfissional.id,
    bio: 'Beleza de luxo com atendimento personalizado no coração de Salto.',
    data_vencimento: addMonths(today(), 10),
    dona_nome: 'Dona Glow',
    dona_email: 'dona@glow.local',
    criado_em: addMonths(today(), -14),
    email_contato: 'contato@estudioglow.com.br',
    cep: '13320-000',
    logradouro: 'Rua das Flores, 120',
    cidade: 'Salto',
    uf: 'SP',
    whatsapp_status: 'DESCONECTADO',
  }

  const tenantVencido: Tenant = {
    id: 'tenant-vencido',
    nome: 'Salão Demo Vencido',
    slug: 'salao-demo-vencido',
    status: 'VENCIDO',
    plano_id: planoEssencial.id,
    bio: 'Demonstração de bloqueio por assinatura.',
    data_vencimento: addMonths(today(), -1),
    dona_nome: 'Maria Demo',
    dona_email: 'demo@vencido.local',
    criado_em: addMonths(today(), -24),
  }

  const espCabelo = { id: 'esp-cabelo', tenant_id: tenantGlow.id, nome: 'Cabelo' }
  const espUnhas = { id: 'esp-unhas', tenant_id: tenantGlow.id, nome: 'Unhas' }
  const espEstetica = { id: 'esp-estetica', tenant_id: tenantGlow.id, nome: 'Estética Facial' }

  const catCabelo = { id: 'cat-cabelo', tenant_id: tenantGlow.id, nome: 'Cabelo', icone: 'scissors' }
  const catUnhas = { id: 'cat-unhas', tenant_id: tenantGlow.id, nome: 'Unhas', icone: 'sparkles' }
  const catEstetica = { id: 'cat-estetica', tenant_id: tenantGlow.id, nome: 'Estética Facial', icone: 'flower' }
  const catDepilacao = { id: 'cat-depilacao', tenant_id: tenantGlow.id, nome: 'Depilação', icone: 'leaf' }
  const catMassagem = { id: 'cat-massagem', tenant_id: tenantGlow.id, nome: 'Massagem', icone: 'spa' }

  const profClaudia: Profissional = {
    id: 'prof-claudia',
    tenant_id: tenantGlow.id,
    nome: 'Cláudia Mendes',
    email: 'claudia@glow.local',
    foto_url:
      'https://images.unsplash.com/photo-1595476108010-b4d1f102b1b1?w=400&q=80',
    especialidade_id: espCabelo.id,
    comissao_percent: 40,
    ativo: true,
    expedientes: [
      {
        dia_semana: 1,
        horario_entrada: '09:00',
        horario_saida: '18:00',
        inicio_almoco: '12:00',
        fim_almoco: '13:00',
      },
      {
        dia_semana: 2,
        horario_entrada: '09:00',
        horario_saida: '18:00',
        inicio_almoco: '12:00',
        fim_almoco: '13:00',
      },
      {
        dia_semana: 3,
        horario_entrada: '09:00',
        horario_saida: '18:00',
        inicio_almoco: '12:00',
        fim_almoco: '13:00',
      },
      {
        dia_semana: 4,
        horario_entrada: '09:00',
        horario_saida: '18:00',
        inicio_almoco: '12:00',
        fim_almoco: '13:00',
      },
      {
        dia_semana: 5,
        horario_entrada: '09:00',
        horario_saida: '17:00',
        inicio_almoco: '12:00',
        fim_almoco: '13:00',
      },
    ],
  }

  const profAna: Profissional = {
    id: 'prof-ana',
    tenant_id: tenantGlow.id,
    nome: 'Ana Paula',
    email: 'ana.p@glow.local',
    foto_url:
      'https://images.unsplash.com/photo-1604654894610-df63bc536371?w=400&q=80',
    especialidade_id: espUnhas.id,
    comissao_percent: 45,
    ativo: true,
    expedientes: [
      {
        dia_semana: 1,
        horario_entrada: '10:00',
        horario_saida: '19:00',
        inicio_almoco: '13:00',
        fim_almoco: '14:00',
      },
      {
        dia_semana: 3,
        horario_entrada: '10:00',
        horario_saida: '19:00',
        inicio_almoco: '13:00',
        fim_almoco: '14:00',
      },
      {
        dia_semana: 5,
        horario_entrada: '10:00',
        horario_saida: '19:00',
        inicio_almoco: '13:00',
        fim_almoco: '14:00',
      },
    ],
  }

  const servicoCorte = {
    id: 'srv-corte',
    tenant_id: tenantGlow.id,
    nome: 'Corte & Finalização',
    descricao: 'Corte personalizado com acabamento premium',
    duracao_minutos: 60,
    preco: 180,
    ativo: true,
    categoria_id: catCabelo.id,
    profissional_ids: [profClaudia.id],
    exibir_catalogo_publico: true,
    permitir_agendamento_online: true,
    imagem_url: 'https://images.unsplash.com/photo-1560066984-138dadb4c035?w=400&q=80',
  }
  const servicoColor = {
    id: 'srv-color',
    tenant_id: tenantGlow.id,
    nome: 'Mechas Californianas',
    descricao:
      'Técnica de clareamento gradual que preserva a raiz natural, criando um efeito de iluminação solar sofisticado. Inclui diagnóstico capilar, aplicação de descolorante premium, tonalização personalizada e tratamento reconstrutor pós-química.',
    duracao_minutos: 180,
    preco: 450,
    ativo: true,
    categoria_id: catCabelo.id,
    profissional_ids: [profClaudia.id],
    exibir_catalogo_publico: true,
    permitir_agendamento_online: true,
    insumos: [
      { id: 'ins-color-1', nome: 'Pó Descolorante Premium (50g)', custo: 45 },
      { id: 'ins-color-2', nome: 'Oxidante 20vol (100ml)', custo: 12 },
      { id: 'ins-color-3', nome: 'Tonalizante Gloss (1un)', custo: 38 },
    ],
    imagem_url:
      'https://images.unsplash.com/photo-1522337360788-8b13dee7a37e?w=400&q=80',
  }
  const servicoManicure = {
    id: 'srv-manicure',
    tenant_id: tenantGlow.id,
    nome: 'Manicure Spa Gel',
    descricao: 'Hidratação com parafina inclusa',
    duracao_minutos: 60,
    preco: 120,
    ativo: true,
    categoria_id: catUnhas.id,
    profissional_ids: [profAna.id],
    exibir_catalogo_publico: true,
    permitir_agendamento_online: true,
    imagem_url:
      'https://images.unsplash.com/photo-1604654894610-df63bc536371?w=400&q=80',
  }

  const servicosExtras: import('../types').Servico[] = [
    {
      id: 'srv-hidratacao',
      tenant_id: tenantGlow.id,
      nome: 'Hidratação Profunda',
      descricao: 'Máscara reconstrutora com vapor ozonizado',
      duracao_minutos: 90,
      preco: 220,
      ativo: true,
      categoria_id: catCabelo.id,
      imagem_url: 'https://images.unsplash.com/photo-1519699047748-de8e457a634e?w=400&q=80',
    },
    {
      id: 'srv-escova',
      tenant_id: tenantGlow.id,
      nome: 'Escova Modelada',
      descricao: 'Finalização com brilho intenso',
      duracao_minutos: 45,
      preco: 95,
      ativo: true,
      categoria_id: catCabelo.id,
      imagem_url: 'https://images.unsplash.com/photo-1492106087820-71f1a00d2b11?w=400&q=80',
    },
    {
      id: 'srv-progressiva',
      tenant_id: tenantGlow.id,
      nome: 'Progressiva Orgânica',
      descricao: 'Alisamento sem formol',
      duracao_minutos: 240,
      preco: 680,
      ativo: true,
      categoria_id: catCabelo.id,
      imagem_url: 'https://images.unsplash.com/photo-1527799820374-dcf8d9a4e388?w=400&q=80',
    },
    {
      id: 'srv-tonalizacao',
      tenant_id: tenantGlow.id,
      nome: 'Tonalização Gloss',
      descricao: 'Revitalização de cor entre mechas',
      duracao_minutos: 75,
      preco: 190,
      ativo: true,
      categoria_id: catCabelo.id,
      imagem_url: 'https://images.unsplash.com/photo-1562322140-8baeececf3df?w=400&q=80',
    },
    {
      id: 'srv-penteado',
      tenant_id: tenantGlow.id,
      nome: 'Penteado Eventos',
      descricao: 'Produção completa para ocasiões especiais',
      duracao_minutos: 120,
      preco: 320,
      ativo: true,
      categoria_id: catCabelo.id,
      imagem_url: 'https://images.unsplash.com/photo-1487412947147-5cebf100ffc2?w=400&q=80',
    },
    {
      id: 'srv-pedicure',
      tenant_id: tenantGlow.id,
      nome: 'Pedicure Spa',
      descricao: 'Esfoliação e massagem nos pés',
      duracao_minutos: 60,
      preco: 110,
      ativo: true,
      categoria_id: catUnhas.id,
      imagem_url: 'https://images.unsplash.com/photo-1519014816548-bf789fe71516?w=400&q=80',
    },
    {
      id: 'srv-nail-art',
      tenant_id: tenantGlow.id,
      nome: 'Nail Art Premium',
      descricao: 'Design exclusivo com detalhes em folha de ouro',
      duracao_minutos: 90,
      preco: 185,
      ativo: true,
      categoria_id: catUnhas.id,
      imagem_url: 'https://images.unsplash.com/photo-1604654894610-df63bc536371?w=400&q=80',
    },
    {
      id: 'srv-alongamento',
      tenant_id: tenantGlow.id,
      nome: 'Alongamento em Gel',
      descricao: 'Aplicação e nivelamento profissional',
      duracao_minutos: 120,
      preco: 260,
      ativo: true,
      categoria_id: catUnhas.id,
      imagem_url: 'https://images.unsplash.com/photo-1632345031431-874cee6748b5?w=400&q=80',
    },
    {
      id: 'srv-massagem-pedras',
      tenant_id: tenantGlow.id,
      nome: 'Massagem Pedras Quentes',
      descricao: 'Terapia relaxante profunda',
      duracao_minutos: 90,
      preco: 280,
      ativo: false,
      categoria_id: catMassagem.id,
      imagem_url: 'https://images.unsplash.com/photo-1544161515-4ab6ce6db874?w=400&q=80',
    },
    {
      id: 'srv-massagem-relax',
      tenant_id: tenantGlow.id,
      nome: 'Massagem Relaxante',
      descricao: 'Óleos essenciais e aromaterapia',
      duracao_minutos: 60,
      preco: 210,
      ativo: true,
      categoria_id: catMassagem.id,
      imagem_url: 'https://images.unsplash.com/photo-1519823551278-64fa9279728a?w=400&q=80',
    },
    {
      id: 'srv-drenagem',
      tenant_id: tenantGlow.id,
      nome: 'Drenagem Linfática',
      descricao: 'Redução de retenção e detox corporal',
      duracao_minutos: 75,
      preco: 240,
      ativo: true,
      categoria_id: catMassagem.id,
      imagem_url: 'https://images.unsplash.com/photo-1596178065887-1198b6148b2b?w=400&q=80',
    },
    {
      id: 'srv-reflexologia',
      tenant_id: tenantGlow.id,
      nome: 'Reflexologia Podal',
      descricao: 'Estímulo de pontos energéticos',
      duracao_minutos: 45,
      preco: 150,
      ativo: true,
      categoria_id: catMassagem.id,
      imagem_url: 'https://images.unsplash.com/photo-1515377901643-9a295d252642?w=400&q=80',
    },
    {
      id: 'srv-limpeza-pele',
      tenant_id: tenantGlow.id,
      nome: 'Limpeza de Pele Diamond',
      descricao: 'Peeling ultrassônico incluso',
      duracao_minutos: 120,
      preco: 350,
      ativo: true,
      categoria_id: catEstetica.id,
      imagem_url: 'https://images.unsplash.com/photo-1570172619644-dfd03ed5d881?w=400&q=80',
    },
    {
      id: 'srv-peeling',
      tenant_id: tenantGlow.id,
      nome: 'Peeling Químico Suave',
      descricao: 'Renovação celular com ácidos de frutas',
      duracao_minutos: 60,
      preco: 290,
      ativo: true,
      categoria_id: catEstetica.id,
      imagem_url: 'https://images.unsplash.com/photo-1616394584738-fc6e612e71b9?w=400&q=80',
    },
    {
      id: 'srv-microagulhamento',
      tenant_id: tenantGlow.id,
      nome: 'Microagulhamento Facial',
      descricao: 'Estímulo de colágeno com LED',
      duracao_minutos: 90,
      preco: 420,
      ativo: true,
      categoria_id: catEstetica.id,
      imagem_url: 'https://images.unsplash.com/photo-1512290923902-8a9f81dc2360?w=400&q=80',
    },
    {
      id: 'srv-lifting',
      tenant_id: tenantGlow.id,
      nome: 'Lifting Facial Manual',
      descricao: 'Técnica coreana de modelagem',
      duracao_minutos: 75,
      preco: 310,
      ativo: true,
      categoria_id: catEstetica.id,
      imagem_url: 'https://images.unsplash.com/photo-1576091160399-112ba8d25d1d?w=400&q=80',
    },
    {
      id: 'srv-design-sobrancelha',
      tenant_id: tenantGlow.id,
      nome: 'Design de Sobrancelha',
      descricao: 'Visagismo e henna natural',
      duracao_minutos: 45,
      preco: 85,
      ativo: true,
      categoria_id: catEstetica.id,
      imagem_url: 'https://images.unsplash.com/photo-1522335789203-aabd1fc54bc9?w=400&q=80',
    },
    {
      id: 'srv-depilacao-axila',
      tenant_id: tenantGlow.id,
      nome: 'Depilação Axila',
      descricao: 'Cera hipoalergênica premium',
      duracao_minutos: 30,
      preco: 55,
      ativo: true,
      categoria_id: catDepilacao.id,
      imagem_url: 'https://images.unsplash.com/photo-1515377901643-9a295d252642?w=400&q=80',
    },
    {
      id: 'srv-depilacao-perna',
      tenant_id: tenantGlow.id,
      nome: 'Depilação Perna Completa',
      descricao: 'Acabamento com óleo calmante',
      duracao_minutos: 50,
      preco: 95,
      ativo: true,
      categoria_id: catDepilacao.id,
      imagem_url: 'https://images.unsplash.com/photo-1515377901643-9a295d252642?w=400&q=80',
    },
    {
      id: 'srv-botox-capilar',
      tenant_id: tenantGlow.id,
      nome: 'Botox Capilar',
      descricao: 'Reconstrução intensiva de fibras',
      duracao_minutos: 120,
      preco: 380,
      ativo: true,
      categoria_id: catCabelo.id,
      imagem_url: 'https://images.unsplash.com/photo-1522337360788-8b13dee7a37e?w=400&q=80',
    },
    {
      id: 'srv-spa-maos',
      tenant_id: tenantGlow.id,
      nome: 'Spa das Mãos',
      descricao: 'Esfoliação, parafina e massagem',
      duracao_minutos: 45,
      preco: 98,
      ativo: true,
      categoria_id: catUnhas.id,
      imagem_url: 'https://images.unsplash.com/photo-1604654894610-df63bc536371?w=400&q=80',
    },
  ]

  const servicos = [servicoCorte, servicoColor, servicoManicure, ...servicosExtras]

  const adicionalHidrata = {
    id: 'add-hidrata',
    servico_id: servicoCorte.id,
    nome: 'Hidratação Express',
    duracao_minutos: 30,
    preco: 80,
  }

  const dataHoje = today()

  const agendamentos: Agendamento[] = [
    {
      id: 'ag-mariana-0',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoCorte.id,
      adicional_ids: [],
      cliente_nome: 'Mariana Silva',
      cliente_telefone: '5511999887766',
      data: addMonths(dataHoje, -3),
      hora_inicio: '10:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-mariana-1',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoCorte.id,
      adicional_ids: [],
      cliente_nome: 'Mariana Silva',
      cliente_telefone: '5511999887766',
      data: addMonths(dataHoje, -2),
      hora_inicio: '10:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-mariana-2',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoCorte.id,
      adicional_ids: [],
      cliente_nome: 'Mariana Silva',
      cliente_telefone: '5511999887766',
      data: addMonths(dataHoje, -1),
      hora_inicio: '11:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-1',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoCorte.id,
      adicional_ids: [],
      cliente_nome: 'Mariana Silva',
      cliente_telefone: '5511999887766',
      data: dataHoje,
      hora_inicio: '09:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-2',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoColor.id,
      adicional_ids: [adicionalHidrata.id],
      cliente_nome: 'Juliana Costa',
      cliente_telefone: '5511988776655',
      data: dataHoje,
      hora_inicio: '14:00',
      status: 'EM_APROVACAO',
      minutos_invadidos: 15,
    },
    {
      id: 'ag-camila-1',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoCorte.id,
      adicional_ids: [],
      cliente_nome: 'Camila Rocha',
      cliente_telefone: '5511966554433',
      data: addMonths(dataHoje, -1),
      hora_inicio: '15:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-camila-2',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Camila Rocha',
      cliente_telefone: '5511966554433',
      data: dataHoje,
      hora_inicio: '16:00',
      status: 'CONFIRMADO',
    },
    {
      id: 'ag-beatriz-1',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Beatriz Alves',
      cliente_telefone: '5511977665544',
      data: addMonths(dataHoje, -1),
      hora_inicio: '14:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-novo-1',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Fernanda Lima',
      cliente_telefone: '5511911223344',
      data: dataHoje,
      hora_inicio: '11:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-vip-patricia-1',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Patricia Souza',
      cliente_telefone: '5511933445566',
      data: addMonths(dataHoje, -2),
      hora_inicio: '09:30',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-vip-patricia-2',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Patricia Souza',
      cliente_telefone: '5511933445566',
      data: addMonths(dataHoje, -1),
      hora_inicio: '10:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-sandra-1',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoColor.id,
      adicional_ids: [],
      cliente_nome: 'Sandra Moura',
      cliente_telefone: '5511900112233',
      data: addMonths(dataHoje, -1),
      hora_inicio: '17:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-vip-patricia-3',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Patricia Souza',
      cliente_telefone: '5511933445566',
      data: dataHoje,
      hora_inicio: '15:30',
      status: 'CONFIRMADO',
    },
    {
      id: 'ag-vip-ana-1',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Beatriz Alves',
      cliente_telefone: '5511977665544',
      data: dataHoje,
      hora_inicio: '10:30',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-hoje-c2',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoCorte.id,
      adicional_ids: [],
      cliente_nome: 'Carlos Eduardo',
      cliente_telefone: '5511910101010',
      data: dataHoje,
      hora_inicio: '08:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-hoje-c3',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Ana Paula',
      cliente_telefone: '5511920202020',
      data: dataHoje,
      hora_inicio: '08:30',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-hoje-c4',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoColor.id,
      adicional_ids: [],
      cliente_nome: 'Luciana Ferreira',
      cliente_telefone: '5511930303030',
      data: dataHoje,
      hora_inicio: '09:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-hoje-ns1',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoCorte.id,
      adicional_ids: [],
      cliente_nome: 'Renata Dias',
      cliente_telefone: '5511940404040',
      data: dataHoje,
      hora_inicio: '09:15',
      status: 'CONFIRMADO',
    },
    {
      id: 'ag-hoje-c5',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Clara Nunes',
      cliente_telefone: '5511950505050',
      data: dataHoje,
      hora_inicio: '10:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-hoje-ns2',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Débora Martins',
      cliente_telefone: '5511960606060',
      data: dataHoje,
      hora_inicio: '10:15',
      status: 'CONFIRMADO',
    },
    {
      id: 'ag-hoje-c6',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoCorte.id,
      adicional_ids: [adicionalHidrata.id],
      cliente_nome: 'Beatriz Helena',
      cliente_telefone: '5511970707070',
      data: dataHoje,
      hora_inicio: '11:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-hoje-ns3',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoColor.id,
      adicional_ids: [],
      cliente_nome: 'Elisa Prado',
      cliente_telefone: '5511980808080',
      data: dataHoje,
      hora_inicio: '11:15',
      status: 'CONFIRMADO',
    },
    {
      id: 'ag-hoje-c7',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Gabriela Santos',
      cliente_telefone: '5511990909090',
      data: dataHoje,
      hora_inicio: '12:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-hoje-c8',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoCorte.id,
      adicional_ids: [],
      cliente_nome: 'Helena Moura',
      cliente_telefone: '5511901010101',
      data: dataHoje,
      hora_inicio: '12:30',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-hoje-c9',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Isabela Costa',
      cliente_telefone: '5511911111111',
      data: dataHoje,
      hora_inicio: '13:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-hoje-c10',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoColor.id,
      adicional_ids: [],
      cliente_nome: 'Joana Lima',
      cliente_telefone: '5511921212121',
      data: dataHoje,
      hora_inicio: '13:30',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-hoje-ns4',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Karina Alves',
      cliente_telefone: '5511931313131',
      data: dataHoje,
      hora_inicio: '13:15',
      status: 'CONFIRMADO',
    },
    {
      id: 'ag-hoje-c11',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoCorte.id,
      adicional_ids: [],
      cliente_nome: 'Laura Mendes',
      cliente_telefone: '5511941414141',
      data: dataHoje,
      hora_inicio: '14:30',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-hoje-c12',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Mônica Ribeiro',
      cliente_telefone: '5511951515151',
      data: dataHoje,
      hora_inicio: '15:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-hoje-c13',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoColor.id,
      adicional_ids: [adicionalHidrata.id],
      cliente_nome: 'Natália Souza',
      cliente_telefone: '5511961616161',
      data: dataHoje,
      hora_inicio: '15:45',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-hoje-c14',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Olívia Teixeira',
      cliente_telefone: '5511971717171',
      data: dataHoje,
      hora_inicio: '16:30',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-hoje-c15',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoCorte.id,
      adicional_ids: [],
      cliente_nome: 'Paula Andrade',
      cliente_telefone: '5511981818181',
      data: dataHoje,
      hora_inicio: '17:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-hoje-c16',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Raquel Pires',
      cliente_telefone: '5511991919191',
      data: dataHoje,
      hora_inicio: '17:30',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-hoje-c17',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoColor.id,
      adicional_ids: [],
      cliente_nome: 'Simone Duarte',
      cliente_telefone: '5511902020202',
      data: dataHoje,
      hora_inicio: '18:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-hoje-can1',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Tatiana Rocha',
      cliente_telefone: '5511912121212',
      data: dataHoje,
      hora_inicio: '11:30',
      status: 'CANCELADO',
    },
    {
      id: 'ag-hoje-can2',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoCorte.id,
      adicional_ids: [],
      cliente_nome: 'Vanessa Lins',
      cliente_telefone: '5511922222222',
      data: dataHoje,
      hora_inicio: '14:15',
      status: 'CANCELADO',
    },
    {
      id: 'ag-inat-1',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoColor.id,
      adicional_ids: [],
      cliente_nome: 'Mariana Albuquerque',
      cliente_telefone: '5511933333333',
      data: addDaysISO(dataHoje, -42),
      hora_inicio: '10:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-inat-2',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Beatriz Telles',
      cliente_telefone: '5511944444444',
      data: addDaysISO(dataHoje, -36),
      hora_inicio: '15:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-inat-3',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoCorte.id,
      adicional_ids: [adicionalHidrata.id],
      cliente_nome: 'Camila Lins',
      cliente_telefone: '5511955555555',
      data: addDaysISO(dataHoje, -32),
      hora_inicio: '11:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-inat-4',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Daniela Prado',
      cliente_telefone: '5511966666666',
      data: addDaysISO(dataHoje, -45),
      hora_inicio: '09:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-inat-5',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoColor.id,
      adicional_ids: [],
      cliente_nome: 'Eduarda Nunes',
      cliente_telefone: '5511977777777',
      data: addDaysISO(dataHoje, -38),
      hora_inicio: '14:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-inat-6',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Flávia Borges',
      cliente_telefone: '5511988888888',
      data: addDaysISO(dataHoje, -33),
      hora_inicio: '16:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-inat-7',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoCorte.id,
      adicional_ids: [],
      cliente_nome: 'Gisele Amorim',
      cliente_telefone: '5511999999999',
      data: addDaysISO(dataHoje, -31),
      hora_inicio: '10:30',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-inat-8',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Henrietta Silva',
      cliente_telefone: '5511900000001',
      data: addDaysISO(dataHoje, -35),
      hora_inicio: '11:30',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-inat-9',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoColor.id,
      adicional_ids: [],
      cliente_nome: 'Ingrid Vasconcelos',
      cliente_telefone: '5511900000002',
      data: addDaysISO(dataHoje, -40),
      hora_inicio: '13:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-inat-10',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Juliana Moraes',
      cliente_telefone: '5511900000003',
      data: addDaysISO(dataHoje, -34),
      hora_inicio: '17:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-inat-11',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoCorte.id,
      adicional_ids: [],
      cliente_nome: 'Kelly Fontes',
      cliente_telefone: '5511900000004',
      data: addDaysISO(dataHoje, -37),
      hora_inicio: '08:30',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-inat-12',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Larissa Campos',
      cliente_telefone: '5511900000005',
      data: addDaysISO(dataHoje, -41),
      hora_inicio: '12:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-inat-13',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoColor.id,
      adicional_ids: [],
      cliente_nome: 'Marta Oliveira',
      cliente_telefone: '5511900000006',
      data: addDaysISO(dataHoje, -43),
      hora_inicio: '15:30',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-inat-14',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Nádia Freitas',
      cliente_telefone: '5511900000007',
      data: addDaysISO(dataHoje, -39),
      hora_inicio: '18:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-ontem-c1',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoColor.id,
      adicional_ids: [],
      cliente_nome: 'Cliente Ontem',
      cliente_telefone: '5511900000099',
      data: addDaysISO(dataHoje, -1),
      hora_inicio: '10:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-ontem-c2',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Cliente Ontem 2',
      cliente_telefone: '5511900000098',
      data: addDaysISO(dataHoje, -1),
      hora_inicio: '14:00',
      status: 'CONCLUIDO',
    },
    {
      id: 'ag-amanha-1',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoCorte.id,
      adicional_ids: [],
      cliente_nome: 'Cliente Amanhã',
      cliente_telefone: '5511900000097',
      data: addDaysISO(dataHoje, 1),
      hora_inicio: '09:00',
      status: 'CONFIRMADO',
    },
    {
      id: 'ag-amanha-2',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoColor.id,
      adicional_ids: [],
      cliente_nome: 'Cliente Amanhã 2',
      cliente_telefone: '5511900000096',
      data: addDaysISO(dataHoje, 1),
      hora_inicio: '10:00',
      status: 'CONFIRMADO',
    },
    {
      id: 'ag-amanha-3',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Cliente Amanhã 3',
      cliente_telefone: '5511900000095',
      data: addDaysISO(dataHoje, 1),
      hora_inicio: '11:00',
      status: 'CONFIRMADO',
    },
    {
      id: 'ag-amanha-4',
      tenant_id: tenantGlow.id,
      profissional_id: profAna.id,
      servico_id: servicoManicure.id,
      adicional_ids: [],
      cliente_nome: 'Cliente Amanhã 4',
      cliente_telefone: '5511900000094',
      data: addDaysISO(dataHoje, 1),
      hora_inicio: '14:00',
      status: 'CONFIRMADO',
    },
    {
      id: 'ag-amanha-5',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      servico_id: servicoCorte.id,
      adicional_ids: [adicionalHidrata.id],
      cliente_nome: 'Cliente Amanhã 5',
      cliente_telefone: '5511900000093',
      data: addDaysISO(dataHoje, 1),
      hora_inicio: '15:00',
      status: 'CONFIRMADO',
    },
  ]

  const fila_espera: FilaEspera[] = [
    {
      id: 'fila-1',
      tenant_id: tenantGlow.id,
      profissional_id: profClaudia.id,
      cliente_nome: 'Camila Rocha',
      cliente_telefone: '5511966554433',
      status: 'AGUARDANDO',
    },
  ]

  const lancamentos: Lancamento[] = buildFinanceiroLancamentos(
    tenantGlow.id,
    dataHoje,
    profClaudia.id,
  )

  const clientes: Cliente[] = [
    {
      id: 'cli-mariana',
      tenant_id: tenantGlow.id,
      nome: 'Mariana Silva',
      telefone: '5511999887766',
      email: 'mariana@email.com',
      ativo: true,
      criado_em: addMonths(today(), -14),
      data_nascimento: '1992-03-15',
      foto_url:
        'https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=200&q=80',
      alergias: ['Parabenos'],
      observacoes_medicas:
        'Pele sensível com tendência a rosácea. Evitar produtos com álcool cetílico.',
      preferencias: [
        'Café sem açúcar',
        'Prefere silêncio',
        'Ar condicionado (22°C)',
        'Revistas de moda',
      ],
    },
    {
      id: 'cli-juliana',
      tenant_id: tenantGlow.id,
      nome: 'Juliana Costa',
      telefone: '5511988776655',
      email: 'juliana@email.com',
      ativo: true,
      criado_em: addMonths(today(), -1),
    },
    {
      id: 'cli-beatriz',
      tenant_id: tenantGlow.id,
      nome: 'Beatriz Alves',
      telefone: '5511977665544',
      ativo: true,
      criado_em: dataHoje,
    },
  ]

  const tenants = [tenantGlow, tenantVencido]
  const planos = [planoEssencial, planoProfissional, planoElite]

  const cliente_notas: ClienteNotaInterna[] = [
    {
      id: 'nota-mariana-1',
      cliente_id: 'cli-mariana',
      autor: 'Cláudia Mendes',
      data: addMonths(today(), -1),
      texto:
        'Cliente relatou leve desconforto com vapor intenso no último procedimento. Reduzi a intensidade e ela se sentiu melhor. Recomendei protetor solar FPS 60.',
    },
  ]

  const cliente_galeria: ClienteGaleriaItem[] = [
    {
      id: 'gal-mariana-1',
      cliente_id: 'cli-mariana',
      url: 'https://images.unsplash.com/photo-1570172619644-dfd03ed5d881?w=400&q=80',
      legenda: 'Antes — coloração',
      data: addMonths(today(), -2),
    },
    {
      id: 'gal-mariana-2',
      cliente_id: 'cli-mariana',
      url: 'https://images.unsplash.com/photo-1522337360788-8b13dee7a37e?w=400&q=80',
      legenda: 'Depois — coloração',
      data: addMonths(today(), -2),
    },
  ]

  const insumos: import('../types').InsumoEstoque[] = buildInsumosCatalog(tenantGlow.id)

  return {
    planos,
    tenants,
    faturas_saas: buildFaturasSaas(tenants, planos),
    categorias: [catCabelo, catUnhas, catMassagem, catEstetica, catDepilacao],
    clientes,
    cliente_notas,
    cliente_galeria,
    especialidades: [espCabelo, espUnhas, espEstetica],
    profissionais: [
      profClaudia,
      profAna,
      ...buildEquipeCatalog(tenantGlow.id, espCabelo.id, espUnhas.id, espEstetica.id),
    ],
    servicos,
    insumos,
    adicionais: [adicionalHidrata],
    agendamentos,
    fila_espera,
    lancamentos,
    contas_cliente: [
      {
        id: 'conta-mariana',
        telefone: '5511999887766',
        email: 'mariana@email.com',
        password: 'AgendaGlow@2026',
        nome: 'Mariana Silva',
        criado_em: addMonths(today(), -3),
      },
    ],
    cliente_preferencias: [
      {
        conta_cliente_id: 'conta-mariana',
        tenant_id: tenantGlow.id,
        profissional_favorito_id: profClaudia.id,
      },
    ],
    users: [
      {
        id: 'user-sa',
        email: 'ferrariwill@gmail.com',
        password: 'AgendaGlow@2026',
        role: 'SUPER_ADMIN',
        nome: 'Super Admin',
      },
      {
        id: 'user-dona',
        email: 'dona@glow.local',
        password: 'AgendaGlow@2026',
        role: 'DONA',
        tenant_id: tenantGlow.id,
        nome: 'Dona Glow',
      },
      {
        id: 'user-prof',
        email: 'claudia@glow.local',
        password: 'AgendaGlow@2026',
        role: 'PROFISSIONAL',
        tenant_id: tenantGlow.id,
        profissional_id: profClaudia.id,
        nome: 'Cláudia Mendes',
      },
      {
        id: 'user-secretaria',
        email: 'secretaria@glow.local',
        password: 'AgendaGlow@2026',
        role: 'SECRETARIA',
        tenant_id: tenantGlow.id,
        nome: 'Marina Secretária',
      },
    ],
  }
}

function loadRaw(): MockDatabase {
  try {
    const seedV = localStorage.getItem(SEED_VERSION_KEY)
    if (seedV !== String(SEED_VERSION)) {
      localStorage.removeItem(STORAGE_KEY)
      localStorage.setItem(SEED_VERSION_KEY, String(SEED_VERSION))
    }
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) {
      const parsed = JSON.parse(raw) as MockDatabase
      parsed.tenants = parsed.tenants.map((t) => ({
        ...t,
        data_vencimento: t.data_vencimento ?? addMonths(today(), 12),
        criado_em: t.criado_em ?? today(),
      }))
      if (!parsed.categorias) parsed.categorias = []
      if (!parsed.clientes) {
        const seen = new Set<string>()
        parsed.clientes = []
        for (const ag of parsed.agendamentos ?? []) {
          const key = `${ag.tenant_id}:${ag.cliente_telefone}`
          if (seen.has(key)) continue
          seen.add(key)
          parsed.clientes.push({
            id: `cli-${ag.cliente_telefone}`,
            tenant_id: ag.tenant_id,
            nome: ag.cliente_nome,
            telefone: ag.cliente_telefone,
            ativo: true,
            criado_em: ag.data,
          })
        }
      }
      parsed.agendamentos = (parsed.agendamentos ?? []).map((ag) => ({
        ...ag,
        servico_ids: ag.servico_ids?.length ? ag.servico_ids : [ag.servico_id],
      }))
      if (!parsed.contas_cliente) parsed.contas_cliente = []
      if (!parsed.cliente_preferencias) parsed.cliente_preferencias = []
      if (!parsed.faturas_saas?.length) {
        parsed.faturas_saas = buildFaturasSaas(parsed.tenants, parsed.planos)
      }
      if (!parsed.cliente_notas) parsed.cliente_notas = []
      if (!parsed.cliente_galeria) parsed.cliente_galeria = []
      if (!parsed.insumos) parsed.insumos = []
      return parsed
    }
  } catch {
    /* reset on corrupt data */
  }
  const seeded = seedDatabase()
  saveRaw(seeded)
  return seeded
}

function saveRaw(db: MockDatabase): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(db))
}

let cache: MockDatabase | null = null

export function getDb(): MockDatabase {
  if (!IS_MOCK) return getStoreDb()
  if (!cache) cache = loadRaw()
  return cache
}

export function persistDb(db: MockDatabase): void {
  if (!IS_MOCK) {
    setStoreDb(db)
    return
  }
  cache = db
  saveRaw(db)
}

export function resetDb(): void {
  localStorage.removeItem(STORAGE_KEY)
  cache = null
}

// ——— Helpers ———

export function timeToMinutes(t: string): number {
  const [h, m] = t.split(':').map(Number)
  return h * 60 + m
}

export function minutesToTime(mins: number): string {
  const h = Math.floor(mins / 60)
  const m = mins % 60
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`
}

export function getAgendamentoServicoIds(ag: Agendamento): string[] {
  if (ag.servico_ids?.length) return ag.servico_ids
  return [ag.servico_id]
}

export function getAgendamentoServicosNomes(ag: Agendamento, db: MockDatabase): string {
  return getAgendamentoServicoIds(ag)
    .map((id) => db.servicos.find((s) => s.id === id)?.nome)
    .filter(Boolean)
    .join(' + ') || '—'
}

export function getAgendamentoValor(ag: Agendamento, db: MockDatabase): number {
  let total = 0
  for (const sid of getAgendamentoServicoIds(ag)) {
    total += db.servicos.find((s) => s.id === sid)?.preco ?? 0
  }
  for (const aid of ag.adicional_ids) {
    total += db.adicionais.find((a) => a.id === aid)?.preco ?? 0
  }
  return total
}

export function getAgendamentoDuration(ag: Agendamento, db: MockDatabase): number {
  let total = 0
  for (const sid of getAgendamentoServicoIds(ag)) {
    const servico = db.servicos.find((s) => s.id === sid)
    total += servico?.duracao_minutos ?? 0
  }
  for (const aid of ag.adicional_ids) {
    const add = db.adicionais.find((a) => a.id === aid)
    if (add) total += add.duracao_minutos
  }
  return total || 30
}

export function getTenantBySlug(slug: string): Tenant | undefined {
  if (!IS_MOCK) return getStoreDb().tenants.find((t) => t.slug === slug)
  return getDb().tenants.find((t) => t.slug === slug)
}

export function getPlano(id: string): PlanoSaas | undefined {
  return getDb().planos.find((p) => p.id === id)
}

function assertPlanoNomeUnico(db: MockDatabase, nome: string, excludeId?: string) {
  const n = nome.trim().toLowerCase()
  if (db.planos.some((p) => p.nome.trim().toLowerCase() === n && p.id !== excludeId)) {
    throw new Error('Já existe um plano com este nome')
  }
}

export async function createPlanoSaas(data: Omit<PlanoSaas, 'id'>): Promise<PlanoSaas> {
  if (!IS_MOCK) {
    const { id } = await apiFetch<{ id: string }>('/api/v1/admin/plans', {
      method: 'POST',
      body: JSON.stringify({
        nome: data.nome,
        preco_mensal: data.preco_mensal,
        limite_profissionais: data.limite_profissionais,
      }),
    })
    await syncAdminBootstrap()
    return getDb().planos.find((p) => p.id === id) ?? { ...data, id }
  }

  const db = getDb()
  const nome = data.nome.trim()
  if (!nome) throw new Error('Nome do plano é obrigatório')
  if (data.preco_mensal < 0) throw new Error('Preço mensal deve ser ≥ 0')
  if (data.limite_profissionais < 1) throw new Error('Limite de profissionais deve ser maior que zero')
  assertPlanoNomeUnico(db, nome)
  const plano: PlanoSaas = { ...data, nome, id: uid() }
  db.planos.push(plano)
  persistDb(db)
  return plano
}

export async function updatePlanoSaas(id: string, patch: Partial<Omit<PlanoSaas, 'id'>>): Promise<PlanoSaas> {
  if (!IS_MOCK) {
    const current = getDb().planos.find((p) => p.id === id)
    if (!current) throw new Error('Plano não encontrado')
    const merged = { ...current, ...patch }
    await apiFetch(`/api/v1/admin/plans/${id}`, {
      method: 'PUT',
      body: JSON.stringify({
        nome: merged.nome,
        preco_mensal: merged.preco_mensal,
        limite_profissionais: merged.limite_profissionais,
        ativo: merged.ativo,
      }),
    })
    await syncAdminBootstrap()
    return getDb().planos.find((p) => p.id === id) ?? merged
  }

  const db = getDb()
  const idx = db.planos.findIndex((p) => p.id === id)
  if (idx < 0) throw new Error('Plano não encontrado')
  const current = db.planos[idx]
  const nome = patch.nome !== undefined ? patch.nome.trim() : current.nome
  if (!nome) throw new Error('Nome do plano é obrigatório')
  const preco = patch.preco_mensal ?? current.preco_mensal
  const limite = patch.limite_profissionais ?? current.limite_profissionais
  if (preco < 0) throw new Error('Preço mensal deve ser ≥ 0')
  if (limite < 1) throw new Error('Limite de profissionais deve ser maior que zero')
  assertPlanoNomeUnico(db, nome, id)
  db.planos[idx] = {
    ...current,
    ...patch,
    nome,
    preco_mensal: preco,
    limite_profissionais: limite,
  }
  persistDb(db)
  return db.planos[idx]
}

export function deletePlanoSaas(id: string): void {
  const db = getDb()
  if (db.planos.length <= 1) throw new Error('Deve existir ao menos um plano na plataforma')
  if (db.tenants.some((t) => t.plano_id === id)) {
    throw new Error('Não é possível excluir: existem salões vinculados a este plano')
  }
  const idx = db.planos.findIndex((p) => p.id === id)
  if (idx < 0) throw new Error('Plano não encontrado')
  db.planos.splice(idx, 1)
  persistDb(db)
}

export function countTenantsByPlano(planoId: string): number {
  return getDb().tenants.filter((t) => t.plano_id === planoId).length
}

export function getProfissionalById(tenantId: string, id: string): Profissional | undefined {
  return getDb().profissionais.find((p) => p.id === id && p.tenant_id === tenantId)
}

export function countActiveProfessionals(tenantId: string): number {
  return getDb().profissionais.filter((p) => p.tenant_id === tenantId && p.ativo).length
}

export function getEquipeStats(tenantId: string) {
  const profs = getDb().profissionais.filter((p) => p.tenant_id === tenantId)
  return {
    total: profs.length,
    ativos: profs.filter((p) => p.ativo && !p.pendente_aprovacao).length,
    pendentes: profs.filter((p) => p.pendente_aprovacao).length,
  }
}

export function getProfissionalEmail(prof: Profissional): string {
  if (prof.email) return prof.email
  const db = getDb()
  const user = db.users.find(
    (u) => u.profissional_id === prof.id || u.id === prof.user_id,
  )
  if (user?.email) return user.email
  const slug = prof.nome
    .toLowerCase()
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/\s+/g, '.')
  return `${slug}@salao.local`
}

export function profissionalStatusLabel(prof: Profissional): 'ATIVO' | 'INATIVO' | 'PENDENTE' {
  if (prof.pendente_aprovacao) return 'PENDENTE'
  return prof.ativo ? 'ATIVO' : 'INATIVO'
}

export async function createProfissional(
  data: Omit<Profissional, 'id'>,
): Promise<Profissional> {
  if (!IS_MOCK) {
    const { id } = await apiFetch<{ id: string }>('/api/v1/professionals', {
      method: 'POST',
      body: JSON.stringify({
        nome: data.nome,
        especialidade_id: data.especialidade_id,
        comissao_porcentagem: data.comissao_percent,
      }),
    })
    await refreshAfterMutation()
    const prof = getDb().profissionais.find((p) => p.id === id)
    if (prof) return prof
    return { ...data, id }
  }

  const db = getDb()
  const tenant = db.tenants.find((t) => t.id === data.tenant_id)
  if (!tenant) throw new Error('Tenant não encontrado')
  const plano = getPlano(tenant.plano_id)
  if (!plano) throw new Error('Plano não encontrado')
  const ativos = countActiveProfessionals(data.tenant_id)
  if (data.ativo && ativos >= plano.limite_profissionais) {
    throw new PlanLimitExceededError()
  }
  const prof: Profissional = { ...data, id: uid() }
  db.profissionais.push(prof)
  persistDb(db)
  return prof
}

export async function updateProfissional(id: string, patch: Partial<Profissional>): Promise<Profissional> {
  if (!IS_MOCK) {
    const current = getDb().profissionais.find((p) => p.id === id)
    if (!current) throw new Error('Profissional não encontrado')
    const merged = { ...current, ...patch }
    await apiFetch(`/api/v1/professionals/${id}`, {
      method: 'PUT',
      body: JSON.stringify({
        nome: merged.nome,
        especialidade_id: merged.especialidade_id,
        comissao_porcentagem: merged.comissao_percent,
        ativo: merged.ativo,
      }),
    })
    await refreshAfterMutation()
    return getDb().profissionais.find((p) => p.id === id) ?? merged
  }

  const db = getDb()
  const idx = db.profissionais.findIndex((p) => p.id === id)
  if (idx < 0) throw new Error('Profissional não encontrado')
  const current = db.profissionais[idx]
  if (patch.ativo && !current.ativo) {
    const tenant = db.tenants.find((t) => t.id === current.tenant_id)
    const plano = tenant ? getPlano(tenant.plano_id) : undefined
    if (plano && countActiveProfessionals(current.tenant_id) >= plano.limite_profissionais) {
      throw new PlanLimitExceededError()
    }
  }
  const updated = { ...current, ...patch, id }
  db.profissionais[idx] = updated
  persistDb(db)
  return updated
}

export async function updateAgendamentoStatus(
  id: string,
  status: AgendamentoStatus,
  opts?: { role?: import('../types').UserRole },
): Promise<Agendamento> {
  if (!IS_MOCK) {
    if (status === 'CONCLUIDO') {
      const path =
        opts?.role === 'PROFISSIONAL'
          ? `/api/v1/professional/appointments/${id}/complete`
          : `/api/v1/appointments/${id}/charge`
      await apiFetch(path, { method: 'POST' })
      await refreshAfterMutation(opts?.role)
    } else if (status === 'CANCELADO') {
      await apiFetch(`/api/v1/appointments/${id}/cancel`, { method: 'POST' })
      await refreshAfterMutation(opts?.role)
    }
    const ag = getDb().agendamentos.find((a) => a.id === id)
    if (ag) return { ...ag, status }
    throw new Error('Agendamento não encontrado')
  }

  const db = getDb()
  const idx = db.agendamentos.findIndex((a) => a.id === id)
  if (idx < 0) throw new Error('Agendamento não encontrado')
  db.agendamentos[idx] = { ...db.agendamentos[idx], status }
  persistDb(db)
  return db.agendamentos[idx]
}

export async function cancelAgendamento(id: string): Promise<Agendamento> {
  return updateAgendamentoStatus(id, 'CANCELADO')
}

export class AgendaConflitoError extends Error {
  constructor(message = 'Horário indisponível — conflito na agenda') {
    super(message)
    this.name = 'AgendaConflitoError'
  }
}

function agendamentosBloqueantes(
  db: MockDatabase,
  profissionalId: string,
  data: string,
  excludeId?: string,
): Agendamento[] {
  return db.agendamentos.filter(
    (a) =>
      a.profissional_id === profissionalId &&
      a.data === data &&
      a.status !== 'CANCELADO' &&
      a.status !== 'EM_APROVACAO' &&
      a.id !== excludeId,
  )
}

export function avaliarNovoAgendamento(
  profissionalId: string,
  data: string,
  horaInicio: string,
  duracaoMinutos: number,
  excludeAgendamentoId?: string,
): { status: AgendamentoStatus; minutos_invadidos?: number } {
  const db = getDb()
  const inicio = timeToMinutes(horaInicio)
  const fim = inicio + duracaoMinutos
  const bloqueantes = agendamentosBloqueantes(db, profissionalId, data, excludeAgendamentoId)

  for (const ag of bloqueantes) {
    const agInicio = timeToMinutes(ag.hora_inicio)
    const agFim = agInicio + getAgendamentoDuration(ag, db)
    if (inicio >= agFim || fim <= agInicio) continue
    if (inicio >= agInicio && fim <= agFim) {
      throw new AgendaConflitoError('Horário totalmente ocupado')
    }
    if (inicio < agFim && fim > agFim && inicio >= agInicio) {
      const invadidos = fim - agFim
      if (invadidos >= 1) {
        return { status: 'EM_APROVACAO', minutos_invadidos: invadidos }
      }
    }
    throw new AgendaConflitoError('Conflito com outro agendamento')
  }
  return { status: 'CONFIRMADO' }
}

export async function createAgendamento(
  input: {
    tenant_id: string
    profissional_id: string
    servico_ids: string[]
    adicional_ids?: string[]
    cliente_nome: string
    cliente_telefone: string
    data: string
    hora_inicio: string
    observacoes?: string
    forcar_status?: AgendamentoStatus
    aceita_adiantar?: boolean
  },
  opts?: { publicSlug?: string },
): Promise<Agendamento> {
  if (!IS_MOCK) {
    const servicoId = input.servico_ids[0]
    const path = opts?.publicSlug
      ? `/api/v1/public/${opts.publicSlug}/appointments`
      : '/api/v1/appointments'
    const result = await apiFetch<{ id: string; status: string; management_url?: string }>(path, {
      method: 'POST',
      body: JSON.stringify({
        cliente_nome: input.cliente_nome,
        cliente_telefone: input.cliente_telefone,
        profissional_id: input.profissional_id,
        servico_id: servicoId,
        adicional_ids: input.adicional_ids ?? [],
        data: input.data,
        hora_inicio: input.hora_inicio.slice(0, 5),
        aceita_adiantar: input.aceita_adiantar,
      }),
    })
    await refreshAfterMutation()
    const ag = getDb().agendamentos.find((a) => a.id === result.id)
    if (ag) return { ...ag, management_url: result.management_url }
    return {
      id: result.id,
      tenant_id: input.tenant_id,
      profissional_id: input.profissional_id,
      servico_id: servicoId,
      servico_ids: input.servico_ids,
      adicional_ids: input.adicional_ids ?? [],
      cliente_nome: input.cliente_nome,
      cliente_telefone: input.cliente_telefone.replace(/\D/g, ''),
      data: input.data,
      hora_inicio: input.hora_inicio,
      status: result.status as AgendamentoStatus,
      management_url: result.management_url,
    }
  }

  const ids = [...new Set(input.servico_ids.filter(Boolean))]
  if (ids.length === 0) throw new AgendaConflitoError('Selecione ao menos um serviço')

  const db = getDb()
  const draft: Agendamento = {
    id: uid(),
    tenant_id: input.tenant_id,
    profissional_id: input.profissional_id,
    servico_id: ids[0],
    servico_ids: ids,
    adicional_ids: input.adicional_ids ?? [],
    cliente_nome: input.cliente_nome,
    cliente_telefone: input.cliente_telefone.replace(/\D/g, ''),
    data: input.data,
    hora_inicio: input.hora_inicio,
    status: 'AGENDADO',
    observacoes: input.observacoes,
    aceita_adiantar: input.aceita_adiantar,
    management_url: opts?.publicSlug ? '/p/agendamento/mock-valid' : undefined,
  }
  const duracao = getAgendamentoDuration(draft, db)
  const avaliacao = input.forcar_status
    ? { status: input.forcar_status }
    : avaliarNovoAgendamento(input.profissional_id, input.data, input.hora_inicio, duracao)
  draft.status = avaliacao.status
  if (avaliacao.minutos_invadidos) draft.minutos_invadidos = avaliacao.minutos_invadidos

  db.agendamentos.push(draft)
  if (input.aceita_adiantar) {
    inscreverFilaEspera({
      tenant_id: input.tenant_id,
      profissional_id: input.profissional_id,
      cliente_nome: input.cliente_nome,
      cliente_telefone: input.cliente_telefone,
      agendamento_id: draft.id,
      data: input.data,
      hora: input.hora_inicio,
    })
  }
  persistDb(db)
  return draft
}

export function updateAgendamento(
  id: string,
  patch: Partial<
    Pick<
      Agendamento,
      | 'data'
      | 'hora_inicio'
      | 'servico_ids'
      | 'servico_id'
      | 'cliente_nome'
      | 'cliente_telefone'
      | 'observacoes'
      | 'status'
      | 'adicional_ids'
      | 'profissional_id'
    >
  >,
): Agendamento {
  const db = getDb()
  const idx = db.agendamentos.findIndex((a) => a.id === id)
  if (idx < 0) throw new AgendaConflitoError('Agendamento não encontrado')

  const current = db.agendamentos[idx]
  const servicoIds = patch.servico_ids?.length
    ? [...new Set(patch.servico_ids.filter(Boolean))]
    : current.servico_ids?.length
      ? current.servico_ids
      : [current.servico_id]

  const merged: Agendamento = {
    ...current,
    ...patch,
    servico_ids: servicoIds,
    servico_id: servicoIds[0],
  }

  if (
    patch.data !== undefined ||
    patch.hora_inicio !== undefined ||
    patch.servico_ids !== undefined ||
    patch.profissional_id !== undefined
  ) {
    const duracao = getAgendamentoDuration(merged, db)
    const avaliacao = avaliarNovoAgendamento(
      merged.profissional_id,
      merged.data,
      merged.hora_inicio,
      duracao,
      id,
    )
    merged.status = avaliacao.status
    merged.minutos_invadidos = avaliacao.minutos_invadidos
  }

  db.agendamentos[idx] = merged
  persistDb(db)
  return merged
}

export function calcValorAgendamento(ag: Agendamento, db?: MockDatabase): number {
  const data = db ?? getDb()
  return getAgendamentoServicoIds(ag).reduce((s, sid) => {
    const srv = data.servicos.find((x) => x.id === sid)
    return s + (srv?.preco ?? 0)
  }, 0)
}

export async function registrarCobrancaAgendamento(
  agendamentoId: string,
  input: {
    metodo: MetodoPagamento
    valor?: number
    concluir?: boolean
  },
): Promise<Agendamento> {
  if (!IS_MOCK) {
    await apiFetch(`/api/v1/appointments/${agendamentoId}/charge`, {
      method: 'POST',
      body: JSON.stringify({
        metodo: input.metodo,
        valor: input.valor,
        concluir: input.concluir !== false,
      }),
    })
    await refreshAfterMutation()
    const updated = getDb().agendamentos.find((a) => a.id === agendamentoId)
    if (updated) return updated
    throw new Error('Agendamento não encontrado')
  }

  const db = getDb()
  const idx = db.agendamentos.findIndex((a) => a.id === agendamentoId)
  if (idx < 0) throw new Error('Agendamento não encontrado')
  const ag = db.agendamentos[idx]
  if (ag.cobrado_em) throw new Error('Este procedimento já foi cobrado.')

  const valor = input.valor ?? calcValorAgendamento(ag, db)
  if (valor <= 0) throw new Error('Valor inválido para cobrança.')

  const hoje = today()
  db.lancamentos.push({
    id: uid(),
    tenant_id: ag.tenant_id,
    tipo: 'ENTRADA',
    valor: Math.round(valor * 100) / 100,
    descricao: `Cobrança — ${getAgendamentoServicosNomes(ag, db)} (${ag.cliente_nome})`,
    subtitulo: `${ag.data} às ${ag.hora_inicio.slice(0, 5)}`,
    data: hoje,
    categoria: 'SERVICO',
    status_transacao: 'PAGO',
    metodo_pagamento: input.metodo,
  })

  const concluir = input.concluir !== false
  db.agendamentos[idx] = {
    ...ag,
    valor_cobrado: valor,
    metodo_pagamento: input.metodo,
    cobrado_em: hoje,
    status: concluir ? 'CONCLUIDO' : ag.status,
  }

  if (concluir) {
    registrarComissaoAgendamento(agendamentoId, ag.profissional_id, ag.tenant_id)
  }

  persistDb(db)
  return db.agendamentos[idx]
}

export interface ProfissionalComissaoItem {
  id: string
  descricao: string
  valor: number
  data: string
  status: 'PENDENTE' | 'LIQUIDADO'
}

export interface ProfissionalComissoesResumo {
  mesReferencia: string
  totalMes: number
  pendente: number
  liquidado: number
  atendimentosConcluidos: number
  atendimentosHoje: number
  comissaoHoje: number
  historico: ProfissionalComissaoItem[]
}

function comissaoAgendamento(ag: Agendamento, db: MockDatabase, profId: string): number {
  const prof = db.profissionais.find((p) => p.id === profId)
  const pct = prof?.comissao_percent ?? 40
  const preco = getAgendamentoServicoIds(ag).reduce((s, sid) => {
    const srv = db.servicos.find((x) => x.id === sid)
    return s + (srv?.preco ?? 0)
  }, 0)
  return Math.round((preco * pct) / 100 * 100) / 100
}

export function getProfissionalComissoesResumo(
  profId: string,
  tenantId: string,
  yearMonth?: string,
): ProfissionalComissoesResumo {
  const db = getDb()
  const ym = yearMonth ?? new Date().toISOString().slice(0, 7)
  const hoje = today()

  const lancs = db.lancamentos.filter(
    (l) =>
      l.tenant_id === tenantId &&
      l.profissional_id === profId &&
      l.tipo === 'CUSTO_VARIAVEL' &&
      l.data.slice(0, 7) === ym,
  )

  const pendente = lancs
    .filter((l) => l.status_pagamento === 'PENDENTE')
    .reduce((s, l) => s + l.valor, 0)
  const liquidado = lancs
    .filter((l) => l.status_pagamento === 'LIQUIDADO')
    .reduce((s, l) => s + l.valor, 0)

  const atendimentosConcluidos = db.agendamentos.filter(
    (a) =>
      a.profissional_id === profId &&
      a.tenant_id === tenantId &&
      a.status === 'CONCLUIDO' &&
      a.data.slice(0, 7) === ym,
  ).length

  const hojeAgs = db.agendamentos.filter(
    (a) =>
      a.profissional_id === profId &&
      a.tenant_id === tenantId &&
      a.data === hoje &&
      a.status !== 'CANCELADO',
  )
  const atendimentosHoje = hojeAgs.filter((a) => a.status === 'CONCLUIDO').length
  const comissaoHoje = hojeAgs
    .filter((a) => a.status === 'CONCLUIDO')
    .reduce((s, ag) => s + comissaoAgendamento(ag, db, profId), 0)

  const historico = lancs
    .map((l) => ({
      id: l.id,
      descricao: l.descricao,
      valor: l.valor,
      data: l.data,
      status: (l.status_pagamento === 'LIQUIDADO' ? 'LIQUIDADO' : 'PENDENTE') as
        | 'PENDENTE'
        | 'LIQUIDADO',
    }))
    .sort((a, b) => b.data.localeCompare(a.data))

  return {
    mesReferencia: ym,
    totalMes: pendente + liquidado,
    pendente,
    liquidado,
    atendimentosConcluidos,
    atendimentosHoje,
    comissaoHoje,
    historico,
  }
}

export function registrarComissaoAgendamento(
  agendamentoId: string,
  profId: string,
  tenantId: string,
): void {
  const db = getDb()
  const ag = db.agendamentos.find((a) => a.id === agendamentoId)
  if (!ag || ag.status !== 'CONCLUIDO') return
  const exists = db.lancamentos.some(
    (l) =>
      l.profissional_id === profId &&
      l.tenant_id === tenantId &&
      l.descricao.includes(ag.cliente_nome) &&
      l.data === ag.data,
  )
  if (exists) return
  const valor = comissaoAgendamento(ag, db, profId)
  db.lancamentos.push({
    id: uid(),
    tenant_id: tenantId,
    tipo: 'CUSTO_VARIAVEL',
    valor,
    descricao: `Comissão — ${getAgendamentoServicosNomes(ag, db)} (${ag.cliente_nome})`,
    data: ag.data,
    profissional_id: profId,
    status_pagamento: 'PENDENTE',
    categoria: 'SALARIOS',
    status_transacao: 'PENDENTE',
  })
  persistDb(db)
}

export function getDonaProfissional(tenantId: string): Profissional | undefined {
  return getDb().profissionais.find((p) => p.tenant_id === tenantId && p.eh_dona)
}

export function setDonaComoProfissional(
  tenantId: string,
  userId: string,
  opts: { especialidade_id: string; comissao_percent: number; expedientes: Profissional['expedientes'] },
): Profissional {
  const db = getDb()
  const userIdx = db.users.findIndex((u) => u.id === userId)
  if (userIdx < 0) throw new Error('Usuário não encontrado')
  const user = db.users[userIdx]
  const tenantIdx = db.tenants.findIndex((t) => t.id === tenantId)
  if (tenantIdx < 0) throw new Error('Salão não encontrado')

  let prof = db.profissionais.find((p) => p.tenant_id === tenantId && p.eh_dona)
  if (prof) {
    const idx = db.profissionais.findIndex((p) => p.id === prof!.id)
    db.profissionais[idx] = {
      ...prof,
      ativo: true,
      especialidade_id: opts.especialidade_id,
      comissao_percent: opts.comissao_percent,
      expedientes: opts.expedientes,
      user_id: userId,
      eh_dona: true,
      nome: user.nome,
    }
    prof = db.profissionais[idx]
  } else {
    const ativos = countActiveProfessionals(tenantId)
    const plano = getPlano(db.tenants[tenantIdx].plano_id)
    if (plano && ativos >= plano.limite_profissionais) {
      throw new PlanLimitExceededError()
    }
    prof = {
      id: uid(),
      tenant_id: tenantId,
      nome: user.nome,
      especialidade_id: opts.especialidade_id,
      comissao_percent: opts.comissao_percent,
      ativo: true,
      expedientes: opts.expedientes,
      user_id: userId,
      eh_dona: true,
    }
    db.profissionais.push(prof)
  }

  db.users[userIdx].profissional_id = prof.id
  db.tenants[tenantIdx].dona_atua_como_profissional = true
  persistDb(db)
  return prof
}

export function unsetDonaComoProfissional(tenantId: string, userId: string): void {
  const db = getDb()
  const prof = db.profissionais.find((p) => p.tenant_id === tenantId && p.eh_dona)
  if (prof) {
    const idx = db.profissionais.findIndex((p) => p.id === prof.id)
    db.profissionais[idx].ativo = false
  }
  const userIdx = db.users.findIndex((u) => u.id === userId)
  if (userIdx >= 0) db.users[userIdx].profissional_id = undefined
  const tenantIdx = db.tenants.findIndex((t) => t.id === tenantId)
  if (tenantIdx >= 0) db.tenants[tenantIdx].dona_atua_como_profissional = false
  persistDb(db)
}

export function getFilaAguardando(profissionalId: string): FilaEspera | undefined {
  return getDb().fila_espera.find(
    (f) => f.profissional_id === profissionalId && f.status === 'AGUARDANDO',
  )
}

export function inscreverFilaEspera(input: {
  tenant_id: string
  profissional_id: string
  cliente_nome: string
  cliente_telefone: string
  agendamento_id?: string
  data?: string
  hora?: string
}): FilaEspera {
  const db = getDb()
  const tel = normalizeTelefone(input.cliente_telefone)
  db.fila_espera = db.fila_espera.filter(
    (f) =>
      !(
        f.profissional_id === input.profissional_id &&
        f.cliente_telefone === tel &&
        f.status === 'AGUARDANDO'
      ),
  )
  const entry: FilaEspera = {
    id: uid(),
    tenant_id: input.tenant_id,
    profissional_id: input.profissional_id,
    cliente_nome: input.cliente_nome.trim(),
    cliente_telefone: tel,
    status: 'AGUARDANDO',
    agendamento_id: input.agendamento_id,
    data: input.data,
    hora_desejada: input.hora,
  }
  db.fila_espera.push(entry)
  persistDb(db)
  return entry
}

export function getServicosDoProfissional(
  tenantId: string,
  profissionalId: string,
  servicos?: import('../types').Servico[],
): import('../types').Servico[] {
  const list = servicos ?? getDb().servicos
  return list.filter(
    (s) =>
      s.tenant_id === tenantId &&
      s.ativo &&
      s.permitir_agendamento_online !== false &&
      (!s.profissional_ids?.length || s.profissional_ids.includes(profissionalId)),
  )
}

export function markFilaNotificada(id: string): void {
  const db = getDb()
  const idx = db.fila_espera.findIndex((f) => f.id === id)
  if (idx >= 0) {
    db.fila_espera[idx].status = 'NOTIFICADO'
    persistDb(db)
  }
}

export function liquidarComissoesPendentes(tenantId: string, profissionalId?: string): number {
  const db = getDb()
  const hoje = formatNowBR()
  let count = 0
  for (const l of db.lancamentos) {
    if (
      l.tenant_id === tenantId &&
      l.tipo === 'CUSTO_VARIAVEL' &&
      l.status_pagamento === 'PENDENTE' &&
      (!profissionalId || l.profissional_id === profissionalId)
    ) {
      l.status_pagamento = 'LIQUIDADO'
      l.descricao = `${l.descricao} | LIQUIDADO em ${hoje}`
      l.liquidado_em = hoje
      count++
    }
  }
  if (count > 0) persistDb(db)
  return count
}

export class LancamentoValidationError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'LancamentoValidationError'
  }
}

export function getLancamentoById(tenantId: string, id: string): Lancamento | undefined {
  return getDb().lancamentos.find((l) => l.id === id && l.tenant_id === tenantId)
}

export async function createLancamento(input: {
  tenant_id: string
  tipo: LancamentoTipo
  valor: number
  descricao: string
  data?: string
  categoria?: import('../types').TransacaoCategoria
  subtitulo?: string
  status_transacao?: import('../types').TransacaoStatus
  vencimento?: string
  profissional_id?: string
  metodo_pagamento?: import('../types').MetodoPagamento
  fornecedor?: string
  natureza?: import('../types').NaturezaValor
  recorrente?: boolean
  frequencia_recorrencia?: import('../types').FrequenciaRecorrencia
}): Promise<Lancamento> {
  const descricao = input.descricao.trim()
  if (!descricao) throw new LancamentoValidationError('Descrição é obrigatória')
  if (!Number.isFinite(input.valor) || input.valor <= 0) {
    throw new LancamentoValidationError('Valor deve ser maior que zero')
  }
  if (!['ENTRADA', 'CUSTO_FIXO', 'CUSTO_VARIAVEL'].includes(input.tipo)) {
    throw new LancamentoValidationError('Tipo de lançamento inválido')
  }

  if (!IS_MOCK) {
    await apiFetch('/api/v1/cash-flow', {
      method: 'POST',
      body: JSON.stringify({
        tipo: input.tipo,
        descricao,
        valor: input.valor,
      }),
    })
    await refreshAfterMutation()
    const found = getDb().lancamentos.find(
      (l) => l.descricao === descricao && l.valor === input.valor,
    )
    if (found) return found
  }

  const db = getDb()
  const lanc: Lancamento = {
    id: uid(),
    tenant_id: input.tenant_id,
    tipo: input.tipo,
    valor: Math.round(input.valor * 100) / 100,
    descricao,
    data: input.data ?? today(),
    categoria: input.categoria,
    subtitulo: input.subtitulo?.trim() || undefined,
    status_transacao: input.status_transacao ?? 'PAGO',
    vencimento: input.vencimento,
    profissional_id: input.profissional_id,
    metodo_pagamento: input.metodo_pagamento,
    fornecedor: input.fornecedor?.trim() || undefined,
    natureza: input.natureza,
    recorrente: input.recorrente,
    frequencia_recorrencia: input.recorrente ? input.frequencia_recorrencia : undefined,
  }
  db.lancamentos.push(lanc)
  persistDb(db)
  return lanc
}

export function updateLancamento(
  id: string,
  patch: Partial<Omit<Lancamento, 'id' | 'tenant_id'>>,
): Lancamento {
  const db = getDb()
  const idx = db.lancamentos.findIndex((l) => l.id === id)
  if (idx < 0) throw new LancamentoValidationError('Lançamento não encontrado')
  const updated = { ...db.lancamentos[idx], ...patch, id }
  db.lancamentos[idx] = updated
  persistDb(db)
  return updated
}

export function deleteLancamento(id: string): void {
  const db = getDb()
  const idx = db.lancamentos.findIndex((l) => l.id === id)
  if (idx < 0) throw new LancamentoValidationError('Lançamento não encontrado')
  db.lancamentos.splice(idx, 1)
  persistDb(db)
}

export function lancamentoCategoria(l: Lancamento): import('../types').TransacaoCategoria {
  if (l.categoria) return l.categoria
  if (l.tipo === 'ENTRADA') return 'SERVICO'
  if (l.tipo === 'CUSTO_FIXO') return 'ALUGUEL'
  return 'OUTRO'
}

export function lancamentoStatus(l: Lancamento): import('../types').TransacaoStatus {
  if (l.status_transacao) return l.status_transacao
  if (l.status_pagamento === 'PENDENTE') return 'PENDENTE'
  return 'PAGO'
}

export function isReceita(l: Lancamento): boolean {
  return l.tipo === 'ENTRADA'
}

export interface FinanceiroStats {
  saldoMensal: number
  totalReceitas: number
  totalDespesas: number
  variacaoSaldoPct: number | null
  atendimentos: number
  faturasPendentes: number
  mesAnteriorLabel: string
}

export function getFinanceiroStats(tenantId: string, yearMonth: string): FinanceiroStats {
  const db = getDb()
  const inMonth = (iso: string) => iso.slice(0, 7) === yearMonth
  const lancs = db.lancamentos.filter((l) => l.tenant_id === tenantId && inMonth(l.data))

  const totalReceitas = lancs.filter(isReceita).reduce((s, l) => s + l.valor, 0)
  const totalDespesas = lancs.filter((l) => !isReceita(l)).reduce((s, l) => s + l.valor, 0)
  const saldoMensal = totalReceitas - totalDespesas

  const prevYm = prevYearMonth(yearMonth)
  const prevReceitas = db.lancamentos
    .filter((l) => l.tenant_id === tenantId && l.data.slice(0, 7) === prevYm && isReceita(l))
    .reduce((s, l) => s + l.valor, 0)
  const prevDespesas = db.lancamentos
    .filter((l) => l.tenant_id === tenantId && l.data.slice(0, 7) === prevYm && !isReceita(l))
    .reduce((s, l) => s + l.valor, 0)
  const prevSaldo = prevReceitas - prevDespesas
  const variacaoSaldoPct =
    prevSaldo !== 0 ? Math.round(((saldoMensal - prevSaldo) / Math.abs(prevSaldo)) * 1000) / 10 : null

  const atendimentos = db.agendamentos.filter(
    (a) =>
      a.tenant_id === tenantId &&
      a.status === 'CONCLUIDO' &&
      inMonth(a.data),
  ).length

  const faturasPendentes = lancs.filter((l) => lancamentoStatus(l) === 'PENDENTE').length

  return {
    saldoMensal,
    totalReceitas,
    totalDespesas,
    variacaoSaldoPct,
    atendimentos,
    faturasPendentes,
    mesAnteriorLabel: formatMonthYearBR(prevYm),
  }
}

export interface TransacaoFilters {
  yearMonth?: string
  categoria?: import('../types').TransacaoCategoria | 'all'
  tipo?: 'all' | 'receita' | 'despesa'
  search?: string
}

export function getTransacoesFiltradas(
  tenantId: string,
  filters: TransacaoFilters,
): Lancamento[] {
  const q = filters.search?.trim().toLowerCase() ?? ''
  let list = getDb().lancamentos.filter((l) => l.tenant_id === tenantId)

  if (filters.yearMonth) {
    list = list.filter((l) => l.data.slice(0, 7) === filters.yearMonth)
  }
  if (filters.categoria && filters.categoria !== 'all') {
    list = list.filter((l) => lancamentoCategoria(l) === filters.categoria)
  }
  if (filters.tipo === 'receita') list = list.filter(isReceita)
  if (filters.tipo === 'despesa') list = list.filter((l) => !isReceita(l))
  if (q) {
    list = list.filter(
      (l) =>
        l.descricao.toLowerCase().includes(q) ||
        l.subtitulo?.toLowerCase().includes(q) ||
        lancamentoCategoria(l).toLowerCase().includes(q),
    )
  }

  return list.sort((a, b) => b.data.localeCompare(a.data) || b.id.localeCompare(a.id))
}

export function calcLucroLiquido(tenantId: string): {
  entradas: number
  custosFixos: number
  comissoes: number
  lucro: number
  pendentes: number
} {
  const lanc = getDb().lancamentos.filter((l) => l.tenant_id === tenantId)
  const entradas = lanc.filter((l) => l.tipo === 'ENTRADA').reduce((s, l) => s + l.valor, 0)
  const custosFixos = lanc.filter((l) => l.tipo === 'CUSTO_FIXO').reduce((s, l) => s + l.valor, 0)
  const comissoes = lanc
    .filter((l) => l.tipo === 'CUSTO_VARIAVEL')
    .reduce((s, l) => s + l.valor, 0)
  const pendentes = lanc
    .filter((l) => l.tipo === 'CUSTO_VARIAVEL' && l.status_pagamento === 'PENDENTE')
    .reduce((s, l) => s + l.valor, 0)
  return { entradas, custosFixos, comissoes, lucro: entradas - custosFixos - comissoes, pendentes }
}

export function getMRR(): number {
  const db = getDb()
  return db.tenants
    .filter((t) => t.status === 'ATIVO')
    .reduce((sum, t) => {
      const plano = getPlano(t.plano_id)
      return sum + (plano?.preco_mensal ?? 0)
    }, 0)
}

export interface FaturaSaasView extends FaturaSaas {
  tenant_nome: string
  tenant_slug: string
}

export function getFaturasSaasRecentes(search = '', limit = 20): FaturaSaasView[] {
  const db = getDb()
  const q = search.trim().toLowerCase()
  return db.faturas_saas
    .map((f) => {
      const tenant = db.tenants.find((t) => t.id === f.tenant_id)
      return {
        ...f,
        tenant_nome: tenant?.nome ?? 'Salão removido',
        tenant_slug: tenant?.slug ?? '',
      }
    })
    .filter(
      (f) =>
        !q ||
        f.tenant_nome.toLowerCase().includes(q) ||
        f.id.toLowerCase().includes(q) ||
        f.tenant_slug.toLowerCase().includes(q),
    )
    .sort((a, b) => b.data.localeCompare(a.data))
    .slice(0, limit)
}

export function getDistribuicaoPlanos(): {
  id: string
  nome: string
  count: number
  pct: number
}[] {
  const db = getDb()
  const total = db.tenants.length
  if (total === 0) return []
  return db.planos
    .map((p) => {
      const count = db.tenants.filter((t) => t.plano_id === p.id).length
      return {
        id: p.id,
        nome: p.nome,
        count,
        pct: Math.round((count / total) * 100),
      }
    })
    .filter((d) => d.count > 0)
    .sort((a, b) => b.count - a.count)
}

export function getMRRHistorico6Meses(): { label: string; mrr: number }[] {
  const current = getMRR()
  const months = [...recentYearMonths(6)].reverse()
  const factors = [0.72, 0.78, 0.84, 0.9, 0.95, 1]
  return months.map((ym, i) => ({
    label: formatMonthYearBR(ym).split(' ')[0]?.slice(0, 3) ?? ym,
    mrr: Math.round(current * factors[i] * 100) / 100,
  }))
}

export function getMRRVariacao(): { pct: number; positive: boolean } {
  const hist = getMRRHistorico6Meses()
  if (hist.length < 2) return { pct: 0, positive: true }
  const prev = hist[hist.length - 2].mrr
  const curr = hist[hist.length - 1].mrr
  if (prev <= 0) return { pct: 0, positive: true }
  const pct = Math.round(((curr - prev) / prev) * 1000) / 10
  return { pct: Math.abs(pct), positive: pct >= 0 }
}

export function getChurnStats(): { rate: number; delta: number } {
  const db = getDb()
  const total = db.tenants.length
  if (total === 0) return { rate: 0, delta: 0 }
  const vencidos = db.tenants.filter((t) => t.status === 'VENCIDO').length
  const rate = Math.round((vencidos / total) * 1000) / 10
  const delta = vencidos > 0 ? -0.2 : 0.2
  return { rate, delta }
}

export function getFinanceiroInsights(): { type: 'success' | 'warning' | 'danger'; title: string; body: string }[] {
  const db = getDb()
  const dist = getDistribuicaoPlanos()
  const vencidos = db.tenants.filter((t) => t.status === 'VENCIDO')
  const insights: { type: 'success' | 'warning' | 'danger'; title: string; body: string }[] = []

  const top = dist[0]
  if (top && top.pct >= 40) {
    insights.push({
      type: 'success',
      title: 'Plano em destaque',
      body: `${top.pct}% dos salões usam o plano ${top.nome}. Considere campanhas de upgrade para os demais tiers.`,
    })
  }

  if (vencidos.length > 0) {
    insights.push({
      type: 'danger',
      title: 'Risco de churn',
      body: `${vencidos.length} salão${vencidos.length > 1 ? 'ões' : ''} com assinatura vencida. Renove ou entre em contato preventivo.`,
    })
  }

  const pendentes = db.faturas_saas.filter((f) => f.status === 'PENDENTE').length
  if (pendentes > 0) {
    insights.push({
      type: 'warning',
      title: 'Cobranças pendentes',
      body: `${pendentes} fatura${pendentes > 1 ? 's' : ''} aguardando confirmação de pagamento neste ciclo.`,
    })
  }

  return insights.slice(0, 3)
}

export function getSlotsLivres(
  profissionalId: string,
  data: string,
  count = 3,
): { hora: string; data: string }[] {
  const db = getDb()
  const prof = db.profissionais.find((p) => p.id === profissionalId)
  if (!prof) return []

  const day = new Date(data + 'T12:00:00').getDay()
  const exp = prof.expedientes.find((e) => e.dia_semana === day)
  if (!exp) return []

  const start = timeToMinutes(exp.horario_entrada)
  const end = timeToMinutes(exp.horario_saida)
  const lunchStart = exp.inicio_almoco ? timeToMinutes(exp.inicio_almoco) : null
  const lunchEnd = exp.fim_almoco ? timeToMinutes(exp.fim_almoco) : null

  const ocupados = db.agendamentos
    .filter(
      (a) =>
        a.profissional_id === profissionalId &&
        a.data === data &&
        a.status !== 'CANCELADO',
    )
    .map((a) => {
      const dur = getAgendamentoDuration(a, db)
      return { start: timeToMinutes(a.hora_inicio), end: timeToMinutes(a.hora_inicio) + dur }
    })

  const livres: { hora: string; data: string }[] = []
  for (let m = start; m < end - 30 && livres.length < count; m += 30) {
    if (lunchStart !== null && lunchEnd !== null && m >= lunchStart && m < lunchEnd) continue
    const slotEnd = m + 30
    const conflito = ocupados.some((o) => m < o.end && slotEnd > o.start)
    if (!conflito) {
      livres.push({ hora: minutesToTime(m), data })
    }
  }
  return livres
}

/** Horários em que cabe um novo agendamento com a duração informada (somente CONFIRMADO). */
export function getHorariosDisponiveis(
  profissionalId: string,
  data: string,
  duracaoMinutos: number,
): string[] {
  const db = getDb()
  const prof = db.profissionais.find((p) => p.id === profissionalId)
  if (!prof) return []

  const day = new Date(data + 'T12:00:00').getDay()
  const exp = prof.expedientes.find((e) => e.dia_semana === day)
  if (!exp) return []

  const start = timeToMinutes(exp.horario_entrada)
  const end = timeToMinutes(exp.horario_saida)
  const lunchStart = exp.inicio_almoco ? timeToMinutes(exp.inicio_almoco) : null
  const lunchEnd = exp.fim_almoco ? timeToMinutes(exp.fim_almoco) : null

  const horarios: string[] = []
  for (let m = start; m + duracaoMinutos <= end; m += 30) {
    if (lunchStart !== null && lunchEnd !== null) {
      const slotEnd = m + duracaoMinutos
      if (m < lunchEnd && slotEnd > lunchStart) continue
    }
    const hora = minutesToTime(m)
    try {
      const r = avaliarNovoAgendamento(profissionalId, data, hora, duracaoMinutos)
      if (r.status === 'CONFIRMADO') horarios.push(hora)
    } catch {
      /* ocupado */
    }
  }
  return horarios
}

export function normalizeTelefone(tel: string): string {
  return tel.replace(/\D/g, '')
}

export function findContaClienteByEmail(email: string): ContaCliente | undefined {
  const q = email.trim().toLowerCase()
  return getDb().contas_cliente.find((c) => c.email.toLowerCase() === q)
}

export function findContaClienteByTelefone(telefone: string): ContaCliente | undefined {
  const tel = normalizeTelefone(telefone)
  return getDb().contas_cliente.find((c) => c.telefone === tel)
}

export interface PerfilGlobalCliente {
  nome: string
  email?: string
  hasConta: boolean
  conta_id?: string
}

/** Busca dados em contas globais ou cadastros de salões (cross-tenant por telefone). */
export function lookupPerfilGlobal(telefone: string): PerfilGlobalCliente | null {
  const tel = normalizeTelefone(telefone)
  if (tel.length < 10) return null

  const db = getDb()
  const conta = db.contas_cliente.find((c) => c.telefone === tel)
  if (conta) {
    return { nome: conta.nome, email: conta.email, hasConta: true, conta_id: conta.id }
  }

  const cadastro = db.clientes.find((c) => c.telefone === tel && c.ativo)
  if (cadastro) {
    return { nome: cadastro.nome, email: cadastro.email, hasConta: false }
  }
  return null
}

export class ClienteAuthError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'ClienteAuthError'
  }
}

export function registerContaCliente(input: {
  nome: string
  telefone: string
  email: string
  password: string
}): ContaCliente {
  const db = getDb()
  const tel = normalizeTelefone(input.telefone)
  const email = input.email.trim().toLowerCase()
  const nome = input.nome.trim()

  if (!nome || tel.length < 10) throw new ClienteAuthError('Nome e telefone válidos são obrigatórios')
  if (!email.includes('@')) throw new ClienteAuthError('E-mail inválido')
  if (input.password.length < 6) throw new ClienteAuthError('Senha deve ter ao menos 6 caracteres')

  if (db.contas_cliente.some((c) => c.telefone === tel)) {
    throw new ClienteAuthError('Telefone já cadastrado. Faça login.')
  }
  if (db.contas_cliente.some((c) => c.email.toLowerCase() === email)) {
    throw new ClienteAuthError('E-mail já cadastrado. Faça login.')
  }

  const conta: ContaCliente = {
    id: uid(),
    telefone: tel,
    email,
    password: input.password,
    nome,
    criado_em: today(),
  }
  db.contas_cliente.push(conta)
  persistDb(db)
  return conta
}

/** Cadastro rápido do cliente (nome, telefone, e-mail) — cria conta ou reutiliza telefone existente. */
export function registerOrLoginCliente(input: {
  nome: string
  telefone: string
  email: string
}): ContaCliente {
  const db = getDb()
  const tel = normalizeTelefone(input.telefone)
  const email = input.email.trim().toLowerCase()
  const nome = input.nome.trim()

  if (!nome || tel.length < 10) throw new ClienteAuthError('Nome e telefone válidos são obrigatórios')
  if (!email.includes('@')) throw new ClienteAuthError('E-mail inválido')

  const byTel = db.contas_cliente.find((c) => c.telefone === tel)
  if (byTel) {
    const idx = db.contas_cliente.findIndex((c) => c.id === byTel.id)
    db.contas_cliente[idx] = { ...byTel, nome, email }
    persistDb(db)
    return db.contas_cliente[idx]
  }

  const emailTaken = db.contas_cliente.find(
    (c) => c.email.toLowerCase() === email && c.telefone !== tel,
  )
  if (emailTaken) {
    throw new ClienteAuthError('E-mail já cadastrado. Use o telefone vinculado à conta.')
  }

  const autoPassword = `Agenda@${tel.slice(-4)}`
  return registerContaCliente({ nome, telefone: tel, email, password: autoPassword })
}

export function loginContaCliente(email: string, password: string): ContaCliente {
  const conta = findContaClienteByEmail(email)
  if (!conta || conta.password !== password) {
    throw new ClienteAuthError('E-mail ou senha inválidos')
  }
  return conta
}

export function ensureClienteNoTenant(
  tenantId: string,
  data: { nome: string; telefone: string; email?: string },
): Cliente {
  const db = getDb()
  const tel = normalizeTelefone(data.telefone)
  const idx = db.clientes.findIndex((c) => c.tenant_id === tenantId && c.telefone === tel)
  if (idx >= 0) {
    if (data.email?.trim() && !db.clientes[idx].email) {
      db.clientes[idx].email = data.email.trim()
      persistDb(db)
    }
    return db.clientes[idx]
  }
  const cliente: Cliente = {
    id: uid(),
    tenant_id: tenantId,
    nome: data.nome.trim(),
    telefone: tel,
    email: data.email?.trim(),
    ativo: true,
    criado_em: today(),
  }
  db.clientes.push(cliente)
  persistDb(db)
  return cliente
}

export function getProfissionalFavorito(contaId: string, tenantId: string): string | undefined {
  return getDb().cliente_preferencias.find(
    (p) => p.conta_cliente_id === contaId && p.tenant_id === tenantId,
  )?.profissional_favorito_id
}

export function setProfissionalFavorito(
  contaId: string,
  tenantId: string,
  profissionalId: string,
): void {
  const db = getDb()
  const idx = db.cliente_preferencias.findIndex(
    (p) => p.conta_cliente_id === contaId && p.tenant_id === tenantId,
  )
  if (idx >= 0) {
    db.cliente_preferencias[idx].profissional_favorito_id = profissionalId
  } else {
    db.cliente_preferencias.push({
      conta_cliente_id: contaId,
      tenant_id: tenantId,
      profissional_favorito_id: profissionalId,
    })
  }
  persistDb(db)
}

export function getAgendamentosCliente(telefone: string, tenantId?: string) {
  const tel = normalizeTelefone(telefone)
  const db = getDb()
  return db.agendamentos
    .filter(
      (a) =>
        a.cliente_telefone === tel &&
        (!tenantId || a.tenant_id === tenantId) &&
        a.status !== 'CANCELADO',
    )
    .sort((a, b) => b.data.localeCompare(a.data) || b.hora_inicio.localeCompare(a.hora_inicio))
    .map((a) => {
      const tenant = db.tenants.find((t) => t.id === a.tenant_id)
      const prof = db.profissionais.find((p) => p.id === a.profissional_id)
      return {
        ...a,
        tenant_nome: tenant?.nome ?? '—',
        tenant_slug: tenant?.slug ?? '',
        servico_nome: getAgendamentoServicosNomes(a, db),
        profissional_nome: prof?.nome ?? '—',
      }
    })
}

export function remarcarAgendamento(
  agendamentoAntigoId: string,
  novaData: string,
  novaHora: string,
): Agendamento {
  const db = getDb()
  const old = db.agendamentos.find((a) => a.id === agendamentoAntigoId)
  if (!old) throw new Error('Agendamento não encontrado')
  old.status = 'CANCELADO'
  const novo: Agendamento = {
    ...old,
    id: uid(),
    data: novaData,
    hora_inicio: novaHora,
    status: 'CONFIRMADO',
  }
  db.agendamentos.push(novo)
  persistDb(db)
  return novo
}

export const SLUG_REGEX = /^[a-z0-9]+(-[a-z0-9]+)*$/

// ——— Super Admin ———

export interface PlatformClient {
  id: string
  cliente_id: string
  nome: string
  telefone: string
  email?: string
  tenant_id: string
  tenant_nome: string
  ultima_visita: string
  criado_em: string
  gasto_total: number
  visitas: number
  ativo: boolean
}

function getAgendamentoStatsForPhone(
  db: MockDatabase,
  tenantId: string,
  telefone: string,
): { visitas: number; gasto_total: number; ultima_visita: string } {
  let visitas = 0
  let gasto_total = 0
  let ultima_visita = ''
  for (const ag of db.agendamentos) {
    if (
      ag.tenant_id === tenantId &&
      ag.cliente_telefone === telefone &&
      ag.status !== 'CANCELADO'
    ) {
      visitas++
      const servico = db.servicos.find((s) => s.id === ag.servico_id)
      gasto_total += servico?.preco ?? 0
      if (ag.data > ultima_visita) ultima_visita = ag.data
    }
  }
  return { visitas, gasto_total, ultima_visita }
}

function enrichCliente(db: MockDatabase, c: Cliente): PlatformClient {
  const tenant = db.tenants.find((t) => t.id === c.tenant_id)
  const stats = getAgendamentoStatsForPhone(db, c.tenant_id, c.telefone)
  return {
    id: `${c.tenant_id}:${c.telefone}`,
    cliente_id: c.id,
    nome: c.nome,
    telefone: c.telefone,
    email: c.email,
    tenant_id: c.tenant_id,
    tenant_nome: tenant?.nome ?? '—',
    ultima_visita: stats.ultima_visita || c.criado_em,
    criado_em: c.criado_em,
    gasto_total: stats.gasto_total,
    visitas: stats.visitas,
    ativo: c.ativo,
  }
}

export function getPlatformClients(): PlatformClient[] {
  const db = getDb()
  return db.clientes
    .map((c) => enrichCliente(db, c))
    .sort((a, b) => b.gasto_total - a.gasto_total)
}

export function getSuperAdminStats() {
  const db = getDb()
  const clients = getPlatformClients()
  const mrr = getMRR()
  const ativos = db.tenants.filter((t) => t.status === 'ATIVO').length
  const vencidos = db.tenants.filter((t) => t.status === 'VENCIDO').length
  const novosMes = db.tenants.filter((t) => {
    const created = new Date(t.criado_em + 'T12:00:00')
    const now = new Date()
    return created.getMonth() === now.getMonth() && created.getFullYear() === now.getFullYear()
  }).length
  const ticketMedio =
    clients.length > 0
      ? clients.reduce((s, c) => s + c.gasto_total, 0) / clients.length
      : 0
  const taxaRetorno =
    clients.length > 0
      ? Math.round((clients.filter((c) => c.visitas > 1).length / clients.length) * 100)
      : 0

  return { mrr, ativos, vencidos, novosMes, ticketMedio, totalClientes: clients.length, taxaRetorno }
}

export async function assignPlanToTenant(tenantId: string, planoId: string): Promise<Tenant> {
  if (!IS_MOCK) {
    await apiFetch(`/api/v1/admin/establishments/${tenantId}/assign-plan`, {
      method: 'POST',
      body: JSON.stringify({ plano_id: planoId, meses: 12 }),
    })
    await syncAdminBootstrap()
    const t = getDb().tenants.find((x) => x.id === tenantId)
    if (t) return t
    throw new Error('Salão não encontrado')
  }

  const db = getDb()
  const idx = db.tenants.findIndex((t) => t.id === tenantId)
  if (idx < 0) throw new Error('Salão não encontrado')
  db.tenants[idx] = {
    ...db.tenants[idx],
    plano_id: planoId,
    status: 'ATIVO',
    data_vencimento: addMonths(today(), 12),
  }
  persistDb(db)
  return db.tenants[idx]
}

export async function renewTenant(tenantId: string, months = 12): Promise<Tenant> {
  if (!IS_MOCK) {
    const tenant = getDb().tenants.find((t) => t.id === tenantId)
    if (!tenant) throw new Error('Salão não encontrado')
    await apiFetch(`/api/v1/admin/establishments/${tenantId}/assign-plan`, {
      method: 'POST',
      body: JSON.stringify({ plano_id: tenant.plano_id, meses: months }),
    })
    await syncAdminBootstrap()
    return getDb().tenants.find((t) => t.id === tenantId) ?? tenant
  }

  const db = getDb()
  const idx = db.tenants.findIndex((t) => t.id === tenantId)
  if (idx < 0) throw new Error('Salão não encontrado')
  const base = db.tenants[idx].data_vencimento > today()
    ? db.tenants[idx].data_vencimento
    : today()
  db.tenants[idx] = {
    ...db.tenants[idx],
    status: 'ATIVO',
    data_vencimento: addMonths(base, months),
  }
  persistDb(db)
  return db.tenants[idx]
}

export async function suspendTenant(tenantId: string): Promise<Tenant> {
  if (!IS_MOCK) {
    await apiFetch(`/api/v1/admin/establishments/${tenantId}/status`, {
      method: 'PUT',
      body: JSON.stringify({ ativo: false }),
    })
    await syncAdminBootstrap()
    const t = getDb().tenants.find((x) => x.id === tenantId)
    if (t) return t
    throw new Error('Salão não encontrado')
  }

  const db = getDb()
  const idx = db.tenants.findIndex((t) => t.id === tenantId)
  if (idx < 0) throw new Error('Salão não encontrado')
  db.tenants[idx] = { ...db.tenants[idx], status: 'SUSPENSO' }
  persistDb(db)
  return db.tenants[idx]
}

export async function activateTenant(tenantId: string): Promise<Tenant> {
  if (!IS_MOCK) {
    await apiFetch(`/api/v1/admin/establishments/${tenantId}/status`, {
      method: 'PUT',
      body: JSON.stringify({ ativo: true }),
    })
    await syncAdminBootstrap()
    const t = getDb().tenants.find((x) => x.id === tenantId)
    if (t) return t
    throw new Error('Salão não encontrado')
  }

  const db = getDb()
  const idx = db.tenants.findIndex((t) => t.id === tenantId)
  if (idx < 0) throw new Error('Salão não encontrado')
  db.tenants[idx] = { ...db.tenants[idx], status: 'ATIVO' }
  persistDb(db)
  return db.tenants[idx]
}

export function createDonaForTenant(
  tenantId: string,
  nome: string,
  email: string,
): void {
  const db = getDb()
  const idx = db.tenants.findIndex((t) => t.id === tenantId)
  if (idx < 0) throw new Error('Salão não encontrado')
  if (db.users.some((u) => u.email.toLowerCase() === email.toLowerCase())) {
    throw new Error('E-mail já cadastrado')
  }
  db.tenants[idx].dona_nome = nome
  db.tenants[idx].dona_email = email
  db.users.push({
    id: uid(),
    email,
    password: 'AgendaGlow@2026',
    role: 'DONA',
    tenant_id: tenantId,
    nome,
  })
  persistDb(db)
}

export async function createTenant(data: Omit<Tenant, 'id'>): Promise<Tenant> {
  if (!IS_MOCK) {
    const res = await apiFetch<{ id: string; slug: string }>('/api/v1/admin/establishments', {
      method: 'POST',
      body: JSON.stringify({ nome_comercial: data.nome, slug: data.slug }),
    })
    if (data.plano_id) {
      await apiFetch(`/api/v1/admin/establishments/${res.id}/assign-plan`, {
        method: 'POST',
        body: JSON.stringify({ plano_id: data.plano_id, meses: 12 }),
      })
    }
    await syncAdminBootstrap()
    const tenant = getDb().tenants.find((t) => t.id === res.id || t.slug === res.slug)
    if (tenant) return tenant
    return { ...data, id: res.id, slug: res.slug }
  }

  const tenant: Tenant = { ...data, id: uid() }
  const db = getDb()
  db.tenants.push(tenant)
  persistDb(db)
  return tenant
}

export function renewAllExpired(): number {
  const db = getDb()
  let count = 0
  for (const t of db.tenants) {
    if (t.status === 'VENCIDO') {
      const idx = db.tenants.findIndex((x) => x.id === t.id)
      db.tenants[idx] = {
        ...t,
        status: 'ATIVO',
        data_vencimento: addMonths(today(), 12),
      }
      count++
    }
  }
  if (count > 0) persistDb(db)
  return count
}

// ——— Dona / Tenant ———

export function getTenantClients(tenantId: string): PlatformClient[] {
  const db = getDb()
  return db.clientes
    .filter((c) => c.tenant_id === tenantId)
    .map((c) => enrichCliente(db, c))
    .sort((a, b) => b.gasto_total - a.gasto_total)
}

export async function createCliente(
  tenantId: string,
  data: { nome: string; telefone: string; email?: string },
): Promise<Cliente> {
  if (!IS_MOCK) {
    const { id } = await apiFetch<{ id: string }>('/api/v1/clients', {
      method: 'POST',
      body: JSON.stringify(data),
    })
    await refreshAfterMutation()
    const c = getDb().clientes.find((x) => x.id === id)
    if (c) return c
    return {
      id,
      tenant_id: tenantId,
      nome: data.nome.trim(),
      telefone: data.telefone.replace(/\D/g, ''),
      email: data.email?.trim(),
      ativo: true,
      criado_em: today(),
    }
  }

  const db = getDb()
  const tel = data.telefone.replace(/\D/g, '')
  if (db.clientes.some((c) => c.tenant_id === tenantId && c.telefone === tel)) {
    throw new Error('Já existe um cliente com este telefone')
  }
  const cliente: Cliente = {
    id: uid(),
    tenant_id: tenantId,
    nome: data.nome.trim(),
    telefone: tel,
    email: data.email?.trim(),
    ativo: true,
    criado_em: today(),
  }
  db.clientes.push(cliente)
  persistDb(db)
  return cliente
}

export function updateCliente(
  id: string,
  data: { nome?: string; telefone?: string; email?: string },
): Cliente {
  const db = getDb()
  const idx = db.clientes.findIndex((c) => c.id === id)
  if (idx < 0) throw new Error('Cliente não encontrado')
  const current = db.clientes[idx]
  const tel = data.telefone ? data.telefone.replace(/\D/g, '') : current.telefone
  if (
    db.clientes.some(
      (c) => c.tenant_id === current.tenant_id && c.telefone === tel && c.id !== id,
    )
  ) {
    throw new Error('Já existe um cliente com este telefone')
  }
  db.clientes[idx] = {
    ...current,
    nome: data.nome?.trim() ?? current.nome,
    telefone: tel,
    email: data.email !== undefined ? data.email.trim() : current.email,
  }
  persistDb(db)
  return db.clientes[idx]
}

export function setClienteAtivo(id: string, ativo: boolean): void {
  const db = getDb()
  const idx = db.clientes.findIndex((c) => c.id === id)
  if (idx < 0) throw new Error('Cliente não encontrado')
  db.clientes[idx].ativo = ativo
  persistDb(db)
}

export function getClienteById(tenantId: string, clienteId: string): Cliente | undefined {
  return getDb().clientes.find((c) => c.id === clienteId && c.tenant_id === tenantId)
}

export function getClienteNotas(clienteId: string): ClienteNotaInterna[] {
  return getDb()
    .cliente_notas.filter((n) => n.cliente_id === clienteId)
    .sort((a, b) => b.data.localeCompare(a.data))
}

export function getClienteGaleria(clienteId: string): ClienteGaleriaItem[] {
  return getDb()
    .cliente_galeria.filter((g) => g.cliente_id === clienteId)
    .sort((a, b) => b.data.localeCompare(a.data))
}

export function addClienteNota(clienteId: string, autor: string, texto: string): ClienteNotaInterna {
  const db = getDb()
  const nota: ClienteNotaInterna = {
    id: uid(),
    cliente_id: clienteId,
    autor,
    data: today(),
    texto: texto.trim(),
  }
  if (!nota.texto) throw new Error('Texto da nota é obrigatório')
  db.cliente_notas.push(nota)
  persistDb(db)
  return nota
}

export interface ClienteProfileMetrics {
  ltv: number
  ticketMedio: number
  frequenciaMensal: number
  visitas: number
  proximoAgendamento: { data: string; hora: string } | null
}

export function getClienteProfileMetrics(
  tenantId: string,
  telefone: string,
): ClienteProfileMetrics {
  const db = getDb()
  const ags = db.agendamentos.filter(
    (a) =>
      a.tenant_id === tenantId &&
      a.cliente_telefone === telefone &&
      a.status !== 'CANCELADO',
  )
  const ltv = ags.reduce((s, a) => s + getAgendamentoValor(a, db), 0)
  const visitas = ags.length
  const ticketMedio = visitas > 0 ? ltv / visitas : 0

  const dates = ags.map((a) => a.data).sort()
  let frequenciaMensal = 0
  if (dates.length >= 2) {
    const first = new Date(`${dates[0]}T12:00:00`)
    const last = new Date(`${dates[dates.length - 1]}T12:00:00`)
    const months = Math.max(
      1,
      (last.getFullYear() - first.getFullYear()) * 12 + (last.getMonth() - first.getMonth()) + 1,
    )
    frequenciaMensal = Math.round((visitas / months) * 10) / 10
  } else if (visitas === 1) {
    frequenciaMensal = 1
  }

  const hoje = today()
  const futuro = db.agendamentos
    .filter(
      (a) =>
        a.tenant_id === tenantId &&
        a.cliente_telefone === telefone &&
        a.data >= hoje &&
        a.status !== 'CANCELADO' &&
        a.status !== 'CONCLUIDO',
    )
    .sort((a, b) => a.data.localeCompare(b.data) || a.hora_inicio.localeCompare(b.hora_inicio))

  return {
    ltv,
    ticketMedio,
    frequenciaMensal,
    visitas,
    proximoAgendamento: futuro[0]
      ? { data: futuro[0].data, hora: futuro[0].hora_inicio }
      : null,
  }
}

export function getClienteHistorico(tenantId: string, telefone: string) {
  const db = getDb()
  return db.agendamentos
    .filter((a) => a.tenant_id === tenantId && a.cliente_telefone === telefone)
    .sort((a, b) => b.data.localeCompare(a.data) || b.hora_inicio.localeCompare(a.hora_inicio))
    .map((a) => {
      const prof = db.profissionais.find((p) => p.id === a.profissional_id)
      return {
        ...a,
        servico_nome: getAgendamentoServicosNomes(a, db),
        profissional_nome: prof?.nome ?? '—',
        valor: getAgendamentoValor(a, db),
      }
    })
}

export function getDonaStats(tenantId: string) {
  const all = getTenantClients(tenantId)
  const clients = all.filter((c) => c.ativo)
  const db = getDb()
  const ags = db.agendamentos.filter((a) => a.tenant_id === tenantId && a.status !== 'CANCELADO')
  const receita = ags.reduce((s, a) => {
    return (
      s +
      getAgendamentoServicoIds(a).reduce((sum, sid) => {
        const srv = db.servicos.find((x) => x.id === sid)
        return sum + (srv?.preco ?? 0)
      }, 0)
    )
  }, 0)
  const novosMes = all.filter((c) => {
    const d = new Date(c.criado_em + 'T12:00:00')
    const now = new Date()
    return d.getMonth() === now.getMonth() && d.getFullYear() === now.getFullYear()
  }).length
  const taxaRetorno =
    clients.length > 0
      ? Math.round((clients.filter((c) => c.visitas > 1).length / clients.length) * 100)
      : 0
  const ticketMedio =
    clients.length > 0
      ? clients.reduce((s, c) => s + c.gasto_total, 0) / clients.length
      : 0
  const pctAtivos = all.length > 0 ? Math.round((clients.length / all.length) * 100) : 0
  return {
    totalClientes: all.length,
    clientesAtivos: clients.length,
    pctAtivos,
    novosMes,
    taxaRetorno,
    receita,
    ticketMedio,
    agendamentosHoje: ags.filter((a) => a.data === today()).length,
  }
}

export interface ClienteReativacaoRow {
  id: string
  nome: string
  telefone: string
  ultimoServico: string
  ultimaVisita: string
  diasAusente: number
}

export interface DonaFechamentoDia {
  data: string
  receitaHoje: number
  receitaVariacaoPct: number | null
  concluidos: number
  agendadosTotal: number
  metaPct: number
  faltamAtendimentos: number
  noShows: number
  noShowsValorProjetado: number
  cancelamentos: number
  ticketMedioHoje: number
  ticketMedioSemanal: number
  ticketMedioVariacaoPct: number | null
  ocupacaoAmanhaPct: number
  avaliacoesHoje: number
  clientesReativacao: ClienteReativacaoRow[]
  totalInativos: number
}

function agendamentoPassou(ag: Agendamento, db: MockDatabase, agoraMin: number): boolean {
  const start = timeToMinutes(ag.hora_inicio)
  const end = start + getAgendamentoDuration(ag, db)
  return agoraMin >= end
}

function calcOcupacaoDia(
  data: string,
  tenantId: string,
  db: MockDatabase,
): number {
  const day = new Date(`${data}T12:00:00`).getDay()
  const profissionais = db.profissionais.filter((p) => p.tenant_id === tenantId && p.ativo)
  const agendamentos = db.agendamentos.filter(
    (a) => a.tenant_id === tenantId && a.data === data && a.status !== 'CANCELADO',
  )
  let totalAvailable = 0
  let totalBooked = 0
  for (const prof of profissionais) {
    const exp = prof.expedientes.find((e) => e.dia_semana === day)
    if (!exp) continue
    const work = timeToMinutes(exp.horario_saida) - timeToMinutes(exp.horario_entrada)
    const lunch =
      exp.inicio_almoco && exp.fim_almoco
        ? timeToMinutes(exp.fim_almoco) - timeToMinutes(exp.inicio_almoco)
        : 0
    totalAvailable += Math.max(0, work - lunch)
    const ags = agendamentos.filter((a) => a.profissional_id === prof.id)
    totalBooked += ags.reduce((s, a) => s + getAgendamentoDuration(a, db), 0)
  }
  return totalAvailable > 0 ? Math.round((totalBooked / totalAvailable) * 100) : 0
}

function receitaConcluidosDia(tenantId: string, data: string, db: MockDatabase): number {
  return db.agendamentos
    .filter((a) => a.tenant_id === tenantId && a.data === data && a.status === 'CONCLUIDO')
    .reduce((s, a) => s + getAgendamentoValor(a, db), 0)
}

export function getClientesReativacao(tenantId: string, minDias = 30): ClienteReativacaoRow[] {
  const db = getDb()
  const byPhone = new Map<
    string,
    { nome: string; telefone: string; ultimaVisita: string; ultimoAg: Agendamento }
  >()

  for (const ag of db.agendamentos) {
    if (ag.tenant_id !== tenantId || ag.status === 'CANCELADO') continue
    const existing = byPhone.get(ag.cliente_telefone)
    if (
      !existing ||
      ag.data > existing.ultimaVisita ||
      (ag.data === existing.ultimaVisita && ag.hora_inicio > existing.ultimoAg.hora_inicio)
    ) {
      byPhone.set(ag.cliente_telefone, {
        nome: ag.cliente_nome,
        telefone: ag.cliente_telefone,
        ultimaVisita: ag.data,
        ultimoAg: ag,
      })
    }
  }

  const rows: ClienteReativacaoRow[] = []
  for (const v of byPhone.values()) {
    const dias = Math.floor(
      (Date.now() - new Date(`${v.ultimaVisita}T12:00:00`).getTime()) / 86400000,
    )
    if (dias < minDias) continue
    const cliente = db.clientes.find((c) => c.telefone === v.telefone && c.tenant_id === tenantId)
    rows.push({
      id: cliente?.id ?? `tel:${v.telefone}`,
      nome: v.nome,
      telefone: v.telefone,
      ultimoServico: getAgendamentoServicosNomes(v.ultimoAg, db),
      ultimaVisita: v.ultimaVisita,
      diasAusente: dias,
    })
  }

  return rows.sort((a, b) => b.diasAusente - a.diasAusente)
}

export function getDonaFechamentoDia(tenantId: string, data = today()): DonaFechamentoDia {
  const db = getDb()
  const isHoje = data === today()
  const agoraMin = isHoje ? new Date().getHours() * 60 + new Date().getMinutes() : 24 * 60

  const doDia = db.agendamentos.filter((a) => a.tenant_id === tenantId && a.data === data)
  const ativos = doDia.filter((a) => a.status !== 'CANCELADO')
  const concluidos = ativos.filter((a) => a.status === 'CONCLUIDO')
  const cancelamentos = doDia.filter((a) => a.status === 'CANCELADO')

  const noShows = ativos.filter(
    (a) =>
      a.status !== 'CONCLUIDO' &&
      isHoje &&
      agendamentoPassou(a, db, agoraMin),
  )
  const pendentesFuturos = ativos.filter(
    (a) => a.status !== 'CONCLUIDO' && (!isHoje || !agendamentoPassou(a, db, agoraMin)),
  )

  const receitaHoje = receitaConcluidosDia(tenantId, data, db)
  const receitaOntem = receitaConcluidosDia(tenantId, addDaysISO(data, -1), db)
  const receitaVariacaoPct =
    receitaOntem > 0 ? Math.round(((receitaHoje - receitaOntem) / receitaOntem) * 100) : null

  const agendadosTotal = ativos.length
  const metaPct = agendadosTotal > 0 ? Math.round((concluidos.length / agendadosTotal) * 100) : 0

  const ticketMedioHoje =
    concluidos.length > 0 ? receitaHoje / concluidos.length : 0

  const semanaInicio = addDaysISO(data, -6)
  const conclSemana = db.agendamentos.filter(
    (a) =>
      a.tenant_id === tenantId &&
      a.status === 'CONCLUIDO' &&
      a.data >= semanaInicio &&
      a.data <= data,
  )
  const receitaSemana = conclSemana.reduce((s, a) => s + getAgendamentoValor(a, db), 0)
  const ticketMedioSemanal =
    conclSemana.length > 0 ? receitaSemana / conclSemana.length : ticketMedioHoje
  const ticketMedioVariacaoPct =
    ticketMedioSemanal > 0
      ? Math.round(((ticketMedioHoje - ticketMedioSemanal) / ticketMedioSemanal) * 100)
      : null

  const reativacao = getClientesReativacao(tenantId, 30)
  const amanha = addDaysISO(data, 1)

  return {
    data,
    receitaHoje,
    receitaVariacaoPct,
    concluidos: concluidos.length,
    agendadosTotal,
    metaPct,
    faltamAtendimentos: pendentesFuturos.length,
    noShows: noShows.length,
    noShowsValorProjetado: noShows.reduce((s, a) => s + getAgendamentoValor(a, db), 0),
    cancelamentos: cancelamentos.length,
    ticketMedioHoje,
    ticketMedioSemanal,
    ticketMedioVariacaoPct,
    ocupacaoAmanhaPct: calcOcupacaoDia(amanha, tenantId, db),
    avaliacoesHoje: concluidos.length >= 5 ? 5 : concluidos.length,
    clientesReativacao: reativacao,
    totalInativos: reativacao.length,
  }
}

export function getFechamentoNota(tenantId: string, data: string): string {
  try {
    const all = JSON.parse(localStorage.getItem(FECHAMENTO_NOTA_KEY) || '{}') as Record<string, string>
    return all[`${tenantId}:${data}`] ?? ''
  } catch {
    return ''
  }
}

export function saveFechamentoNota(tenantId: string, data: string, texto: string) {
  const all = JSON.parse(localStorage.getItem(FECHAMENTO_NOTA_KEY) || '{}') as Record<string, string>
  all[`${tenantId}:${data}`] = texto
  localStorage.setItem(FECHAMENTO_NOTA_KEY, JSON.stringify(all))
}

export interface RetornoProfissionalRow {
  profissional_id: string
  profissional_nome: string
  /** Clientes com 1ª visita com este profissional no mês anterior. */
  novos_mes_anterior: number
  novos_retornaram: number
  taxa_retorno_novos: number | null
  /** VIPs (3+ visitas no salão) atendidos por este profissional no mês anterior. */
  fidelizados_mes_anterior: number
  fidelizados_retornaram: number
  taxa_retorno_fidelizados: number | null
}

export interface RetornoProfissionalResumo {
  mes_referencia: string
  mes_anterior: string
  mes_referencia_label: string
  mes_anterior_label: string
  por_profissional: RetornoProfissionalRow[]
  media_novos: number | null
  media_fidelizados: number | null
}

function yearMonthFromISO(iso: string): string {
  return iso.slice(0, 7)
}

function prevYearMonth(ym: string): string {
  const [y, m] = ym.split('-').map(Number)
  const d = new Date(y, m - 2, 1)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
}

function agendamentoAtivo(ag: Agendamento): boolean {
  return ag.status !== 'CANCELADO'
}

function visitasAteMes(db: MockDatabase, tenantId: string, telefone: string, ateYm: string): number {
  return db.agendamentos.filter(
    (a) =>
      a.tenant_id === tenantId &&
      a.cliente_telefone === telefone &&
      agendamentoAtivo(a) &&
      yearMonthFromISO(a.data) <= ateYm,
  ).length
}

function taxaPct(retornaram: number, base: number): number | null {
  if (base <= 0) return null
  return Math.round((retornaram / base) * 100)
}

function mediaTaxas(rows: RetornoProfissionalRow[], field: 'taxa_retorno_novos' | 'taxa_retorno_fidelizados'): number | null {
  const vals = rows.map((r) => r[field]).filter((v): v is number => v !== null)
  if (vals.length === 0) return null
  return Math.round(vals.reduce((s, v) => s + v, 0) / vals.length)
}

/** Retorno mensal de novos clientes e fidelizados (VIP) por profissional. */
export function getRetornoProfissionalMensal(
  tenantId: string,
  mesReferencia?: string,
): RetornoProfissionalResumo {
  const db = getDb()
  const mesRef = mesReferencia ?? yearMonthFromISO(today())
  const mesAnt = prevYearMonth(mesRef)
  const profs = db.profissionais.filter((p) => p.tenant_id === tenantId && p.ativo)

  const por_profissional: RetornoProfissionalRow[] = profs.map((prof) => {
    const agsProf = db.agendamentos.filter(
      (a) => a.tenant_id === tenantId && a.profissional_id === prof.id && agendamentoAtivo(a),
    )

    const porTelefone = new Map<string, Agendamento[]>()
    for (const ag of agsProf) {
      const list = porTelefone.get(ag.cliente_telefone) ?? []
      list.push(ag)
      porTelefone.set(ag.cliente_telefone, list)
    }

    const novosMesAnterior: string[] = []
    for (const [tel, list] of porTelefone) {
      const sorted = [...list].sort(
        (a, b) => a.data.localeCompare(b.data) || a.hora_inicio.localeCompare(b.hora_inicio),
      )
      const primeira = sorted[0]
      if (yearMonthFromISO(primeira.data) === mesAnt) {
        novosMesAnterior.push(tel)
      }
    }

    const phonesMesRef = new Set(
      agsProf.filter((a) => yearMonthFromISO(a.data) === mesRef).map((a) => a.cliente_telefone),
    )
    const novos_retornaram = novosMesAnterior.filter((tel) => phonesMesRef.has(tel)).length

    const fidelizadosMesAnterior = new Set<string>()
    for (const ag of agsProf) {
      if (yearMonthFromISO(ag.data) !== mesAnt) continue
      if (visitasAteMes(db, tenantId, ag.cliente_telefone, mesAnt) > 2) {
        fidelizadosMesAnterior.add(ag.cliente_telefone)
      }
    }
    const fidelizados_mes_anterior = fidelizadosMesAnterior.size
    const fidelizados_retornaram = [...fidelizadosMesAnterior].filter((tel) =>
      phonesMesRef.has(tel),
    ).length

    return {
      profissional_id: prof.id,
      profissional_nome: prof.nome,
      novos_mes_anterior: novosMesAnterior.length,
      novos_retornaram,
      taxa_retorno_novos: taxaPct(novos_retornaram, novosMesAnterior.length),
      fidelizados_mes_anterior,
      fidelizados_retornaram,
      taxa_retorno_fidelizados: taxaPct(fidelizados_retornaram, fidelizados_mes_anterior),
    }
  })

  return {
    mes_referencia: mesRef,
    mes_anterior: mesAnt,
    mes_referencia_label: formatMonthYearBR(mesRef),
    mes_anterior_label: formatMonthYearBR(mesAnt),
    por_profissional,
    media_novos: mediaTaxas(por_profissional, 'taxa_retorno_novos'),
    media_fidelizados: mediaTaxas(por_profissional, 'taxa_retorno_fidelizados'),
  }
}

export function updateTenant(id: string, patch: Partial<Tenant>): Tenant {
  const db = getDb()
  const idx = db.tenants.findIndex((t) => t.id === id)
  if (idx < 0) throw new Error('Salão não encontrado')
  db.tenants[idx] = { ...db.tenants[idx], ...patch }
  persistDb(db)
  return db.tenants[idx]
}

export async function createEspecialidade(tenantId: string, nome: string) {
  if (!IS_MOCK) {
    const { id } = await apiFetch<{ id: string }>('/api/v1/specialties', {
      method: 'POST',
      body: JSON.stringify({ nome }),
    })
    await refreshAfterMutation()
    return getDb().especialidades.find((e) => e.id === id) ?? { id, tenant_id: tenantId, nome }
  }
  const db = getDb()
  const esp = { id: uid(), tenant_id: tenantId, nome }
  db.especialidades.push(esp)
  persistDb(db)
  return esp
}

export async function updateEspecialidade(id: string, nome: string) {
  if (!IS_MOCK) {
    await apiFetch(`/api/v1/specialties/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ nome, ativo: true }),
    })
    await refreshAfterMutation()
    const esp = getDb().especialidades.find((e) => e.id === id)
    if (esp) return esp
    return { id, tenant_id: '', nome }
  }
  const db = getDb()
  const idx = db.especialidades.findIndex((e) => e.id === id)
  if (idx < 0) throw new Error('Especialidade não encontrada')
  db.especialidades[idx].nome = nome
  persistDb(db)
  return db.especialidades[idx]
}

export async function createServico(data: Omit<import('../types').Servico, 'id'>) {
  if (!IS_MOCK) {
    const { id } = await apiFetch<{ id: string }>('/api/v1/services', {
      method: 'POST',
      body: JSON.stringify({
        nome: data.nome,
        preco_base: data.preco,
        duracao_base_minutos: data.duracao_minutos,
      }),
    })
    await refreshAfterMutation()
    return getDb().servicos.find((s) => s.id === id) ?? { ...data, id }
  }
  const db = getDb()
  const srv = { ...data, id: uid() }
  db.servicos.push(srv)
  persistDb(db)
  return srv
}

export function getServicoById(tenantId: string, id: string) {
  return getDb().servicos.find((s) => s.tenant_id === tenantId && s.id === id)
}

export function updateServico(id: string, patch: Partial<Omit<import('../types').Servico, 'id'>>) {
  const db = getDb()
  const idx = db.servicos.findIndex((s) => s.id === id)
  if (idx < 0) throw new Error('Serviço não encontrado')
  db.servicos[idx] = { ...db.servicos[idx], ...patch }
  persistDb(db)
  return db.servicos[idx]
}

export function deleteServico(id: string) {
  const db = getDb()
  db.servicos = db.servicos.filter((s) => s.id !== id)
  persistDb(db)
}

export type InsumoNivelEstoque = 'ok' | 'atencao' | 'critico'

export function getInsumoNivelEstoque(i: import('../types').InsumoEstoque): InsumoNivelEstoque {
  if (i.quantidade <= Math.max(1, Math.floor(i.estoque_minimo * 0.5))) return 'critico'
  if (i.quantidade <= i.estoque_minimo) return 'atencao'
  return 'ok'
}

export function getInsumosStats(tenantId: string) {
  const items = getDb().insumos.filter((i) => i.tenant_id === tenantId && i.ativo)
  const totalSkus = items.length
  const totalUnidades = items.reduce((s, i) => s + i.quantidade, 0)
  const estoqueBaixo = items.filter((i) => getInsumoNivelEstoque(i) !== 'ok').length
  const valorEstoque = items.reduce((s, i) => s + i.quantidade * i.valor_unitario, 0)
  const investimentoMensal = Math.round(valorEstoque * 0.12)
  const investimentoMeta = Math.round(investimentoMensal / 0.72)
  const investimentoPct = investimentoMeta > 0
    ? Math.min(100, Math.round((investimentoMensal / investimentoMeta) * 100))
    : 0
  return {
    totalSkus,
    totalUnidades,
    estoqueBaixo,
    investimentoMensal,
    investimentoPct,
    variacaoMes: 12,
    valorEstoque,
  }
}

export function createInsumo(data: Omit<import('../types').InsumoEstoque, 'id'>) {
  const db = getDb()
  const item = { ...data, id: uid() }
  db.insumos.push(item)
  persistDb(db)
  return item
}

export function getInsumoById(tenantId: string, id: string) {
  return getDb().insumos.find((i) => i.tenant_id === tenantId && i.id === id)
}

export function updateInsumo(id: string, patch: Partial<Omit<import('../types').InsumoEstoque, 'id'>>) {
  const db = getDb()
  const idx = db.insumos.findIndex((i) => i.id === id)
  if (idx < 0) throw new Error('Insumo não encontrado')
  db.insumos[idx] = { ...db.insumos[idx], ...patch }
  persistDb(db)
  return db.insumos[idx]
}

export function deleteInsumo(id: string) {
  const db = getDb()
  db.insumos = db.insumos.filter((i) => i.id !== id)
  persistDb(db)
}

export function ajustarEstoqueInsumo(id: string, delta: number) {
  const db = getDb()
  const idx = db.insumos.findIndex((i) => i.id === id)
  if (idx < 0) throw new Error('Insumo não encontrado')
  db.insumos[idx].quantidade = Math.max(0, db.insumos[idx].quantidade + delta)
  persistDb(db)
  return db.insumos[idx]
}

export function countByEspecialidade(tenantId: string, espId: string) {
  const db = getDb()
  const profs = db.profissionais.filter((p) => p.tenant_id === tenantId && p.especialidade_id === espId).length
  const servicos = db.servicos.filter((s) => s.tenant_id === tenantId && s.ativo).length
  return { profs, servicos }
}
