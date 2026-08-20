import type { EarlySlotOfferResumo } from './earlySlot'

export type TenantStatus = 'ATIVO' | 'VENCIDO' | 'SUSPENSO'
export type WhatsAppIntegrationStatus = 'DESCONECTADO' | 'PENDENTE' | 'CONECTADO'
export type AgendamentoStatus =
  | 'AGENDADO'
  | 'CONFIRMADO'
  | 'EM_APROVACAO'
  | 'CANCELADO'
  | 'CONCLUIDO'
export type ConfirmacaoCliente =
  | 'PENDENTE'
  | 'CONFIRMADO_CLIENTE'
  | 'CANCELADO_CLIENTE'
export type FilaStatus = 'AGUARDANDO' | 'NOTIFICADO'
export type UserRole = 'SUPER_ADMIN' | 'DONA' | 'PROFISSIONAL' | 'SECRETARIA' | 'CLIENTE'

export interface PlanoSaas {
  id: string
  nome: string
  preco_mensal: number
  limite_profissionais: number
  ativo: boolean
}

export interface Tenant {
  id: string
  nome: string
  slug: string
  status: TenantStatus
  logo_url?: string
  plano_id: string
  bio?: string
  data_vencimento: string
  dona_nome?: string
  dona_email?: string
  criado_em: string
  email_contato?: string
  cep?: string
  logradouro?: string
  cidade?: string
  uf?: string
  dona_atua_como_profissional?: boolean
  whatsapp_status?: WhatsAppIntegrationStatus
  whatsapp_waba_id?: string
  whatsapp_phone_number_id?: string
  whatsapp_connected_at?: string
  /** Canonical staff signal — true only when the queue can fire offers. */
  early_slot_queue_active?: boolean
  early_slot_queue_inactive_reason?: 'whatsapp_indisponivel' | null
  /** Canonical public/booking signal for opt-in helper copy. */
  early_slot_notifications_available?: boolean
}

export interface CategoriaServico {
  id: string
  tenant_id: string
  nome: string
  icone?: string
}

export interface Especialidade {
  id: string
  tenant_id: string
  nome: string
}

export interface Expediente {
  dia_semana: number
  horario_entrada: string
  horario_saida: string
  inicio_almoco?: string
  fim_almoco?: string
}

export interface Profissional {
  id: string
  tenant_id: string
  nome: string
  especialidade_id: string
  comissao_percent: number
  ativo: boolean
  expedientes: Expediente[]
  user_id?: string
  eh_dona?: boolean
  email?: string
  foto_url?: string
  /** Convite enviado, aguardando aceite / cadastro pelo profissional */
  pendente_aprovacao?: boolean
  telefone?: string
  data_nascimento?: string
  biografia?: string
  portfolio_url?: string
  data_contratacao?: string
  especialidade_ids?: string[]
  modelo_pagamento?: 'PERCENTUAL' | 'FIXO'
  valor_fixo_atendimento?: number
}

/** Ficha técnica BOM (contrato API GET/PUT …/services/{id}/supplies). */
export interface ServicoBomItem {
  insumo_id: string
  nome: string
  unidade: string
  quantidade_uso: number
}

export interface ServicoBomInput {
  insumo_id: string
  quantidade_uso: number
}

/** @deprecated Mock-only cost rows; use ServicoBomItem for API BOM. */
export interface ServicoInsumo {
  id: string
  nome: string
  custo: number
}

export type InsumoCategoria = 'CUIDADOS_CAPILARES' | 'TECNICA_UNHAS' | 'ESTETICA' | 'GERAL'

export interface InsumoEstoque {
  id: string
  tenant_id: string
  nome: string
  marca?: string
  categoria: InsumoCategoria
  quantidade: number
  estoque_minimo: number
  estoque_ideal: number
  valor_unitario: number
  unidade: string
  imagem_url?: string
  instrucoes_uso?: string
  ativo: boolean
}

export interface Servico {
  id: string
  tenant_id: string
  nome: string
  duracao_minutos: number
  preco: number
  ativo: boolean
  categoria_id?: string | null
  categoria_nome?: string
  imagem_url?: string
  descricao?: string
  profissional_ids?: string[]
  exibir_catalogo_publico?: boolean
  permitir_agendamento_online?: boolean
  /** @deprecated Mock-only; API BOM vive em ServicoBomItem via …/supplies. */
  insumos?: ServicoInsumo[]
}

export interface AdicionalServico {
  id: string
  servico_id: string
  nome: string
  duracao_minutos: number
  preco: number
}

export interface Agendamento {
  id: string
  tenant_id: string
  profissional_id: string
  servico_id: string
  /** Um ou mais serviços no mesmo agendamento (o primeiro espelha servico_id). */
  servico_ids?: string[]
  adicional_ids: string[]
  cliente_nome: string
  cliente_telefone: string
  data: string
  hora_inicio: string
  status: AgendamentoStatus
  minutos_invadidos?: number
  observacoes?: string
  aceita_adiantar?: boolean
  /** Resumo da oferta de antecipação envolvendo este agendamento (DEV-85). */
  early_slot_offer?: EarlySlotOfferResumo | null
  valor_cobrado?: number
  metodo_pagamento?: MetodoPagamento
  cobrado_em?: string
  confirmacao_cliente?: ConfirmacaoCliente
  ultimo_lembrete_enviado_em?: string | null
  /** Retornada apenas na criação pública; o token nunca aparece em listagens. */
  management_url?: string
}

export interface PublicAppointmentManageResponse {
  appointment: {
    service: string
    professional: string
    starts_at: string
    status: AgendamentoStatus
    customer_confirmation: ConfirmacaoCliente
  }
  establishment: {
    name: string
    contact_phone?: string | null
  }
  cancellation: {
    allowed: boolean
    reason_required: boolean
    minimum_notice_hours: number
    denial_reason?: string | null
  }
}

export interface PublicCancelAppointmentResponse {
  status: 'cancelled'
  appointment_status: 'CANCELADO'
  customer_confirmation: 'CANCELADO_CLIENTE'
  slot_released: boolean
}

export interface NotificacoesAgendaSettings {
  lembretes_ativos: boolean
  antecedencia_confirmacao_horas: number
  antecedencia_lembrete_horas: number
  janela_minima_cancelamento_horas: number
  motivo_cancelamento_obrigatorio: boolean
  template_confirmacao: string
  template_lembrete: string
  allowed_variables: string[]
  whatsapp_status: WhatsAppIntegrationStatus
}

export interface FilaEspera {
  id: string
  tenant_id: string
  profissional_id: string
  cliente_nome: string
  cliente_telefone: string
  status: FilaStatus
  agendamento_id?: string
  data?: string
  hora_desejada?: string
}

export type LancamentoTipo = 'ENTRADA' | 'CUSTO_FIXO' | 'CUSTO_VARIAVEL'

export type TransacaoCategoria =
  | 'SERVICO'
  | 'PRODUTO'
  | 'ALUGUEL'
  | 'SALARIOS'
  | 'SUPRIMENTOS'
  | 'MARKETING'
  | 'OUTRO'

export type TransacaoStatus = 'PAGO' | 'PENDENTE'

export type MetodoPagamento =
  | 'PIX'
  | 'CARTAO_DEBITO'
  | 'CARTAO_CREDITO'
  | 'DINHEIRO'
  | 'TRANSFERENCIA'
  | 'CARTAO'

export type NaturezaValor = 'FIXO' | 'VARIAVEL'

export type FrequenciaRecorrencia = 'SEMANAL' | 'MENSAL' | 'ANUAL'

export interface Lancamento {
  id: string
  tenant_id: string
  tipo: LancamentoTipo
  valor: number
  descricao: string
  data: string
  profissional_id?: string
  status_pagamento?: 'PENDENTE' | 'LIQUIDADO'
  liquidado_em?: string
  categoria?: TransacaoCategoria
  subtitulo?: string
  status_transacao?: TransacaoStatus
  vencimento?: string
  metodo_pagamento?: MetodoPagamento
  fornecedor?: string
  natureza?: NaturezaValor
  recorrente?: boolean
  frequencia_recorrencia?: FrequenciaRecorrencia
}

export interface MockUser {
  id: string
  email: string
  password: string
  role: UserRole
  tenant_id?: string
  profissional_id?: string
  telefone?: string
  nome: string
}

/** Conta global do cliente (login único na plataforma, telefone único). */
export interface ContaCliente {
  id: string
  telefone: string
  email: string
  password: string
  nome: string
  criado_em: string
}

/** Preferências do cliente por salão. */
export interface ClientePreferenciaSalao {
  conta_cliente_id: string
  tenant_id: string
  profissional_favorito_id?: string
}

export interface AuthSession {
  token: string
  user: Omit<MockUser, 'password'>
}

export interface Cliente {
  id: string
  tenant_id: string
  nome: string
  telefone: string
  email?: string
  ativo: boolean
  criado_em: string
  data_nascimento?: string
  foto_url?: string
  alergias?: string[]
  observacoes_medicas?: string
  preferencias?: string[]
}

export interface ClienteNotaInterna {
  id: string
  cliente_id: string
  autor: string
  data: string
  texto: string
}

export interface ClienteGaleriaItem {
  id: string
  cliente_id: string
  url: string
  legenda: string
  data: string
}

export type FaturaSaasStatus = 'SUCESSO' | 'PENDENTE' | 'FALHA'

/** Cobrança mensal SaaS de um salão (visão Super Admin). */
export interface FaturaSaas {
  id: string
  tenant_id: string
  valor: number
  data: string
  status: FaturaSaasStatus
  referencia_mes: string
}

export interface MockDatabase {
  planos: PlanoSaas[]
  tenants: Tenant[]
  faturas_saas: FaturaSaas[]
  categorias: CategoriaServico[]
  clientes: Cliente[]
  especialidades: Especialidade[]
  profissionais: Profissional[]
  servicos: Servico[]
  insumos: InsumoEstoque[]
  adicionais: AdicionalServico[]
  agendamentos: Agendamento[]
  fila_espera: FilaEspera[]
  lancamentos: Lancamento[]
  users: MockUser[]
  contas_cliente: ContaCliente[]
  cliente_preferencias: ClientePreferenciaSalao[]
  cliente_notas: ClienteNotaInterna[]
  cliente_galeria: ClienteGaleriaItem[]
}

export class PlanLimitExceededError extends Error {
  constructor(message = 'Limite de profissionais do plano atingido') {
    super(message)
    this.name = 'ErrPlanLimitExceeded'
  }
}
