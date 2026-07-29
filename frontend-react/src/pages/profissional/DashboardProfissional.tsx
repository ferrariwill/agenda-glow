import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Calendar,
  ChevronRight,
  Clock,
  TrendingUp,
  Wallet,
  Banknote,
} from 'lucide-react'
import { ProfissionalLayout, ProfissionalGLASS } from '../../components/profissional/ProfissionalLayout'
import { useAuth } from '../../contexts/AuthContext'
import { getDb, getProfissionalComissoesResumo } from '../../utils/mockDb'
import { currentYearMonth, formatBRL, formatMonthYearBR, formatTableDateBR } from '../../utils/format'

export function DashboardProfissional() {
  const { session } = useAuth()
  const profId = session?.user.profissional_id ?? ''
  const tenantId = session?.user.tenant_id ?? ''
  const [mes, setMes] = useState(currentYearMonth())
  const [tick, setTick] = useState(0)

  const db = useMemo(() => {
    void tick
    return getDb()
  }, [tick])

  const resumo = useMemo(
    () => getProfissionalComissoesResumo(profId, tenantId, mes),
    [db, profId, tenantId, mes],
  )

  const prof = db.profissionais.find((p) => p.id === profId)
  const proximos = db.agendamentos
    .filter(
      (a) =>
        a.profissional_id === profId &&
        a.tenant_id === tenantId &&
        a.data >= new Date().toISOString().slice(0, 10) &&
        a.status !== 'CANCELADO' &&
        a.status !== 'CONCLUIDO',
    )
    .sort((a, b) => a.data.localeCompare(b.data) || a.hora_inicio.localeCompare(b.hora_inicio))
    .slice(0, 4)

  const meses = useMemo(() => {
    const out: string[] = []
    const now = new Date()
    for (let i = 0; i < 6; i++) {
      const d = new Date(now.getFullYear(), now.getMonth() - i, 1)
      out.push(`${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`)
    }
    return out
  }, [])

  return (
    <ProfissionalLayout
      title={`Olá, ${session?.user.nome?.split(' ')[0] ?? 'Profissional'}`}
      subtitle="Acompanhe suas comissões e próximos atendimentos"
    >
      <div className="mb-6 flex flex-wrap items-center justify-between gap-4">
        <p className="text-sm text-[#514440]">
          Comissões de{' '}
          <span className="font-semibold text-[#7d5141]">{formatMonthYearBR(mes)}</span>
          {prof && (
            <span className="text-[#514440]"> · {prof.comissao_percent}% por serviço</span>
          )}
        </p>
        <select
          value={mes}
          onChange={(e) => {
            setMes(e.target.value)
            setTick((t) => t + 1)
          }}
          className="rounded-lg border border-[#d6c2bd] bg-white px-3 py-2 text-sm text-[#514440] focus:border-[#7d5141] focus:outline-none"
        >
          {meses.map((m) => (
            <option key={m} value={m}>
              {formatMonthYearBR(m)}
            </option>
          ))}
        </select>
      </div>

      <section className="mb-8 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <div className={`p-6 ${ProfissionalGLASS}`}>
          <div className="mb-3 flex items-center justify-between">
            <div className="rounded-lg bg-[#996958]/20 p-2 text-[#7d5141]">
              <Wallet className="h-5 w-5" />
            </div>
            <span className="text-[10px] font-bold uppercase tracking-widest text-[#514440]">
              Total do mês
            </span>
          </div>
          <p className="font-display text-2xl font-bold text-[#7d5141] sm:text-3xl">
            {formatBRL(resumo.totalMes)}
          </p>
        </div>
        <div className={`p-6 ${ProfissionalGLASS}`}>
          <div className="mb-3 flex items-center justify-between">
            <div className="rounded-lg bg-[#efdcd1]/50 p-2 text-[#695c53]">
              <Clock className="h-5 w-5" />
            </div>
            <span className="text-[10px] font-bold uppercase tracking-widest text-[#514440]">
              A receber
            </span>
          </div>
          <p className="font-display text-2xl font-bold text-[#6d6057] sm:text-3xl">
            {formatBRL(resumo.pendente)}
          </p>
        </div>
        <div className={`p-6 ${ProfissionalGLASS}`}>
          <div className="mb-3 flex items-center justify-between">
            <div className="rounded-lg bg-[#f0f4f1] p-2 text-[#2d4a3e]">
              <Banknote className="h-5 w-5" />
            </div>
            <span className="text-[10px] font-bold uppercase tracking-widest text-[#514440]">
              Já recebido
            </span>
          </div>
          <p className="font-display text-2xl font-bold text-[#2d4a3e] sm:text-3xl">
            {formatBRL(resumo.liquidado)}
          </p>
        </div>
        <div className={`p-6 ${ProfissionalGLASS}`}>
          <div className="mb-3 flex items-center justify-between">
            <div className="rounded-lg bg-[#ffdbcf]/50 p-2 text-[#7d5141]">
              <TrendingUp className="h-5 w-5" />
            </div>
            <span className="text-[10px] font-bold uppercase tracking-widest text-[#514440]">
              Hoje
            </span>
          </div>
          <p className="font-display text-2xl font-bold text-[#7d5141] sm:text-3xl">
            {formatBRL(resumo.comissaoHoje)}
          </p>
          <p className="mt-1 text-xs text-[#514440]">
            {resumo.atendimentosHoje} atendimento(s) concluído(s)
          </p>
        </div>
      </section>

      <div className="grid gap-6 lg:grid-cols-12">
        <section className={`lg:col-span-7 ${ProfissionalGLASS} overflow-hidden`}>
          <div className="border-b border-[#d6c2bd]/20 px-6 py-4">
            <h2 className="font-display text-lg font-semibold text-[#7d5141]">
              Histórico de Comissões
            </h2>
            <p className="text-sm text-[#514440]">
              {resumo.atendimentosConcluidos} atendimentos concluídos no período
            </p>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="bg-[#f4f3f2]/50 text-[11px] font-bold uppercase tracking-widest text-[#514440]">
                <tr>
                  <th className="px-6 py-3">Data</th>
                  <th className="px-6 py-3">Descrição</th>
                  <th className="px-6 py-3 text-right">Valor</th>
                  <th className="px-6 py-3 text-center">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#d6c2bd]/10">
                {resumo.historico.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="px-6 py-10 text-center text-[#514440]">
                      Nenhuma comissão registrada neste mês.
                    </td>
                  </tr>
                ) : (
                  resumo.historico.map((item) => (
                    <tr key={item.id} className="hover:bg-[#996958]/5">
                      <td className="px-6 py-3 text-[#514440]">
                        {formatTableDateBR(item.data)}
                      </td>
                      <td className="px-6 py-3 font-medium text-[#1a1c1c]">
                        {item.descricao}
                      </td>
                      <td className="px-6 py-3 text-right font-semibold text-[#7d5141]">
                        {formatBRL(item.valor)}
                      </td>
                      <td className="px-6 py-3 text-center">
                        <span
                          className={[
                            'rounded-full px-3 py-1 text-[10px] font-bold uppercase',
                            item.status === 'LIQUIDADO'
                              ? 'bg-[#E1F5FE] text-[#01579B]'
                              : 'bg-[#efdcd1]/50 text-[#50443c]',
                          ].join(' ')}
                        >
                          {item.status === 'LIQUIDADO' ? 'Pago' : 'Pendente'}
                        </span>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </section>

        <section className={`lg:col-span-5 ${ProfissionalGLASS} p-6`}>
          <div className="mb-4 flex items-center justify-between">
            <h2 className="font-display text-lg font-semibold text-[#7d5141]">
              Próximos Atendimentos
            </h2>
            <Link
              to="/profissional/agenda"
              className="flex items-center gap-1 text-sm font-semibold text-[#7d5141] hover:underline"
            >
              Ver agenda
              <ChevronRight className="h-4 w-4" />
            </Link>
          </div>
          {proximos.length === 0 ? (
            <p className="text-sm text-[#514440]">Nenhum agendamento futuro.</p>
          ) : (
            <ul className="space-y-3">
              {proximos.map((ag) => (
                <li
                  key={ag.id}
                  className="rounded-lg border border-[#d6c2bd]/20 bg-[#faf9f8] px-4 py-3"
                >
                  <p className="font-medium text-[#1a1c1c]">{ag.cliente_nome}</p>
                  <p className="text-xs text-[#514440]">
                    {formatTableDateBR(ag.data)} · {ag.hora_inicio.slice(0, 5)}
                  </p>
                </li>
              ))}
            </ul>
          )}
          <Link
            to="/profissional/agenda"
            className="mt-6 flex w-full items-center justify-center gap-2 rounded-lg bg-[#7d5141] py-3 text-sm font-semibold text-white transition-opacity hover:opacity-90"
          >
            <Calendar className="h-4 w-4" />
            Gerenciar minha agenda
          </Link>
        </section>
      </div>
    </ProfissionalLayout>
  )
}
