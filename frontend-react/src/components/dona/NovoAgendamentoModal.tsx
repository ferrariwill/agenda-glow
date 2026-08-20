import { useEffect, useMemo, useState } from 'react'
import { Alert } from '../ui/Alert'
import { Button } from '../ui/Button'
import { DateInput } from '../ui/DateInput'
import { Input } from '../ui/Input'
import { Modal } from '../ui/Modal'
import { TimeInput } from '../ui/TimeInput'
import { ApiError } from '../../lib/api'
import { enviarAlertaEncaixe, enviarConfirmacaoAgendamento } from '../../services/whatsappService'
import type { UserRole } from '../../types'
import {
  AgendaConflitoError,
  createAgendamento,
  createCliente,
  getDb,
  getTenantClients,
} from '../../utils/mockDb'
import { formatBRL, todayISO } from '../../utils/format'

export interface NovoAgendamentoPreset {
  data?: string
  hora_inicio?: string
  profissional_id?: string
  cliente_nome?: string
  cliente_telefone?: string
  cliente_id?: string
  servico_ids?: string[]
}

interface Props {
  open: boolean
  onClose: () => void
  tenantId: string
  preset?: NovoAgendamentoPreset
  onSuccess?: (msg: string) => void
  profissionalFixo?: string
  cadastrarClienteNovo?: boolean
  /** Contexto de API — só AgendaProfissional passa PROFISSIONAL. */
  apiRole?: UserRole
}

export function NovoAgendamentoModal({
  open,
  onClose,
  tenantId,
  preset,
  onSuccess,
  profissionalFixo,
  cadastrarClienteNovo = false,
  apiRole,
}: Props) {
  const db = getDb()
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const profissionais = db.profissionais.filter((p) => p.tenant_id === tenantId && p.ativo)
  const servicos = db.servicos.filter((s) => s.tenant_id === tenantId && s.ativo)
  const clientes = getTenantClients(tenantId).filter((c) => c.ativo)

  const [clienteModo, setClienteModo] = useState<'cadastrado' | 'novo'>('cadastrado')
  const [clienteId, setClienteId] = useState('')
  const [clienteNome, setClienteNome] = useState('')
  const [clienteTel, setClienteTel] = useState('')
  const [profId, setProfId] = useState('')
  const [servicoIds, setServicoIds] = useState<string[]>([])
  const [data, setData] = useState(todayISO())
  const [hora, setHora] = useState('09:00')

  useEffect(() => {
    if (!open) return
    setError('')
    setData(preset?.data ?? todayISO())
    setHora(preset?.hora_inicio ?? '09:00')
    setProfId(profissionalFixo ?? preset?.profissional_id ?? profissionais[0]?.id ?? '')
    const presetServicos =
      preset?.servico_ids?.length ? preset.servico_ids : servicos[0]?.id ? [servicos[0].id] : []
    setServicoIds(presetServicos)
    if (preset?.cliente_nome) {
      setClienteModo('novo')
      setClienteNome(preset.cliente_nome)
      setClienteTel(preset.cliente_telefone ?? '')
    } else if (preset?.cliente_id) {
      setClienteModo('cadastrado')
      setClienteId(preset.cliente_id)
    } else {
      setClienteModo(clientes.length > 0 ? 'cadastrado' : 'novo')
      setClienteId(clientes[0]?.cliente_id ?? '')
    }
  }, [open, preset, profissionalFixo])

  const profissionalEfetivo = profissionalFixo ?? profId

  const clienteSelecionado = useMemo(
    () => clientes.find((c) => c.cliente_id === clienteId),
    [clientes, clienteId],
  )

  const servicosSelecionados = useMemo(
    () => servicos.filter((s) => servicoIds.includes(s.id)),
    [servicos, servicoIds],
  )

  const totalDuracao = servicosSelecionados.reduce((s, srv) => s + srv.duracao_minutos, 0)
  const totalPreco = servicosSelecionados.reduce((s, srv) => s + srv.preco, 0)

  const toggleServico = (id: string) => {
    setServicoIds((prev) => {
      if (prev.includes(id)) {
        const next = prev.filter((x) => x !== id)
        return next.length > 0 ? next : prev
      }
      return [...prev, id]
    })
  }

  const resolverCliente = () => {
    if (clienteModo === 'cadastrado' && clienteSelecionado) {
      return { nome: clienteSelecionado.nome, telefone: clienteSelecionado.telefone }
    }
    return { nome: clienteNome.trim(), telefone: clienteTel.replace(/\D/g, '') }
  }

  const handleSave = async () => {
    setError('')
    const cliente = resolverCliente()
    if (clienteModo === 'cadastrado' && !clienteSelecionado) {
      setError('Selecione um cliente cadastrado ou use Novo / avulso')
      return
    }
    if (
      !cliente.nome ||
      !cliente.telefone ||
      !profissionalEfetivo ||
      servicoIds.length === 0 ||
      !data ||
      !hora
    ) {
      setError(
        profissionalFixo
          ? 'Preencha cliente, ao menos um serviço, data e horário'
          : 'Preencha cliente, profissional, ao menos um serviço, data e horário',
      )
      return
    }
    setLoading(true)
    try {
      if (cadastrarClienteNovo && clienteModo === 'novo') {
        try {
          await createCliente(tenantId, { nome: cliente.nome, telefone: cliente.telefone })
        } catch (err) {
          const msg = err instanceof Error ? err.message : ''
          if (!msg.includes('telefone')) throw err
        }
      }

      const ag = await createAgendamento(
        {
          tenant_id: tenantId,
          profissional_id: profissionalEfetivo,
          servico_ids: servicoIds,
          cliente_nome: cliente.nome,
          cliente_telefone: cliente.telefone,
          data,
          hora_inicio: hora,
        },
        apiRole ? { role: apiRole } : undefined,
      )
      const nomesServicos = servicosSelecionados.map((s) => s.nome).join(' + ')
      const prof = profissionais.find((p) => p.id === profissionalEfetivo)
      if (ag.status === 'EM_APROVACAO') {
        await enviarAlertaEncaixe({
          telefone: cliente.telefone,
          clienteNome: cliente.nome,
          linkAprovacao: `${window.location.origin}/publico/aprovacao/${ag.id}`,
          minutosInvadidos: ag.minutos_invadidos ?? 1,
        })
        onSuccess?.('Agendamento criado — aguardando aprovação do cliente (encaixe).')
      } else {
        await enviarConfirmacaoAgendamento({
          telefone: cliente.telefone,
          clienteNome: cliente.nome,
          servico: nomesServicos,
          profissional: prof?.nome ?? '',
          data,
          hora,
        })
        onSuccess?.('Agendamento confirmado e WhatsApp enviado!')
      }
      onClose()
    } catch (err) {
      if (err instanceof AgendaConflitoError) {
        setError(err.message)
      } else if (
        err instanceof ApiError &&
        err.status === 409 &&
        (err.code === 'slot_unavailable' || apiRole === 'PROFISSIONAL')
      ) {
        setError('Esse horário acabou de ser ocupado. Escolha outro.')
      } else {
        setError(err instanceof Error ? err.message : 'Erro ao agendar')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Novo agendamento"
      maxWidthClass="max-w-lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Cancelar
          </Button>
          <Button onClick={handleSave} loading={loading}>
            Confirmar agendamento
          </Button>
        </>
      }
    >
      {error && <Alert variant="error" className="mb-4">{error}</Alert>}

      <div className="space-y-4">
        <div>
          <label className="mb-1.5 block text-sm font-medium">Cliente</label>
          <div className="mb-2 flex gap-2">
            <button
              type="button"
              onClick={() => setClienteModo('cadastrado')}
              className={`rounded-lg px-3 py-1 text-xs ${clienteModo === 'cadastrado' ? 'bg-aura-primary text-white' : 'bg-aura-surface'}`}
            >
              Cadastrado
            </button>
            <button
              type="button"
              onClick={() => setClienteModo('novo')}
              className={`rounded-lg px-3 py-1 text-xs ${clienteModo === 'novo' ? 'bg-aura-primary text-white' : 'bg-aura-surface'}`}
            >
              Novo / avulso
            </button>
          </div>
          {clienteModo === 'cadastrado' ? (
            clientes.length === 0 ? (
              <p className="rounded-lg bg-aura-surface px-3 py-2 text-sm text-aura-muted">
                Nenhum cliente cadastrado. Use &quot;Novo / avulso&quot; para agendar.
              </p>
            ) : (
              <select
                value={clienteId}
                onChange={(e) => setClienteId(e.target.value)}
                className="w-full rounded-lg border border-aura-border px-3 py-2.5 text-sm"
              >
                {clientes.map((c) => (
                  <option key={c.cliente_id} value={c.cliente_id}>
                    {c.nome} — {c.telefone}
                  </option>
                ))}
              </select>
            )
          ) : (
            <div className="space-y-2">
              <Input label="Nome" value={clienteNome} onChange={(e) => setClienteNome(e.target.value)} />
              <Input label="WhatsApp" value={clienteTel} onChange={(e) => setClienteTel(e.target.value)} />
            </div>
          )}
        </div>

        {!profissionalFixo && (
          <div>
            <label className="mb-1.5 block text-sm font-medium">Profissional</label>
            <select
              value={profId}
              onChange={(e) => setProfId(e.target.value)}
              className="w-full rounded-lg border border-aura-border px-3 py-2.5 text-sm"
            >
              {profissionais.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.nome}
                  {p.eh_dona ? ' (Dona)' : ''}
                </option>
              ))}
            </select>
          </div>
        )}

        <div>
          <label className="mb-1.5 block text-sm font-medium">Serviços</label>
          <p className="mb-2 text-xs text-aura-muted">Selecione um ou mais serviços para o mesmo horário.</p>
          <div className="max-h-44 space-y-1 overflow-y-auto rounded-lg border border-aura-border p-2">
            {servicos.length === 0 ? (
              <p className="px-2 py-3 text-sm text-aura-muted">Nenhum serviço ativo cadastrado.</p>
            ) : (
              servicos.map((s) => {
                const checked = servicoIds.includes(s.id)
                return (
                  <label
                    key={s.id}
                    className={[
                      'flex cursor-pointer items-center gap-3 rounded-lg px-2 py-2 text-sm transition-colors',
                      checked ? 'bg-aura-primary/10' : 'hover:bg-aura-surface',
                    ].join(' ')}
                  >
                    <input
                      type="checkbox"
                      checked={checked}
                      onChange={() => toggleServico(s.id)}
                      className="rounded border-aura-border text-aura-primary"
                    />
                    <span className="flex-1 font-medium">{s.nome}</span>
                    <span className="text-xs text-aura-muted">
                      {s.duracao_minutos} min · {formatBRL(s.preco)}
                    </span>
                  </label>
                )
              })
            )}
          </div>
          {servicosSelecionados.length > 0 && (
            <p className="mt-2 text-xs font-medium text-aura-primary">
              Total: {totalDuracao} min · {formatBRL(totalPreco)}
            </p>
          )}
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <DateInput label="Data" value={data} onChange={setData} minDate={todayISO()} />
          <TimeInput label="Horário" value={hora} onChange={setHora} withPicker />
        </div>
      </div>
    </Modal>
  )
}
