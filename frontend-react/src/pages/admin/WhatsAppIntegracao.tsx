import { useCallback, useEffect, useState } from 'react'
import { CheckCircle2, ExternalLink, Loader2, MessageCircle, RefreshCw, Shield } from 'lucide-react'
import { DonaLayout, DonaFooter, PageHeader } from '../../components/dona/DonaLayout'
import { NotificacoesAgendaPanel } from '../../components/dona/NotificacoesAgendaPanel'
import { Alert } from '../../components/ui/Alert'
import { Badge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { useAuth } from '../../contexts/AuthContext'
import { IS_MOCK } from '../../lib/config'
import { apiFetch } from '../../lib/api'
import {
  buildWhatsAppEmbeddedSignupURL,
  getWhatsAppIntegration,
  markWhatsAppConnected,
  markWhatsAppPending,
  type WhatsAppIntegrationStatus,
} from '../../utils/whatsappIntegration'

type IntegrationView = {
  estabelecimento_id: string
  status: WhatsAppIntegrationStatus
  state: string
  signup_url: string
  waba_id?: string | null
  phone_number_id?: string | null
  connected_at?: string | null
}

function statusBadge(status: WhatsAppIntegrationStatus) {
  switch (status) {
    case 'CONECTADO':
      return { variant: 'success' as const, label: 'Conectado' }
    case 'PENDENTE':
      return { variant: 'warning' as const, label: 'Aguardando conexão' }
    default:
      return { variant: 'muted' as const, label: 'Desconectado' }
  }
}

export function WhatsAppIntegracao() {
  const { session } = useAuth()
  const tenantId = session?.user.tenant_id ?? ''
  const [view, setView] = useState<IntegrationView | null>(null)
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [loading, setLoading] = useState(true)
  const [connecting, setConnecting] = useState(false)

  const load = useCallback(async () => {
    if (!tenantId) return
    setError('')
    try {
      if (IS_MOCK) {
        const local = getWhatsAppIntegration(tenantId)
        setView(local)
        return
      }
      const data = await apiFetch<IntegrationView>('/api/v1/whatsapp/integration')
      setView(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Falha ao carregar integração WhatsApp')
    } finally {
      setLoading(false)
    }
  }, [tenantId])

  useEffect(() => {
    void load()
  }, [load])

  // Enquanto pendente, atualiza status (Gateway → webhook → CONECTADO).
  useEffect(() => {
    if (!view || view.status !== 'PENDENTE') return
    const tick = window.setInterval(() => {
      void load()
    }, 4000)
    const onFocus = () => {
      void load()
    }
    window.addEventListener('focus', onFocus)
    return () => {
      window.clearInterval(tick)
      window.removeEventListener('focus', onFocus)
    }
  }, [view?.status, load])

  const openEmbeddedSignup = async () => {
    if (!tenantId || !view) return
    setConnecting(true)
    setError('')
    setInfo('')
    try {
      let signupURL = view.signup_url
      if (IS_MOCK) {
        markWhatsAppPending(tenantId)
        signupURL = buildWhatsAppEmbeddedSignupURL(tenantId)
        setView(getWhatsAppIntegration(tenantId))
      } else {
        const started = await apiFetch<IntegrationView>('/api/v1/whatsapp/integration/start', {
          method: 'POST',
        })
        setView(started)
        signupURL = started.signup_url
      }

      // Nova aba com a URL da Meta + &state=beleza_{idDoSalao}
      window.open(signupURL, '_blank', 'noopener,noreferrer')
      setInfo(
        'Aba do Embedded Signup aberta. Conclua o fluxo na Meta; o status mudará para Conectado quando o Gateway confirmar.',
      )
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Não foi possível iniciar a conexão')
    } finally {
      setConnecting(false)
    }
  }

  const simularGateway = () => {
    if (!tenantId) return
    markWhatsAppConnected(tenantId, { waba_id: 'mock-waba', phone_number_id: 'mock-phone' })
    setView(getWhatsAppIntegration(tenantId))
    setInfo('Simulação: Gateway confirmou a conexão com sucesso.')
  }

  const badge = statusBadge(view?.status ?? 'DESCONECTADO')

  return (
    <DonaLayout>
      <PageHeader
        title="WhatsApp Oficial"
        subtitle="Conecte o número do salão via Embedded Signup da Meta para enviar confirmações e lembretes."
      />

      {(error || info) && (
        <Alert
          variant={error ? 'error' : 'info'}
          className="mb-6"
          onDismiss={() => {
            setError('')
            setInfo('')
          }}
        >
          {error || info}
        </Alert>
      )}

      <section className="rounded-xl border border-[#e5d3c8]/40 bg-white p-6 shadow-[0px_4px_20px_rgba(183,132,114,0.08)] sm:p-8">
        <div className="flex flex-col gap-6 sm:flex-row sm:items-start sm:justify-between">
          <div className="flex gap-4">
            <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-[#25D366]/15 text-[#128C7E]">
              <MessageCircle className="h-6 w-6" />
            </div>
            <div>
              <div className="mb-2 flex flex-wrap items-center gap-2">
                <h2 className="font-display text-xl font-semibold text-aura-anthracite">
                  Integração WhatsApp Business
                </h2>
                <Badge variant={badge.variant}>{badge.label}</Badge>
              </div>
              <p className="max-w-xl text-sm text-aura-muted">
                O botão abre o Embedded Signup da Meta com o parâmetro{' '}
                <code className="rounded bg-aura-surface px-1.5 py-0.5 text-xs">
                  state=beleza_{tenantId || 'idDoSalao'}
                </code>
                , identificando este salão no Gateway.
              </p>
            </div>
          </div>

          <div className="flex shrink-0 flex-col gap-2 sm:items-end">
            <Button
              type="button"
              onClick={() => void openEmbeddedSignup()}
              loading={connecting}
              disabled={loading || !tenantId || view?.status === 'CONECTADO'}
              className="!bg-[#128C7E] hover:!bg-[#0e6e62]"
            >
              <MessageCircle className="mr-2 h-4 w-4" />
              Conectar WhatsApp Oficial
              <ExternalLink className="ml-2 h-3.5 w-3.5 opacity-80" />
            </Button>
            <button
              type="button"
              onClick={() => void load()}
              className="inline-flex items-center gap-1.5 text-xs font-medium text-aura-muted hover:text-[#7d5141]"
            >
              <RefreshCw className="h-3.5 w-3.5" />
              Atualizar status
            </button>
          </div>
        </div>

        {loading ? (
          <div className="mt-8 flex items-center gap-2 text-sm text-aura-muted">
            <Loader2 className="h-4 w-4 animate-spin" />
            Carregando integração…
          </div>
        ) : (
          <dl className="mt-8 grid gap-4 border-t border-[#efdcd1]/40 pt-6 sm:grid-cols-2">
            <div>
              <dt className="text-xs font-semibold uppercase tracking-wider text-aura-muted">
                State (Meta)
              </dt>
              <dd className="mt-1 break-all font-mono text-sm text-aura-anthracite">
                {view?.state ?? `beleza_${tenantId}`}
              </dd>
            </div>
            <div>
              <dt className="text-xs font-semibold uppercase tracking-wider text-aura-muted">
                ID do salão
              </dt>
              <dd className="mt-1 break-all font-mono text-sm text-aura-anthracite">
                {view?.estabelecimento_id ?? tenantId}
              </dd>
            </div>
            {view?.waba_id && (
              <div>
                <dt className="text-xs font-semibold uppercase tracking-wider text-aura-muted">
                  WABA ID
                </dt>
                <dd className="mt-1 font-mono text-sm">{view.waba_id}</dd>
              </div>
            )}
            {view?.phone_number_id && (
              <div>
                <dt className="text-xs font-semibold uppercase tracking-wider text-aura-muted">
                  Phone Number ID
                </dt>
                <dd className="mt-1 font-mono text-sm">{view.phone_number_id}</dd>
              </div>
            )}
            {view?.connected_at && (
              <div>
                <dt className="text-xs font-semibold uppercase tracking-wider text-aura-muted">
                  Conectado em
                </dt>
                <dd className="mt-1 text-sm">
                  {new Date(view.connected_at).toLocaleString('pt-BR')}
                </dd>
              </div>
            )}
          </dl>
        )}

        {view?.status === 'CONECTADO' && (
          <div className="mt-6 flex items-start gap-3 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-900">
            <CheckCircle2 className="mt-0.5 h-5 w-5 shrink-0" />
            <p>
              WhatsApp Oficial conectado. O Gateway já pode enviar e receber mensagens em nome deste
              salão.
            </p>
          </div>
        )}

        {view?.status === 'PENDENTE' && (
          <div className="mt-6 flex items-start gap-3 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-950">
            <Shield className="mt-0.5 h-5 w-5 shrink-0" />
            <div>
              <p>
                Aguardando o Gateway confirmar o Embedded Signup (
                <code className="text-xs">POST /api/v1/webhook/whatsapp-connected</code>).
              </p>
              {IS_MOCK && (
                <button
                  type="button"
                  onClick={simularGateway}
                  className="mt-2 text-xs font-semibold underline hover:no-underline"
                >
                  Simular callback do Gateway (somente mock)
                </button>
              )}
            </div>
          </div>
        )}
      </section>

      {tenantId && <NotificacoesAgendaPanel estabelecimentoId={tenantId} />}

      <DonaFooter />
    </DonaLayout>
  )
}
