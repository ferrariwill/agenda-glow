import { useMemo, useState } from 'react'
import { MetricCard } from './DonaLayout'
import { getRetornoProfissionalMensal } from '../../utils/mockDb'
import { currentYearMonth, formatMonthYearBR, recentYearMonths } from '../../utils/format'

interface RetornoProfissionaisPanelProps {
  tenantId: string
}

function fmtTaxa(v: number | null): string {
  return v === null ? '—' : `${v}%`
}

export function RetornoProfissionaisPanel({ tenantId }: RetornoProfissionaisPanelProps) {
  const [mesRef, setMesRef] = useState(currentYearMonth)
  const meses = useMemo(() => recentYearMonths(6), [])
  const resumo = useMemo(
    () => getRetornoProfissionalMensal(tenantId, mesRef),
    [tenantId, mesRef],
  )

  return (
    <section className="mt-10">
      <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h2 className="text-lg font-semibold text-aura-anthracite">Retorno por profissional</h2>
          <p className="mt-1 text-sm text-aura-muted">
            Taxa de retorno em {resumo.mes_referencia_label} de clientes atendidos em{' '}
            {resumo.mes_anterior_label}.
          </p>
        </div>
        <label className="flex flex-col gap-1 text-sm">
          <span className="text-aura-muted">Mês de referência</span>
          <select
            value={mesRef}
            onChange={(e) => setMesRef(e.target.value)}
            className="rounded-md border border-aura-border bg-white px-3 py-2 text-aura-anthracite"
          >
            {meses.map((ym) => (
              <option key={ym} value={ym}>
                {formatMonthYearBR(ym)}
              </option>
            ))}
          </select>
        </label>
      </div>

      <div className="mb-4 grid gap-4 sm:grid-cols-2">
        <MetricCard
          label="Média retorno — novos clientes"
          value={fmtTaxa(resumo.media_novos)}
          highlight={resumo.media_novos !== null && resumo.media_novos >= 50 ? 'success' : undefined}
        />
        <MetricCard
          label="Média retorno — clientes fidelizados (VIP)"
          value={fmtTaxa(resumo.media_fidelizados)}
          highlight={resumo.media_fidelizados !== null && resumo.media_fidelizados >= 50 ? 'success' : undefined}
        />
      </div>

      <div className="space-y-3 md:hidden">
        {resumo.por_profissional.length === 0 ? (
          <p className="rounded-lg border border-aura-border bg-white py-8 text-center text-sm text-aura-muted">
            Nenhum profissional ativo cadastrado.
          </p>
        ) : (
          resumo.por_profissional.map((row) => (
            <div
              key={row.profissional_id}
              className="rounded-lg border border-aura-border bg-white p-4 shadow-sm"
            >
              <p className="font-medium text-aura-anthracite">{row.profissional_nome}</p>
              <div className="mt-3 grid grid-cols-2 gap-3 text-sm">
                <div>
                  <p className="text-xs text-aura-muted">Novos</p>
                  <p>
                    {row.novos_retornaram}/{row.novos_mes_anterior} ·{' '}
                    <span className="font-medium">{fmtTaxa(row.taxa_retorno_novos)}</span>
                  </p>
                </div>
                <div>
                  <p className="text-xs text-aura-muted">VIP</p>
                  <p>
                    {row.fidelizados_retornaram}/{row.fidelizados_mes_anterior} ·{' '}
                    <span className="font-medium">{fmtTaxa(row.taxa_retorno_fidelizados)}</span>
                  </p>
                </div>
              </div>
            </div>
          ))
        )}
      </div>

      <div className="hidden overflow-hidden rounded-lg border border-aura-border bg-white shadow-sm md:block">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-aura-border bg-aura-surface/50 text-left text-aura-muted">
                <th className="px-4 py-3 font-medium">Profissional</th>
                <th className="px-4 py-3 font-medium">Novos ({resumo.mes_anterior_label})</th>
                <th className="px-4 py-3 font-medium">Retornaram</th>
                <th className="px-4 py-3 font-medium">Taxa novos</th>
                <th className="px-4 py-3 font-medium">VIP ({resumo.mes_anterior_label})</th>
                <th className="px-4 py-3 font-medium">Retornaram</th>
                <th className="px-4 py-3 font-medium">Taxa VIP</th>
              </tr>
            </thead>
            <tbody>
              {resumo.por_profissional.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-4 py-12 text-center text-aura-muted">
                    Nenhum profissional ativo cadastrado.
                  </td>
                </tr>
              ) : (
                resumo.por_profissional.map((row) => (
                  <tr
                    key={row.profissional_id}
                    className="border-b border-aura-border/60 hover:bg-aura-surface/30"
                  >
                    <td className="px-4 py-3.5 font-medium text-aura-anthracite">
                      {row.profissional_nome}
                    </td>
                    <td className="px-4 py-3.5 text-aura-muted">{row.novos_mes_anterior}</td>
                    <td className="px-4 py-3.5 text-aura-muted">{row.novos_retornaram}</td>
                    <td className="px-4 py-3.5 font-medium">{fmtTaxa(row.taxa_retorno_novos)}</td>
                    <td className="px-4 py-3.5 text-aura-muted">{row.fidelizados_mes_anterior}</td>
                    <td className="px-4 py-3.5 text-aura-muted">{row.fidelizados_retornaram}</td>
                    <td className="px-4 py-3.5 font-medium">
                      {fmtTaxa(row.taxa_retorno_fidelizados)}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      <p className="mt-3 text-xs text-aura-muted">
        Novos: 1ª visita com o profissional no mês anterior. VIP: clientes com 3+ visitas no salão
        atendidos pelo profissional no mês anterior. Retorno: voltaram ao mesmo profissional no mês
        selecionado.
      </p>
    </section>
  )
}
