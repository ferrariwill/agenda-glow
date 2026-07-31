import { useCallback, useEffect, useMemo, useState } from 'react'
import { BellRing, Loader2 } from 'lucide-react'
import { Alert } from '../ui/Alert'
import { Button } from '../ui/Button'
import { Input } from '../ui/Input'
import {
  getNotificacoesAgenda,
  putNotificacoesAgenda,
  type NotificacoesAgendaUpdate,
} from '../../services/notificacoesAgendaService'
import type { NotificacoesAgendaSettings } from '../../types'

const examples: Record<string, string> = {
  nome_salao: 'Studio Glow',
  servico: 'Corte',
  profissional: 'Ana',
  data_hora: '01/08 às 14:00',
  endereco: 'Rua das Flores, 100',
  link_gestao: 'agendaglow.com/p/agendamento/…',
}

function interpolate(template: string, allowed: string[]) {
  return allowed.reduce((text, variable) => {
    const value = examples[variable] ?? variable
    return text
      .replaceAll(`{{${variable}}}`, value)
      .replaceAll(`{${variable}}`, value)
  }, template)
}

export function LembretesAgendaConfig({ estabelecimentoId }: { estabelecimentoId: string }) {
  const [settings, setSettings] = useState<NotificacoesAgendaSettings | null>(null)
  const [form, setForm] = useState<NotificacoesAgendaUpdate | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)

  const load = useCallback(async () => {
    if (!estabelecimentoId) return
    setLoading(true)
    setError('')
    try {
      const data = await getNotificacoesAgenda(estabelecimentoId)
      setSettings(data)
      const { allowed_variables: _allowed, whatsapp_status: _status, ...editable } = data
      void _allowed
      void _status
      setForm(editable)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Não foi possível carregar as notificações.')
    } finally {
      setLoading(false)
    }
  }, [estabelecimentoId])

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- carga assíncrona inicial do painel
    void load()
  }, [load])

  const preview = useMemo(() => {
    if (!form || !settings) return []
    return [
      interpolate(form.template_confirmacao, settings.allowed_variables),
      interpolate(form.template_lembrete, settings.allowed_variables),
    ]
  }, [form, settings])

  const update = <K extends keyof NotificacoesAgendaUpdate>(
    key: K,
    value: NotificacoesAgendaUpdate[K],
  ) => setForm((current) => current ? { ...current, [key]: value } : current)

  const save = async () => {
    if (!form) return
    setSaving(true)
    setError('')
    setSaved(false)
    try {
      await putNotificacoesAgenda(estabelecimentoId, form)
      await load()
      setSaved(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Não foi possível salvar as notificações.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="mt-6 rounded-xl border border-[#e5d3c8]/40 bg-white p-6 shadow-[0px_4px_20px_rgba(183,132,114,0.08)] sm:p-8">
      <div className="flex items-start gap-4">
        <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-aura-primary/10 text-aura-primary">
          <BellRing className="h-6 w-6" />
        </div>
        <div>
          <h2 className="font-display text-xl font-semibold text-aura-anthracite">
            Confirmações e lembretes
          </h2>
          <p className="mt-1 text-sm text-aura-muted">
            Configure quando o salão envia mensagens e a janela mínima de cancelamento.
          </p>
        </div>
      </div>

      {loading && (
        <div className="mt-6 flex items-center gap-2 text-sm text-aura-muted">
          <Loader2 className="h-4 w-4 animate-spin" /> Carregando configurações…
        </div>
      )}

      {!loading && settings?.whatsapp_status !== 'CONECTADO' && (
        <Alert variant="warning" className="mt-6">
          O WhatsApp não está conectado. As configurações serão salvas, mas os envios não ocorrerão.
        </Alert>
      )}
      {error && <Alert variant="error" className="mt-6">{error}</Alert>}
      {saved && <Alert className="mt-6">Configurações salvas.</Alert>}

      {form && settings && (
        <div className="mt-6 space-y-6">
          <label className="flex items-center gap-3 text-sm font-medium text-aura-anthracite">
            <input
              type="checkbox"
              checked={form.lembretes_ativos}
              onChange={(event) => update('lembretes_ativos', event.target.checked)}
            />
            Ativar confirmações e lembretes automáticos
          </label>

          <div className="grid gap-4 sm:grid-cols-3">
            <Input
              type="number"
              min={1}
              max={168}
              label="Confirmação (horas antes)"
              value={form.antecedencia_confirmacao_horas}
              onChange={(event) => update('antecedencia_confirmacao_horas', Number(event.target.value))}
            />
            <Input
              type="number"
              min={1}
              max={48}
              label="Lembrete (horas antes)"
              value={form.antecedencia_lembrete_horas}
              onChange={(event) => update('antecedencia_lembrete_horas', Number(event.target.value))}
            />
            <Input
              type="number"
              min={0}
              max={168}
              label="Cancelamento (horas antes)"
              value={form.janela_minima_cancelamento_horas}
              onChange={(event) => update('janela_minima_cancelamento_horas', Number(event.target.value))}
            />
          </div>

          <label className="flex items-center gap-3 text-sm font-medium text-aura-anthracite">
            <input
              type="checkbox"
              checked={form.motivo_cancelamento_obrigatorio}
              onChange={(event) => update('motivo_cancelamento_obrigatorio', event.target.checked)}
            />
            Exigir motivo no cancelamento do cliente
          </label>

          <div className="grid gap-4 lg:grid-cols-2">
            {([
              ['template_confirmacao', 'Mensagem de confirmação'],
              ['template_lembrete', 'Mensagem de lembrete'],
            ] as const).map(([key, label], index) => (
              <div key={key} className="space-y-2">
                <label className="text-sm font-medium text-aura-anthracite">{label}</label>
                <textarea
                  rows={4}
                  value={form[key]}
                  onChange={(event) => update(key, event.target.value)}
                  className="w-full rounded-lg border border-aura-border px-3 py-2.5 text-sm"
                />
                <div className="rounded-lg bg-aura-surface p-3 text-xs text-aura-muted">
                  <strong className="block text-aura-anthracite">Prévia</strong>
                  {preview[index]}
                </div>
              </div>
            ))}
          </div>

          <p className="text-xs text-aura-muted">
            Variáveis disponíveis: {settings.allowed_variables.map((item) => `{{${item}}}`).join(', ')}
          </p>

          <Button loading={saving} onClick={() => void save()}>Salvar configurações</Button>
        </div>
      )}
    </section>
  )
}
