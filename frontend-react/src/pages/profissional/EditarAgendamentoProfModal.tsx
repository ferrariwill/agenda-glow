import { useEffect, useMemo, useState } from 'react'
import { Alert } from '../../components/ui/Alert'
import { Button } from '../../components/ui/Button'
import { DateInput } from '../../components/ui/DateInput'
import { Input } from '../../components/ui/Input'
import { Modal } from '../../components/ui/Modal'
import { TimeInput } from '../../components/ui/TimeInput'
import type { Agendamento } from '../../types'
import { IS_MOCK } from '../../lib/config'
import {
  cancelAgendamento,
  getAgendamentoServicoIds,
  getDb,
  registrarComissaoAgendamento,
  updateAgendamento,
  updateAgendamentoStatus,
} from '../../utils/mockDb'
import { formatBRL } from '../../utils/format'

interface Props {
  open: boolean
  agendamento: Agendamento | null
  profId: string
  tenantId: string
  onClose: () => void
  onSuccess: (msg: string) => void
  allowProfissionalChange?: boolean
  onCobrar?: (ag: Agendamento) => void
}

export function EditarAgendamentoProfModal({
  open,
  agendamento,
  profId,
  tenantId,
  onClose,
  onSuccess,
  allowProfissionalChange = false,
  onCobrar,
}: Props) {
  const db = getDb()
  const profissionais = db.profissionais.filter((p) => p.tenant_id === tenantId && p.ativo)
  const [profissionalId, setProfissionalId] = useState(profId)

  useEffect(() => {
    if (open) setProfissionalId(profId)
  }, [open, profId])

  const effectiveProfId = allowProfissionalChange ? profissionalId : profId
  const servicos = db.servicos.filter(
    (s) =>
      s.tenant_id === tenantId &&
      s.ativo &&
      (!s.profissional_ids?.length || s.profissional_ids.includes(effectiveProfId)),
  )

  const [clienteNome, setClienteNome] = useState('')
  const [clienteTel, setClienteTel] = useState('')
  const [servicoIds, setServicoIds] = useState<string[]>([])
  const [data, setData] = useState('')
  const [hora, setHora] = useState('')
  const [observacoes, setObservacoes] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!open || !agendamento) return
    setClienteNome(agendamento.cliente_nome)
    setClienteTel(agendamento.cliente_telefone)
    setServicoIds(getAgendamentoServicoIds(agendamento))
    setData(agendamento.data)
    setHora(agendamento.hora_inicio)
    setObservacoes(agendamento.observacoes ?? '')
    setError('')
  }, [open, agendamento])

  const selecionados = useMemo(
    () => servicos.filter((s) => servicoIds.includes(s.id)),
    [servicos, servicoIds],
  )
  const totalPreco = selecionados.reduce((s, srv) => s + srv.preco, 0)
  const totalDuracao = selecionados.reduce((s, srv) => s + srv.duracao_minutos, 0)

  const toggleServico = (id: string) => {
    setServicoIds((prev) =>
      prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id],
    )
  }

  const salvar = async () => {
    if (!agendamento) return
    setError('')
    setLoading(true)
    try {
      if (!clienteNome.trim()) throw new Error('Informe o nome do cliente.')
      if (servicoIds.length === 0) throw new Error('Selecione ao menos um serviço.')
      await updateAgendamento(agendamento.id, {
        cliente_nome: clienteNome.trim(),
        cliente_telefone: clienteTel.replace(/\D/g, ''),
        servico_ids: servicoIds,
        data,
        hora_inicio: hora,
        observacoes: observacoes.trim() || undefined,
        ...(allowProfissionalChange ? { profissional_id: profissionalId } : {}),
      })
      onSuccess('Agendamento atualizado.')
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erro ao salvar')
    } finally {
      setLoading(false)
    }
  }

  const concluir = async () => {
    if (!agendamento) return
    await updateAgendamentoStatus(agendamento.id, 'CONCLUIDO', { role: 'PROFISSIONAL' })
    if (IS_MOCK) {
      registrarComissaoAgendamento(agendamento.id, profId, tenantId)
    }
    onSuccess('Atendimento concluído. Comissão registrada.')
    onClose()
  }

  const cancelar = async () => {
    if (!agendamento) return
    await cancelAgendamento(agendamento.id)
    onSuccess('Agendamento cancelado.')
    onClose()
  }

  if (!agendamento) return null

  const podeConcluir =
    !onCobrar &&
    (agendamento.status === 'CONFIRMADO' || agendamento.status === 'AGENDADO')
  const podeCobrar =
    !!onCobrar &&
    !agendamento.cobrado_em &&
    agendamento.status !== 'CANCELADO'
  const podeEditar = agendamento.status !== 'CANCELADO' && agendamento.status !== 'CONCLUIDO'

  return (
    <Modal open={open} onClose={onClose} title="Editar agendamento" maxWidthClass="max-w-lg">
      <div className="space-y-4">
        {error && <Alert variant="error">{error}</Alert>}

        <div className="rounded-lg bg-[#f4f3f2] px-3 py-2 text-sm text-[#514440]">
          Status: <strong className="text-[#7d5141]">{agendamento.status}</strong>
          {agendamento.cobrado_em ? (
            <span className="ml-2 text-emerald-700">· Cobrado</span>
          ) : null}
          {agendamento.minutos_invadidos ? (
            <span className="ml-2 text-amber-700">
              (+{agendamento.minutos_invadidos} min aguardando aprovação)
            </span>
          ) : null}
        </div>

        {allowProfissionalChange && podeEditar && (
          <div>
            <label className="mb-1.5 block text-sm font-medium text-[#514440]">Profissional</label>
            <select
              value={profissionalId}
              onChange={(e) => {
                setProfissionalId(e.target.value)
                setServicoIds([])
              }}
              className="w-full rounded-lg border border-[#d6c2bd] bg-white px-3 py-2.5 text-sm"
            >
              {profissionais.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.nome}
                </option>
              ))}
            </select>
          </div>
        )}

        {podeEditar ? (
          <>
            <Input
              label="Cliente"
              value={clienteNome}
              onChange={(e) => setClienteNome(e.target.value)}
            />
            <Input
              label="Telefone"
              value={clienteTel}
              onChange={(e) => setClienteTel(e.target.value)}
            />
            <div>
              <p className="mb-2 text-sm font-medium text-[#514440]">Serviços</p>
              <div className="max-h-40 space-y-2 overflow-y-auto">
                {servicos.map((s) => (
                  <label
                    key={s.id}
                    className="flex cursor-pointer items-center gap-2 rounded-lg border border-[#d6c2bd]/30 px-3 py-2 text-sm hover:bg-[#f4f3f2]"
                  >
                    <input
                      type="checkbox"
                      checked={servicoIds.includes(s.id)}
                      onChange={() => toggleServico(s.id)}
                      className="rounded text-[#7d5141]"
                    />
                    <span className="flex-1">{s.nome}</span>
                    <span className="text-[#7d5141]">{formatBRL(s.preco)}</span>
                  </label>
                ))}
              </div>
              {selecionados.length > 0 && (
                <p className="mt-2 text-xs text-[#514440]">
                  {totalDuracao} min · {formatBRL(totalPreco)}
                </p>
              )}
            </div>
            <div className="grid grid-cols-2 gap-3">
              <DateInput label="Data" value={data} onChange={setData} />
              <TimeInput label="Horário" value={hora} onChange={setHora} />
            </div>
            <div>
              <label className="mb-1.5 block text-sm font-medium">Observações</label>
              <textarea
                value={observacoes}
                onChange={(e) => setObservacoes(e.target.value)}
                rows={2}
                className="w-full rounded-lg border border-[#d6c2bd] px-3 py-2 text-sm"
              />
            </div>
          </>
        ) : (
          <p className="text-sm text-[#514440]">
            Este agendamento não pode mais ser alterado.
          </p>
        )}

        <div className="flex flex-wrap gap-2 border-t border-[#d6c2bd]/20 pt-4">
          {podeEditar && (
            <Button onClick={salvar} loading={loading}>
              Salvar alterações
            </Button>
          )}
          {podeConcluir && (
            <Button variant="secondary" onClick={concluir}>
              Concluir atendimento
            </Button>
          )}
          {podeCobrar && (
            <Button
              variant="secondary"
              onClick={() => onCobrar!(agendamento)}
            >
              Cobrar procedimento
            </Button>
          )}
          {podeEditar && (
            <Button variant="ghost" onClick={cancelar} className="text-[#ba1a1a]">
              Cancelar agendamento
            </Button>
          )}
          <Button variant="ghost" onClick={onClose}>
            Fechar
          </Button>
        </div>
      </div>
    </Modal>
  )
}
