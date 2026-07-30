import { useCallback, useEffect, useState } from 'react'
import { CheckCircle2, Loader2 } from 'lucide-react'
import { useParams } from 'react-router-dom'
import { GestaoAgendamentoDetalhe } from '../../components/public/gestao/GestaoAgendamentoDetalhe'
import { GestaoCancelamentoPainel } from '../../components/public/gestao/GestaoCancelamentoPainel'
import { GestaoEstadoMensagem } from '../../components/public/gestao/GestaoEstadoMensagem'
import { Button } from '../../components/ui/Button'
import { Card } from '../../components/ui/Card'
import { ApiError } from '../../lib/api'
import { cancelManage, getManage } from '../../services/publicAppointmentManage'
import type { PublicAppointmentManageResponse } from '../../types'

type ViewState =
  | { kind: 'loading' }
  | { kind: 'not_found' | 'expired' | 'unexpected'; message?: string }
  | { kind: 'ready' | 'window_closed' | 'terminal'; data: PublicAppointmentManageResponse }
  | { kind: 'success'; data: PublicAppointmentManageResponse; slotReleased: boolean }

function stateFromData(data: PublicAppointmentManageResponse): ViewState {
  if (data.cancellation.allowed) return { kind: 'ready', data }
  if (data.cancellation.denial_reason === 'cancellation_window_closed') {
    return { kind: 'window_closed', data }
  }
  return { kind: 'terminal', data }
}

function denialMessage(reason?: string | null) {
  if (reason === 'cancellation_window_closed') {
    return 'O horário está muito próximo para ser cancelado por este link.'
  }
  if (reason === 'appointment_not_cancellable') {
    return 'Este agendamento já foi cancelado ou concluído.'
  }
  return reason ?? undefined
}

export function GestaoAgendamentoPublica() {
  const { token = '' } = useParams<{ token: string }>()
  const [state, setState] = useState<ViewState>({ kind: 'loading' })
  const [dialogOpen, setDialogOpen] = useState(false)
  const [motivo, setMotivo] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [dialogError, setDialogError] = useState('')

  const load = useCallback(async () => {
    if (!token) {
      setState({ kind: 'not_found' })
      return
    }
    setState({ kind: 'loading' })
    try {
      setState(stateFromData(await getManage(token)))
    } catch (error) {
      if (error instanceof ApiError && error.status === 404) setState({ kind: 'not_found' })
      else if (error instanceof ApiError && error.status === 410) setState({ kind: 'expired' })
      else setState({ kind: 'unexpected', message: error instanceof Error ? error.message : undefined })
    }
  }, [token])

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- carga assíncrona pelo token da rota
    void load()
  }, [load])

  const cancel = async () => {
    if (state.kind !== 'ready') return
    setSubmitting(true)
    setDialogError('')
    try {
      const result = await cancelManage(token, { motivo })
      setDialogOpen(false)
      setState({ kind: 'success', data: state.data, slotReleased: result.slot_released })
    } catch (error) {
      if (error instanceof ApiError && error.code === 'reason_required') {
        setDialogError('Informe o motivo para continuar.')
      } else if (error instanceof ApiError && error.status === 422) {
        setDialogOpen(false)
        setState({ kind: 'window_closed', data: state.data })
      } else if (error instanceof ApiError && error.status === 409) {
        setDialogOpen(false)
        setState({ kind: 'terminal', data: state.data })
      } else if (error instanceof ApiError && error.status === 404) {
        setDialogOpen(false)
        setState({ kind: 'not_found' })
      } else if (error instanceof ApiError && error.status === 410) {
        setDialogOpen(false)
        setState({ kind: 'expired' })
      } else {
        setDialogError(error instanceof Error ? error.message : 'Não foi possível cancelar.')
      }
    } finally {
      setSubmitting(false)
    }
  }

  const data = 'data' in state ? state.data : undefined

  return (
    <main className="min-h-screen bg-aura-surface px-4 py-8 sm:py-12">
      <div className="mx-auto max-w-lg space-y-4">
        {state.kind === 'loading' && (
          <Card className="flex items-center justify-center gap-2 py-12 text-sm text-aura-muted">
            <Loader2 className="h-5 w-5 animate-spin" /> Carregando seu horário…
          </Card>
        )}

        {(state.kind === 'not_found' || state.kind === 'expired' || state.kind === 'unexpected') && (
          <GestaoEstadoMensagem kind={state.kind} message={state.message} />
        )}

        {data && <GestaoAgendamentoDetalhe data={data} />}

        {state.kind === 'ready' && (
          <Card>
            <p className="text-sm text-aura-muted">
              Você pode cancelar até {state.data.cancellation.minimum_notice_hours} hora(s) antes.
            </p>
            <Button variant="danger" fullWidth className="mt-4" onClick={() => setDialogOpen(true)}>
              Cancelar meu horário
            </Button>
          </Card>
        )}

        {(state.kind === 'window_closed' || state.kind === 'terminal') && (
          <GestaoEstadoMensagem
            kind={state.kind}
            message={denialMessage(state.data.cancellation.denial_reason)}
            minimumNoticeHours={state.data.cancellation.minimum_notice_hours}
            contactPhone={state.data.establishment.contact_phone}
          />
        )}

        {state.kind === 'success' && (
          <Card className="text-center">
            <CheckCircle2 className="mx-auto h-12 w-12 text-emerald-600" />
            <h2 className="mt-3 font-display text-2xl font-semibold">Horário liberado</h2>
            <p className="mt-2 text-sm text-aura-muted">
              {state.slotReleased
                ? 'Seu cancelamento foi concluído e o horário já está disponível.'
                : 'Seu cancelamento foi concluído.'}
            </p>
          </Card>
        )}

        <GestaoCancelamentoPainel
          open={dialogOpen}
          reasonRequired={state.kind === 'ready' && state.data.cancellation.reason_required}
          motivo={motivo}
          error={dialogError}
          loading={submitting}
          onMotivoChange={setMotivo}
          onClose={() => setDialogOpen(false)}
          onConfirm={() => void cancel()}
        />
      </div>
    </main>
  )
}
