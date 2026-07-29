import { formatDateTimeBR } from '../utils/format'

/**
 * Envio Beleza → Gateway.
 * Em produção o backend Go chama POST {WHATSAPP_GATEWAY_URL}/send-notification.
 * No front (mock/dev), tenta o mesmo contrato via proxy relativo se VITE_WHATSAPP_GATEWAY_URL existir.
 */
const GATEWAY_URL = (
  (import.meta.env.VITE_WHATSAPP_GATEWAY_URL as string | undefined)?.trim() ||
  ''
).replace(/\/$/, '')

export interface WhatsAppPayload {
  telefone: string
  mensagem: string
  metadata?: Record<string, string>
}

async function postSendNotification(body: {
  sistema_origem: 'beleza'
  tenant_id: string
  phone_number: string
  template_name: string
  language_code?: string
  simple_template?: boolean
  appointment_id?: string
  variables: string[]
}): Promise<{ ok: boolean; id?: string }> {
  if (!GATEWAY_URL) {
    console.warn('[whatsappService] Gateway não configurado — simulando envio', body)
    return { ok: true, id: `mock-${Date.now()}` }
  }
  try {
    const res = await fetch(`${GATEWAY_URL}/send-notification`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        // Key só no backend; no browser o Gateway deve aceitar CORS + outra auth, ou use API Beleza.
      },
      body: JSON.stringify({
        language_code: 'pt_BR',
        simple_template: true,
        ...body,
      }),
    })
    if (!res.ok) {
      console.warn(`[whatsappService] /send-notification → HTTP ${res.status}`)
      return { ok: false }
    }
    const data = (await res.json().catch(() => ({}))) as { id?: string }
    return { ok: true, id: data.id }
  } catch (err) {
    console.warn('[whatsappService] offline — simulando envio', err)
    return { ok: true, id: `mock-${Date.now()}` }
  }
}

export interface ConfirmacaoAgendamentoData {
  telefone: string
  clienteNome: string
  servico: string
  profissional: string
  data: string
  hora: string
  tenantId?: string
  appointmentId?: string
}

export async function enviarConfirmacaoAgendamento(
  data: ConfirmacaoAgendamentoData,
): Promise<{ ok: boolean; id?: string }> {
  return postSendNotification({
    sistema_origem: 'beleza',
    tenant_id: data.tenantId ?? 'unknown',
    phone_number: data.telefone,
    template_name: 'confirmacao_agendamento',
    appointment_id: data.appointmentId,
    variables: [
      data.clienteNome,
      formatDateTimeBR(data.data, data.hora),
      data.servico,
      data.profissional,
    ],
  })
}

export interface AlertaEncaixeData {
  telefone: string
  clienteNome: string
  linkAprovacao: string
  minutosInvadidos: number
  tenantId?: string
  appointmentId?: string
}

export async function enviarAlertaEncaixe(
  data: AlertaEncaixeData,
): Promise<{ ok: boolean; id?: string }> {
  return postSendNotification({
    sistema_origem: 'beleza',
    tenant_id: data.tenantId ?? 'unknown',
    phone_number: data.telefone,
    template_name: 'alerta_encaixe',
    appointment_id: data.appointmentId,
    variables: [data.clienteNome, String(data.minutosInvadidos), data.linkAprovacao],
  })
}

export interface OfertaVagaFilaData {
  telefone: string
  clienteNome: string
  profissional: string
  data: string
  hora: string
  linkConfirmacao: string
  tenantId?: string
}

export async function enviarOfertaVagaFila(
  data: OfertaVagaFilaData,
): Promise<{ ok: boolean; id?: string }> {
  return postSendNotification({
    sistema_origem: 'beleza',
    tenant_id: data.tenantId ?? 'unknown',
    phone_number: data.telefone,
    template_name: 'oferta_vaga_fila',
    variables: [
      data.clienteNome,
      data.profissional,
      formatDateTimeBR(data.data, data.hora),
      data.linkConfirmacao,
    ],
  })
}

export interface ConviteReativacaoData {
  telefone: string
  clienteNome: string
  salaoNome: string
  linkAgendamento: string
  tenantId?: string
}

export async function enviarConviteReativacao(
  data: ConviteReativacaoData,
): Promise<{ ok: boolean; id?: string }> {
  return postSendNotification({
    sistema_origem: 'beleza',
    tenant_id: data.tenantId ?? 'unknown',
    phone_number: data.telefone,
    template_name: 'convite_reativacao',
    variables: [data.clienteNome, data.salaoNome, data.linkAgendamento],
  })
}
