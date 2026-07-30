import { apiFetch } from '../lib/api'
import { IS_MOCK } from '../lib/config'
import type { NotificacoesAgendaSettings } from '../types'

export type NotificacoesAgendaUpdate = Omit<
  NotificacoesAgendaSettings,
  'allowed_variables' | 'whatsapp_status'
>

const defaultSettings: NotificacoesAgendaSettings = {
  lembretes_ativos: true,
  antecedencia_confirmacao_horas: 24,
  antecedencia_lembrete_horas: 1,
  janela_minima_cancelamento_horas: 2,
  motivo_cancelamento_obrigatorio: false,
  template_confirmacao:
    'Olá! Confirme seu horário de {{servico}} com {{profissional}} em {{data_hora}}. {{link_gestao}}',
  template_lembrete:
    'Lembrete: seu horário de {{servico}} é em {{data_hora}} no {{nome_salao}}.',
  allowed_variables: [
    'nome_salao',
    'servico',
    'profissional',
    'data_hora',
    'endereco',
    'link_gestao',
  ],
  whatsapp_status: 'CONECTADO',
}

function storageKey(estabelecimentoId: string) {
  return `agendaglow_notificacoes_agenda_${estabelecimentoId}`
}

export async function getNotificacoesAgenda(
  estabelecimentoId: string,
): Promise<NotificacoesAgendaSettings> {
  if (!IS_MOCK) {
    return apiFetch(
      `/api/v1/estabelecimentos/${encodeURIComponent(estabelecimentoId)}/notificacoes-agenda`,
    )
  }
  const saved = localStorage.getItem(storageKey(estabelecimentoId))
  return saved
    ? { ...defaultSettings, ...(JSON.parse(saved) as NotificacoesAgendaUpdate) }
    : { ...defaultSettings }
}

export async function putNotificacoesAgenda(
  estabelecimentoId: string,
  body: NotificacoesAgendaUpdate,
): Promise<NotificacoesAgendaSettings> {
  if (!IS_MOCK) {
    return apiFetch(
      `/api/v1/estabelecimentos/${encodeURIComponent(estabelecimentoId)}/notificacoes-agenda`,
      { method: 'PUT', body: JSON.stringify(body) },
    )
  }
  localStorage.setItem(storageKey(estabelecimentoId), JSON.stringify(body))
  return { ...defaultSettings, ...body }
}
