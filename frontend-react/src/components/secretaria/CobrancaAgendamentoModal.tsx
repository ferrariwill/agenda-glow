import { useEffect, useState } from 'react'
import { Alert } from '../ui/Alert'
import { Button } from '../ui/Button'
import { Input } from '../ui/Input'
import { Modal } from '../ui/Modal'
import type { Agendamento, MetodoPagamento } from '../../types'
import {
  calcValorAgendamento,
  getAgendamentoServicosNomes,
  getDb,
  registrarCobrancaAgendamento,
} from '../../utils/mockDb'
import { formatBRL } from '../../utils/format'

export const METODOS_COBRANCA: { value: MetodoPagamento; label: string }[] = [
  { value: 'PIX', label: 'PIX' },
  { value: 'CARTAO_DEBITO', label: 'Cartão débito' },
  { value: 'CARTAO_CREDITO', label: 'Cartão crédito' },
  { value: 'DINHEIRO', label: 'Dinheiro' },
]

interface Props {
  open: boolean
  agendamento: Agendamento | null
  onClose: () => void
  onSuccess: (msg: string) => void
}

export function CobrancaAgendamentoModal({ open, agendamento, onClose, onSuccess }: Props) {
  const db = getDb()
  const [metodo, setMetodo] = useState<MetodoPagamento>('PIX')
  const [valorStr, setValorStr] = useState('')
  const [concluir, setConcluir] = useState(true)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!open || !agendamento) return
    const valor = calcValorAgendamento(agendamento, db)
    setMetodo('PIX')
    setValorStr(valor > 0 ? valor.toFixed(2).replace('.', ',') : '')
    setConcluir(true)
    setError('')
  }, [open, agendamento, db])

  if (!agendamento) return null

  const servicos = getAgendamentoServicosNomes(agendamento, db)
  const prof = db.profissionais.find((p) => p.id === agendamento.profissional_id)

  const parseValor = () => {
    const n = Number(valorStr.replace(/\./g, '').replace(',', '.'))
    if (!Number.isFinite(n) || n <= 0) throw new Error('Informe um valor válido.')
    return n
  }

  const confirmar = async () => {
    setError('')
    setLoading(true)
    try {
      const valor = parseValor()
      await registrarCobrancaAgendamento(agendamento.id, { metodo, valor, concluir })
      onSuccess(
        concluir
          ? `Cobrança registrada (${METODOS_COBRANCA.find((m) => m.value === metodo)?.label}). Atendimento concluído.`
          : `Cobrança registrada (${METODOS_COBRANCA.find((m) => m.value === metodo)?.label}).`,
      )
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erro ao registrar cobrança')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="Cobrança do procedimento" maxWidthClass="max-w-md">
      <div className="space-y-4">
        {error && <Alert variant="error">{error}</Alert>}

        <div className="rounded-lg bg-[#f4f3f2] px-4 py-3 text-sm">
          <p className="font-semibold text-[#1a1c1c]">{agendamento.cliente_nome}</p>
          <p className="text-[#514440]">{servicos}</p>
          <p className="mt-1 text-xs text-[#514440]">
            {agendamento.data} às {agendamento.hora_inicio.slice(0, 5)} · {prof?.nome ?? 'Profissional'}
          </p>
        </div>

        <Input
          label="Valor (R$)"
          value={valorStr}
          onChange={(e) => setValorStr(e.target.value)}
          placeholder="0,00"
        />
        <p className="-mt-2 text-xs text-[#514440]">
          Sugerido: {formatBRL(calcValorAgendamento(agendamento, db))}
        </p>

        <div>
          <p className="mb-2 text-sm font-medium text-[#514440]">Método de pagamento</p>
          <div className="grid grid-cols-2 gap-2">
            {METODOS_COBRANCA.map((m) => (
              <button
                key={m.value}
                type="button"
                onClick={() => setMetodo(m.value)}
                className={[
                  'rounded-lg border px-3 py-2.5 text-sm font-medium transition-colors',
                  metodo === m.value
                    ? 'border-[#7d5141] bg-[#efdcd1]/50 text-[#7d5141]'
                    : 'border-[#d6c2bd]/40 text-[#514440] hover:bg-[#f4f3f2]',
                ].join(' ')}
              >
                {m.label}
              </button>
            ))}
          </div>
        </div>

        <label className="flex cursor-pointer items-center gap-2 text-sm text-[#514440]">
          <input
            type="checkbox"
            checked={concluir}
            onChange={(e) => setConcluir(e.target.checked)}
            className="rounded text-[#7d5141]"
          />
          Concluir atendimento após a cobrança
        </label>

        <div className="flex gap-2 border-t border-[#d6c2bd]/20 pt-4">
          <Button onClick={confirmar} loading={loading} className="flex-1">
            Confirmar cobrança
          </Button>
          <Button variant="ghost" onClick={onClose}>
            Cancelar
          </Button>
        </div>
      </div>
    </Modal>
  )
}
