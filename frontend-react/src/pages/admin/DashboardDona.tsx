import { Link } from 'react-router-dom'
import { useEffect, useMemo, useState } from 'react'
import {
  CalendarCheck,
  CheckCircle2,
  Download,
  Info,
  Send,
  Sparkles,
  Star,
  TrendingUp,
  Wallet,
} from 'lucide-react'
import { DonaLayout, DonaFooter } from '../../components/dona/DonaLayout'
import {
  ResponsiveEntityList,
  type EntityColumn,
} from '../../components/ui/ResponsiveEntityList'
import { ToastFeedback } from '../../components/ui/ToastFeedback'
import { useAuth } from '../../contexts/AuthContext'
import { enviarConviteReativacao } from '../../services/whatsappService'
import {
  getDonaFechamentoDia,
  getDb,
  getFechamentoNota,
  saveFechamentoNota,
  type ClienteReativacaoRow,
} from '../../utils/mockDb'
import {
  formatBRL,
  formatDateShortBR,
  formatWeekdayDateLongBR,
  initials,
  todayISO,
} from '../../utils/format'

const GLASS =
  'rounded-xl border border-[#e5d3c8]/50 bg-white/70 shadow-[0px_4px_20px_rgba(183,132,114,0.08)] backdrop-blur-[10px]'

function diasBadgeClass(dias: number) {
  if (dias >= 40) return 'bg-[#ffdad6] text-[#93000a]'
  return 'bg-[#efdcd1] text-[#6d6057]'
}

function avatarRingClass(i: number) {
  const rings = [
    'bg-[#efdcd1] text-[#695c53]',
    'bg-[#ebe0dd] text-[#615b58]',
    'bg-[#efdcd1] text-[#6d6057]',
  ]
  return rings[i % rings.length]
}

export function DashboardDona() {
  const { session } = useAuth()
  const tenantId = session?.user.tenant_id ?? ''
  const hoje = todayISO()
  const tenant = getDb().tenants.find((t) => t.id === tenantId)

  const [search, setSearch] = useState('')
  const [fechamento, setFechamento] = useState(() => getDonaFechamentoDia(tenantId, hoje))
  const [nota, setNota] = useState(() => getFechamentoNota(tenantId, hoje))
  const [toast, setToast] = useState<string | null>(null)
  const [sendingId, setSendingId] = useState<string | null>(null)

  useEffect(() => {
    setFechamento(getDonaFechamentoDia(tenantId, hoje))
    setNota(getFechamentoNota(tenantId, hoje))
  }, [tenantId, hoje])

  const reativacao = useMemo(() => {
    const q = search.trim().toLowerCase()
    let list = fechamento.clientesReativacao
    if (q) {
      list = list.filter(
        (c) =>
          c.nome.toLowerCase().includes(q) ||
          c.ultimoServico.toLowerCase().includes(q),
      )
    }
    return list.slice(0, 3)
  }, [fechamento.clientesReativacao, search])

  const circ = 2 * Math.PI * 58
  const metaOffset = circ * (1 - fechamento.metaPct / 100)

  const variacaoLabel =
    fechamento.receitaVariacaoPct === null
      ? null
      : `${fechamento.receitaVariacaoPct >= 0 ? '+' : ''}${fechamento.receitaVariacaoPct}% vs ontem`

  const ticketVariacao =
    fechamento.ticketMedioVariacaoPct !== null && fechamento.ticketMedioVariacaoPct >= 0
      ? `Superou em ${fechamento.ticketMedioVariacaoPct}% a média semanal.`
      : fechamento.ticketMedioVariacaoPct !== null
        ? `${Math.abs(fechamento.ticketMedioVariacaoPct)}% abaixo da média semanal.`
        : 'Sem histórico suficiente para comparar.'

  const handleExport = () => {
    setToast('Resumo exportado com sucesso para o e-mail cadastrado.')
  }

  const handleEncerrar = () => {
    setToast('Expediente encerrado. Resumo do dia salvo.')
  }

  const handleSalvarNota = () => {
    saveFechamentoNota(tenantId, hoje, nota)
    setToast('Nota de fechamento salva.')
  }

  const handleReativacao = async (row: ClienteReativacaoRow) => {
    setSendingId(row.id)
    const slug = tenant?.slug ?? 'salao'
    await enviarConviteReativacao({
      telefone: row.telefone,
      clienteNome: row.nome,
      salaoNome: tenant?.nome ?? 'Seu salão',
      linkAgendamento: `${window.location.origin}/${slug}/agendar`,
    })
    setSendingId(null)
    setToast(`Convite enviado para ${row.nome} via WhatsApp.`)
  }

  const dateHeader = formatWeekdayDateLongBR(hoje)

  const renderReativacaoAction = (row: ClienteReativacaoRow) => (
    <button
      type="button"
      disabled={sendingId === row.id}
      onClick={() => void handleReativacao(row)}
      className="inline-flex h-11 min-w-11 items-center justify-center gap-2 rounded-lg px-3 text-[#7d5141] transition-colors hover:bg-[#7d5141]/10 disabled:opacity-50"
      aria-label={`Convidar ${row.nome} via WhatsApp`}
    >
      <Send className="h-4 w-4 shrink-0" aria-hidden />
      <span className="hidden text-xs font-semibold sm:inline">Convidar</span>
    </button>
  )

  const reativacaoColumns: EntityColumn<ClienteReativacaoRow>[] = [
    {
      header: 'Cliente',
      cell: (row) => {
        const i = reativacao.indexOf(row)
        return (
          <div className="flex items-center gap-3">
            <div
              className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-xs font-bold ${avatarRingClass(i)}`}
            >
              {initials(row.nome)}
            </div>
            <div className="min-w-0">
              <p className="text-sm font-semibold">{row.nome}</p>
              <p className="text-xs text-[#83746f]">{row.ultimoServico}</p>
            </div>
          </div>
        )
      },
    },
    {
      header: 'Último serviço',
      cell: (row) => (
        <span className="whitespace-nowrap text-sm">
          {formatDateShortBR(row.ultimaVisita)}
        </span>
      ),
      priority: 'secondary',
    },
    {
      header: 'Dias ausente',
      cell: (row) => (
        <span
          className={`rounded-full px-2 py-1 text-[11px] font-bold ${diasBadgeClass(row.diasAusente)}`}
        >
          {row.diasAusente} dias
        </span>
      ),
    },
    {
      header: 'Ação',
      cell: renderReativacaoAction,
      align: 'right',
    },
  ]

  const renderReativacaoCard = (row: ClienteReativacaoRow, index: number) => (
    <div className="rounded-xl border border-[#efdcd1]/30 bg-[#faf9f8]/50 p-4">
      <div className="flex items-start gap-3">
        <div
          className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-full text-xs font-bold ${avatarRingClass(index)}`}
        >
          {initials(row.nome)}
        </div>
        <div className="min-w-0 flex-1">
          <p className="text-sm font-semibold">{row.nome}</p>
          <p className="text-xs text-[#83746f]">{row.ultimoServico}</p>
        </div>
        {renderReativacaoAction(row)}
      </div>
      {/* ≤2 decisões: dias ausente + última visita */}
      <div className="mt-3 flex items-center justify-between gap-3">
        <span
          className={`rounded-full px-2 py-1 text-[11px] font-bold ${diasBadgeClass(row.diasAusente)}`}
        >
          {row.diasAusente} dias
        </span>
        <p className="shrink-0 whitespace-nowrap text-xs text-[#514440]">
          {formatDateShortBR(row.ultimaVisita)}
        </p>
      </div>
    </div>
  )

  return (
    <DonaLayout
      searchPlaceholder="Pesquisar relatório ou cliente…"
      searchValue={search}
      onSearchChange={setSearch}
      headerAction={
        <p className="hidden text-sm font-semibold text-[#7d5141] lg:block">{dateHeader}</p>
      }
    >
      <div className="mb-8 flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <span className="text-xs font-bold uppercase tracking-widest text-[#7d5141]">
            Relatório gerencial
          </span>
          <h1 className="font-display mt-1 text-3xl font-bold text-[#7d5141] sm:text-4xl lg:text-5xl">
            Fechamento do dia
          </h1>
          <p className="mt-1 text-sm text-[#615b58]">
            Visão consolidada do desempenho de hoje, {formatDateShortBR(hoje).replace(/^\d+\//, '')}.
          </p>
        </div>
        <div className="flex flex-wrap gap-3">
          <button
            type="button"
            onClick={handleExport}
            className="flex items-center gap-2 rounded-lg border border-[#7d5141] px-6 py-2.5 text-sm font-semibold text-[#7d5141] transition-colors hover:bg-[#7d5141]/5"
          >
            <Download className="h-4 w-4" />
            Exportar PDF
          </button>
          <button
            type="button"
            onClick={handleEncerrar}
            className="flex items-center gap-2 rounded-lg bg-[#7d5141] px-6 py-2.5 text-sm font-semibold text-white shadow-lg transition-all hover:scale-[1.02] active:scale-95"
          >
            <CheckCircle2 className="h-4 w-4 fill-current" />
            Encerrar expediente
          </button>
        </div>
      </div>

      {/* KPI Bento */}
      <div className="mb-8 grid grid-cols-12 gap-6">
        <div className={`relative col-span-12 overflow-hidden p-8 lg:col-span-4 ${GLASS} group`}>
          <div className="absolute -right-4 -top-4 h-32 w-32 rounded-full bg-[#ffdbcf] opacity-10 blur-3xl transition-transform duration-700 group-hover:scale-150" />
          <div className="relative mb-6 flex items-start justify-between">
            <div className="rounded-lg bg-[#ffdbcf] p-2">
              <Wallet className="h-5 w-5 text-[#7d5141]" />
            </div>
            {variacaoLabel && (
              <span className="rounded-full bg-[#ebe0dd] px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide text-[#4b4543]">
                {variacaoLabel}
              </span>
            )}
          </div>
          <p className="text-xs font-bold uppercase tracking-tight text-[#615b58]">
            Receita de agendamentos
          </p>
          <p className="font-display mt-1 text-4xl font-bold text-[#7d5141] lg:text-5xl">
            {formatBRL(fechamento.receitaHoje)}
          </p>
          <p className="mt-2 flex items-center gap-1 text-sm text-[#83746f]">
            <Info className="h-4 w-4" />
            Inclui serviços concluídos até agora
          </p>
        </div>

        <div className={`col-span-12 p-8 lg:col-span-5 ${GLASS}`}>
          <p className="mb-6 text-xs font-bold uppercase tracking-tight text-[#615b58]">
            Volume de atendimentos
          </p>
          <div className="flex flex-col items-center gap-8 sm:flex-row sm:items-center">
            <div className="relative h-32 w-32 shrink-0">
              <svg className="h-full w-full -rotate-90" viewBox="0 0 128 128">
                <circle
                  cx="64"
                  cy="64"
                  r="58"
                  fill="transparent"
                  stroke="#e9e8e7"
                  strokeWidth="8"
                />
                <circle
                  cx="64"
                  cy="64"
                  r="58"
                  fill="transparent"
                  stroke="#7d5141"
                  strokeWidth="8"
                  strokeDasharray={circ}
                  strokeDashoffset={metaOffset}
                  strokeLinecap="round"
                />
              </svg>
              <div className="absolute inset-0 flex flex-col items-center justify-center">
                <span className="font-display text-2xl font-semibold text-[#7d5141]">
                  {fechamento.metaPct}%
                </span>
                <span className="text-[10px] font-bold uppercase tracking-widest text-[#83746f]">
                  Meta
                </span>
              </div>
            </div>
            <div className="w-full flex-1 space-y-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="h-3 w-3 rounded-full bg-[#7d5141]" />
                  <span className="text-sm">Concluídos</span>
                </div>
                <span className="font-bold">{fechamento.concluidos}</span>
              </div>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="h-3 w-3 rounded-full bg-[#e9e8e7]" />
                  <span className="text-sm">Agendados total</span>
                </div>
                <span className="font-bold">{fechamento.agendadosTotal}</span>
              </div>
              <div className="h-px w-full bg-[#d6c2bd]/30" />
              <div className="flex items-center justify-between text-[#83746f]">
                <span className="text-sm italic">
                  {fechamento.faltamAtendimentos > 0
                    ? `Faltam ${fechamento.faltamAtendimentos} atendimento${fechamento.faltamAtendimentos > 1 ? 's' : ''}`
                    : 'Todos os agendamentos foram concluídos'}
                </span>
                <CalendarCheck className="h-4 w-4" />
              </div>
            </div>
          </div>
        </div>

        <div className="col-span-12 flex flex-col gap-4 lg:col-span-3">
          <div className={`flex flex-1 flex-col justify-center border-l-4 border-[#ba1a1a] p-6 ${GLASS}`}>
            <p className="mb-1 text-xs font-bold uppercase text-[#ba1a1a]">No-shows (faltas)</p>
            <div className="flex flex-wrap items-baseline gap-2">
              <span className="font-display text-3xl font-bold text-[#93000a]">
                {String(fechamento.noShows).padStart(2, '0')}
              </span>
              <span className="text-sm text-[#83746f]">
                -{formatBRL(fechamento.noShowsValorProjetado)} projetado
              </span>
            </div>
          </div>
          <div className={`flex flex-1 flex-col justify-center border-l-4 border-[#695c53] p-6 ${GLASS}`}>
            <p className="mb-1 text-xs font-bold uppercase text-[#695c53]">Cancelamentos</p>
            <div className="flex flex-wrap items-baseline gap-2">
              <span className="font-display text-3xl font-bold text-[#6d6057]">
                {String(fechamento.cancelamentos).padStart(2, '0')}
              </span>
              <span className="text-sm text-[#83746f]">Justificados por app</span>
            </div>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-12 gap-6">
        {/* Reativação table */}
        <div className={`col-span-12 overflow-hidden lg:col-span-8 ${GLASS}`}>
          <div className="flex flex-wrap items-center justify-between gap-2 bg-[#f4f3f2]/50 p-6">
            <div className="flex items-center gap-2">
              <TrendingUp className="h-5 w-5 text-[#7d5141]" />
              <h2 className="font-semibold text-[#7d5141]">
                Clientes para reativação (inativos +30 dias)
              </h2>
            </div>
            <span className="text-xs italic text-[#83746f]">Baseado no último agendamento</span>
          </div>
          <ResponsiveEntityList
            items={reativacao}
            getKey={(row) => row.id}
            renderCard={renderReativacaoCard}
            columns={reativacaoColumns}
            emptyTitle={
              search
                ? 'Nenhum cliente inativo encontrado para esta busca.'
                : 'Nenhum cliente inativo encontrado.'
            }
            emptyAction={
              <Link
                to="/admin/clientes"
                className="inline-flex items-center gap-2 rounded-full bg-[#7d5141] px-5 py-2.5 text-xs font-bold uppercase tracking-wider text-white"
              >
                Ver clientes
              </Link>
            }
            tableFrom="md"
            cardListClassName="space-y-3 p-4"
          />
          {fechamento.totalInativos > 3 && (
            <Link
              to="/admin/clientes"
              className="block w-full border-t border-[#d6c2bd]/30 p-4 text-center text-xs font-bold text-[#7d5141] transition-colors hover:bg-[#ffdbcf]/10"
            >
              Ver todos os {fechamento.totalInativos} clientes inativos
            </Link>
          )}
        </div>

        {/* Sidebar highlights */}
        <div className="col-span-12 flex flex-col gap-6 lg:col-span-4">
          <div className={`flex-1 p-6 ${GLASS}`}>
            <h3 className="mb-4 flex items-center gap-2 font-semibold text-[#7d5141]">
              <Sparkles className="h-4 w-4" />
              Destaques de performance
            </h3>
            <ul className="space-y-4">
              <li className="flex gap-2">
                <TrendingUp className="mt-0.5 h-4 w-4 shrink-0 text-[#7d5141]" />
                <p className="text-sm text-[#514440]">
                  <strong className="text-[#1a1c1c]">Ticket médio:</strong>{' '}
                  {formatBRL(fechamento.ticketMedioHoje)}. {ticketVariacao}
                </p>
              </li>
              <li className="flex gap-2">
                <Star className="mt-0.5 h-4 w-4 shrink-0 text-[#7d5141]" />
                <p className="text-sm text-[#514440]">
                  <strong className="text-[#1a1c1c]">Satisfação:</strong>{' '}
                  {fechamento.avaliacoesHoje > 0
                    ? `${String(fechamento.avaliacoesHoje).padStart(2, '0')} avaliações recebidas hoje, todas 5 estrelas.`
                    : 'Nenhuma avaliação registrada hoje ainda.'}
                </p>
              </li>
              <li className="flex gap-2">
                <CalendarCheck className="mt-0.5 h-4 w-4 shrink-0 text-[#7d5141]" />
                <p className="text-sm text-[#514440]">
                  <strong className="text-[#1a1c1c]">Amanhã:</strong>{' '}
                  {fechamento.ocupacaoAmanhaPct}% de ocupação.
                  {fechamento.ocupacaoAmanhaPct >= 80
                    ? ' Sugerimos abrir horários extras.'
                    : ' Há vagas disponíveis para novos agendamentos.'}
                </p>
              </li>
            </ul>
          </div>

          <div className="rounded-xl border-2 border-dashed border-[#996958]/30 bg-[#ffdbcf]/5 p-4">
            <textarea
              value={nota}
              onChange={(e) => setNota(e.target.value)}
              placeholder="Adicionar nota de fechamento do dia…"
              className="h-24 w-full resize-none border-none bg-transparent text-sm italic placeholder:text-[#83746f]/50 focus:ring-0"
            />
            <div className="mt-2 flex items-center justify-between">
              <span className="text-[10px] font-bold uppercase tracking-widest text-[#83746f]">
                Visível apenas para admins
              </span>
              <button
                type="button"
                onClick={handleSalvarNota}
                className="rounded px-2 py-1 text-xs font-bold text-[#7d5141] hover:bg-[#7d5141]/10"
              >
                Salvar nota
              </button>
            </div>
          </div>
        </div>
      </div>

      <DonaFooter />

      <ToastFeedback message={toast} onDismiss={() => setToast(null)} />
    </DonaLayout>
  )
}
