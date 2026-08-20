import { useMemo, useState } from 'react'
import {
  AlertTriangle,
  ArrowLeft,
  BadgeCheck,
  CalendarPlus,
  Cake,
  Heart,
  ImagePlus,
  Mail,
  MoreVertical,
  NotebookPen,
  Pencil,
  Phone,
  Stethoscope,
  User,
} from 'lucide-react'
import { Link, Navigate, useNavigate, useParams } from 'react-router-dom'
import { DonaLayout, DonaFooter } from '../../components/dona/DonaLayout'
import {
  ResponsiveEntityList,
  type EntityColumn,
} from '../../components/ui/ResponsiveEntityList'
import { ToastFeedback } from '../../components/ui/ToastFeedback'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Input'
import { Modal } from '../../components/ui/Modal'
import { useAuth } from '../../contexts/AuthContext'
import {
  addClienteNota,
  getClienteById,
  getClienteGaleria,
  getClienteHistorico,
  getClienteNotas,
  getClienteProfileMetrics,
  updateCliente,
} from '../../utils/mockDb'
import {
  formatBRL,
  formatBirthDateBR,
  formatDateShortBR,
  formatMonthYearBR,
  formatPhoneBR,
  formatTimeBR,
  initials,
  maskPhoneBRInput,
  phoneDigitsToMaskInput,
  toWhatsAppDigits,
} from '../../utils/format'

type TabId = 'history' | 'gallery' | 'notes'
type HistoricoItem = ReturnType<typeof getClienteHistorico>[number]

function historicoStatusLabel(status: string) {
  switch (status) {
    case 'CONCLUIDO':
      return 'Concluído'
    case 'CANCELADO':
      return 'Cancelado'
    case 'CONFIRMADO':
      return 'Confirmado'
    case 'EM_APROVACAO':
      return 'Em aprovação'
    default:
      return status
  }
}

function historicoStatusClass(status: string) {
  switch (status) {
    case 'CONCLUIDO':
    case 'CONFIRMADO':
      return 'bg-[#e7f3ef] text-[#1e4620]'
    case 'CANCELADO':
      return 'bg-[#e9e8e7] text-[#615b58]'
    case 'EM_APROVACAO':
      return 'bg-[#fff8e1] text-[#f57f17]'
    default:
      return 'bg-[#f4f3f2] text-[#514440]'
  }
}

function isVip(visitas: number, ltv: number) {
  return visitas > 2 && ltv >= 300
}

export function ClienteDetalheDona() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { session } = useAuth()
  const tenantId = session?.user.tenant_id ?? ''
  const [tab, setTab] = useState<TabId>('history')
  const [tick, setTick] = useState(0)
  const [success, setSuccess] = useState('')
  const [error, setError] = useState('')

  const [editOpen, setEditOpen] = useState(false)
  const [nome, setNome] = useState('')
  const [telefone, setTelefone] = useState('')
  const [email, setEmail] = useState('')

  const [notaOpen, setNotaOpen] = useState(false)
  const [notaTexto, setNotaTexto] = useState('')

  const cliente = useMemo(
    () => (id ? getClienteById(tenantId, id) : undefined),
    [tenantId, id, tick],
  )

  const metrics = useMemo(
    () => (cliente ? getClienteProfileMetrics(tenantId, cliente.telefone) : null),
    [tenantId, cliente, tick],
  )

  const historico = useMemo(
    () => (cliente ? getClienteHistorico(tenantId, cliente.telefone) : []),
    [tenantId, cliente, tick],
  )

  const notas = useMemo(
    () => (cliente ? getClienteNotas(cliente.id) : []),
    [cliente, tick],
  )

  const galeria = useMemo(
    () => (cliente ? getClienteGaleria(cliente.id) : []),
    [cliente, tick],
  )

  if (!id || !cliente || !metrics) {
    return <Navigate to="/admin/clientes" replace />
  }

  const vip = isVip(metrics.visitas, metrics.ltv)

  const openEdit = () => {
    setNome(cliente.nome)
    setTelefone(phoneDigitsToMaskInput(cliente.telefone))
    setEmail(cliente.email ?? '')
    setError('')
    setEditOpen(true)
  }

  const saveEdit = () => {
    setError('')
    if (!nome.trim() || !telefone.trim()) {
      setError('Nome e telefone são obrigatórios')
      return
    }
    const telefoneDigits = toWhatsAppDigits(telefone)
    if (telefoneDigits.length < 12) {
      setError('Informe um WhatsApp completo com DDD (ex.: (11) 99999-9999)')
      return
    }
    try {
      updateCliente(cliente.id, { nome, telefone: telefoneDigits, email })
      setSuccess('Perfil atualizado!')
      setEditOpen(false)
      setTick((t) => t + 1)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Erro ao salvar')
    }
  }

  const saveNota = () => {
    setError('')
    try {
      addClienteNota(cliente.id, session?.user.nome ?? 'Equipe', notaTexto)
      setNotaTexto('')
      setNotaOpen(false)
      setSuccess('Nota registrada!')
      setTick((t) => t + 1)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Erro ao salvar nota')
    }
  }

  const openAgendar = () => {
    navigate(`/admin/clientes/${cliente.id}/agendar`)
  }

  const renderHistoricoAction = () => (
    <button
      type="button"
      className="inline-flex h-11 w-11 items-center justify-center rounded-full text-aura-muted transition-colors hover:bg-[#e9e8e7] hover:text-[#7d5141]"
      aria-label="Mais opções do atendimento"
    >
      <MoreVertical className="h-4 w-4" />
    </button>
  )

  const historicoColumns: EntityColumn<HistoricoItem>[] = [
    {
      header: 'Data',
      cell: (h) => (
        <div>
          <p className="whitespace-nowrap text-sm font-semibold">{formatDateShortBR(h.data)}</p>
          <p className="whitespace-nowrap text-xs text-aura-muted">{formatTimeBR(h.hora_inicio)}</p>
        </div>
      ),
    },
    {
      header: 'Serviço',
      cell: (h) => <span className="text-sm">{h.servico_nome}</span>,
    },
    {
      header: 'Profissional',
      cell: (h) => (
        <div className="flex items-center gap-2">
          <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-[#f1dfd4] text-[10px] font-bold text-[#7d5141]">
            {initials(h.profissional_nome)}
          </div>
          <span className="text-sm">{h.profissional_nome}</span>
        </div>
      ),
      priority: 'secondary',
    },
    {
      header: 'Valor',
      cell: (h) => (
        <span className="whitespace-nowrap text-sm font-semibold">{formatBRL(h.valor)}</span>
      ),
      align: 'right',
    },
    {
      header: 'Status',
      cell: (h) => (
        <span
          className={[
            'inline-flex rounded-full px-2 py-0.5 text-xs font-medium',
            historicoStatusClass(h.status),
          ].join(' ')}
        >
          {historicoStatusLabel(h.status)}
        </span>
      ),
    },
    {
      header: 'Ações',
      cell: () => renderHistoricoAction(),
      align: 'right',
    },
  ]

  const renderHistoricoCard = (h: HistoricoItem) => (
    <div className="rounded-xl border border-[#efdcd1]/40 bg-[#faf9f8] p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="font-semibold text-[#1a1c1c]">{h.servico_nome}</p>
          <p className="mt-0.5 whitespace-nowrap text-xs text-aura-muted">
            {formatDateShortBR(h.data)} · {formatTimeBR(h.hora_inicio)}
          </p>
        </div>
        {renderHistoricoAction()}
      </div>
      {/* ≤2 decisões: status + valor */}
      <div className="mt-3 flex items-center justify-between gap-3">
        <span
          className={[
            'inline-flex rounded-full px-2 py-0.5 text-xs font-medium',
            historicoStatusClass(h.status),
          ].join(' ')}
        >
          {historicoStatusLabel(h.status)}
        </span>
        <p className="shrink-0 whitespace-nowrap font-semibold text-[#7d5141]">
          {formatBRL(h.valor)}
        </p>
      </div>
      <p className="mt-2 text-xs text-aura-muted">{h.profissional_nome}</p>
    </div>
  )

  return (
    <DonaLayout searchPlaceholder="Buscar clientes ou agendamentos…">
      <Link
        to="/admin/clientes"
        className="mb-4 inline-flex items-center gap-2 text-sm font-medium text-aura-muted transition-colors hover:text-[#7d5141]"
      >
        <ArrowLeft className="h-4 w-4" />
        Voltar para clientes
      </Link>

      <ToastFeedback message={success || null} onDismiss={() => setSuccess('')} />

      {/* Profile header */}
      <section className="mb-6 rounded-xl bg-white p-5 shadow-[0px_4px_20px_rgba(183,132,114,0.08)] sm:p-6">
        <div className="flex flex-col gap-6 md:flex-row md:items-center md:justify-between">
          <div className="flex flex-col items-center gap-4 sm:flex-row sm:items-center">
            <div className="relative shrink-0">
              {cliente.foto_url ? (
                <img
                  src={cliente.foto_url}
                  alt={cliente.nome}
                  className="h-24 w-24 rounded-full object-cover ring-4 ring-[#efdcd1] shadow-lg"
                />
              ) : (
                <div className="flex h-24 w-24 items-center justify-center rounded-full bg-[#f1dfd4] text-2xl font-bold text-[#7d5141] ring-4 ring-[#efdcd1] shadow-lg">
                  {initials(cliente.nome)}
                </div>
              )}
              {vip && (
                <div className="absolute -bottom-1 -right-1 flex h-8 w-8 items-center justify-center rounded-full border-2 border-white bg-[#7d5141] text-white">
                  <BadgeCheck className="h-4 w-4" />
                </div>
              )}
            </div>
            <div className="text-center sm:text-left">
              <div className="flex flex-wrap items-center justify-center gap-2 sm:justify-start">
                <h1 className="font-display text-2xl font-bold text-[#7d5141] sm:text-3xl">
                  {cliente.nome}
                </h1>
                {vip && (
                  <span className="rounded-full bg-[#e7f3ef] px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-[#1e4620]">
                    VIP Client
                  </span>
                )}
                {!cliente.ativo && (
                  <span className="rounded-full bg-[#e9e8e7] px-2 py-0.5 text-[10px] font-bold uppercase text-[#514440]">
                    Inativo
                  </span>
                )}
              </div>
              <p className="mt-1 text-sm text-aura-muted">
                Cliente desde {formatMonthYearBR(cliente.criado_em.slice(0, 7))} ·{' '}
                {metrics.visitas} {metrics.visitas === 1 ? 'visita' : 'visitas'} totais
              </p>
            </div>
          </div>
          <div className="flex flex-col gap-2 sm:flex-row">
            <Button variant="secondary" className="border-[#7d5141] text-[#7d5141]" onClick={openEdit}>
              <Pencil className="h-4 w-4" />
              Editar perfil
            </Button>
            <Button
              className="bg-[#7d5141] hover:bg-[#996958]"
              onClick={openAgendar}
              disabled={!cliente.ativo}
            >
              <CalendarPlus className="h-4 w-4" />
              Agendar horário
            </Button>
          </div>
        </div>
      </section>

      {/* Grid */}
      <section className="grid grid-cols-1 gap-6 pb-32 lg:grid-cols-12 lg:gap-6">
        {/* Left column */}
        <div className="space-y-6 lg:col-span-4">
          <div className="rounded-xl border border-transparent bg-white p-5 shadow-[0px_4px_20px_rgba(183,132,114,0.08)] transition-colors hover:border-[#f1dfd4]">
            <h3 className="mb-4 flex items-center gap-2 font-display text-lg font-semibold text-[#7d5141]">
              <User className="h-5 w-5" />
              Informações de contato
            </h3>
            <div className="space-y-4">
              {cliente.email && (
                <div className="flex items-start gap-3">
                  <Mail className="mt-0.5 h-5 w-5 shrink-0 text-aura-muted" />
                  <div>
                    <p className="text-[10px] font-bold uppercase tracking-wider text-aura-muted">
                      E-mail
                    </p>
                    <p className="text-sm text-aura-anthracite">{cliente.email}</p>
                  </div>
                </div>
              )}
              <div className="flex items-start gap-3">
                <Phone className="mt-0.5 h-5 w-5 shrink-0 text-aura-muted" />
                <div>
                  <p className="text-[10px] font-bold uppercase tracking-wider text-aura-muted">
                    Telefone
                  </p>
                  <p className="text-sm text-aura-anthracite">{formatPhoneBR(cliente.telefone)}</p>
                </div>
              </div>
              {cliente.data_nascimento && (
                <div className="flex items-start gap-3">
                  <Cake className="mt-0.5 h-5 w-5 shrink-0 text-aura-muted" />
                  <div>
                    <p className="text-[10px] font-bold uppercase tracking-wider text-aura-muted">
                      Data de nascimento
                    </p>
                    <p className="text-sm text-aura-anthracite">
                      {formatBirthDateBR(cliente.data_nascimento)}
                    </p>
                  </div>
                </div>
              )}
            </div>
          </div>

          {(cliente.alergias?.length || cliente.observacoes_medicas) && (
            <div className="rounded-xl bg-white p-5 shadow-[0px_4px_20px_rgba(183,132,114,0.08)]">
              <h3 className="mb-4 flex items-center gap-2 font-display text-lg font-semibold text-[#7d5141]">
                <Stethoscope className="h-5 w-5" />
                Ficha médica & alergias
              </h3>
              <div className="space-y-3">
                {cliente.alergias?.map((a) => (
                  <div
                    key={a}
                    className="flex items-center gap-2 rounded border-l-4 border-red-600 bg-red-50/80 p-3"
                  >
                    <AlertTriangle className="h-4 w-4 shrink-0 text-red-600" />
                    <span className="text-sm font-semibold text-red-800">Alergia a {a}</span>
                  </div>
                ))}
                {cliente.observacoes_medicas && (
                  <div className="rounded-lg bg-[#f4f3f2] p-3">
                    <p className="mb-1 text-[10px] font-bold uppercase tracking-wider text-aura-muted">
                      Observações
                    </p>
                    <p className="text-sm text-aura-anthracite">{cliente.observacoes_medicas}</p>
                  </div>
                )}
              </div>
            </div>
          )}

          {cliente.preferencias && cliente.preferencias.length > 0 && (
            <div className="rounded-xl bg-white p-5 shadow-[0px_4px_20px_rgba(183,132,114,0.08)]">
              <h3 className="mb-4 flex items-center gap-2 font-display text-lg font-semibold text-[#7d5141]">
                <Heart className="h-5 w-5" />
                Preferências do cliente
              </h3>
              <div className="flex flex-wrap gap-2">
                {cliente.preferencias.map((pref) => (
                  <span
                    key={pref}
                    className="inline-flex items-center gap-1 rounded-full bg-[#efdcd1] px-3 py-1 text-sm text-[#6d6057]"
                  >
                    {pref}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Right column — tabs */}
        <div className="lg:col-span-8">
          <div className="flex min-h-[480px] flex-col rounded-xl bg-white shadow-[0px_4px_20px_rgba(183,132,114,0.08)]">
            <div className="flex flex-wrap gap-1 border-b border-[#efdcd1] px-2 sm:px-4">
              {(
                [
                  ['history', 'Histórico de atendimentos'],
                  ['gallery', 'Galeria de resultados'],
                  ['notes', 'Notas internas'],
                ] as const
              ).map(([key, label]) => (
                <button
                  key={key}
                  type="button"
                  onClick={() => setTab(key)}
                  className={[
                    'min-h-11 whitespace-nowrap border-b-2 px-3 py-3 text-sm font-semibold transition-colors sm:px-4',
                    tab === key
                      ? 'border-[#7d5141] text-[#7d5141]'
                      : 'border-transparent text-aura-muted hover:text-[#7d5141]',
                  ].join(' ')}
                >
                  {label}
                </button>
              ))}
            </div>

            {tab === 'history' && (
              <div className="p-2 sm:p-4">
                <ResponsiveEntityList
                  items={historico}
                  getKey={(h) => h.id}
                  renderCard={renderHistoricoCard}
                  columns={historicoColumns}
                  emptyTitle="Nenhum atendimento registrado."
                  emptyAction={
                    cliente.ativo ? (
                      <Button
                        className="bg-[#7d5141] hover:bg-[#996958]"
                        onClick={openAgendar}
                      >
                        <CalendarPlus className="h-4 w-4" />
                        Agendar horário
                      </Button>
                    ) : undefined
                  }
                  tableFrom="md"
                  cardListClassName="space-y-3 p-2"
                />
              </div>
            )}

            {tab === 'gallery' && (
              <div className="p-4 sm:p-5">
                <div className="grid grid-cols-2 gap-4 md:grid-cols-3">
                  {galeria.map((item) => (
                    <div key={item.id} className="overflow-hidden rounded-lg">
                      <div className="relative aspect-square overflow-hidden">
                        <img
                          src={item.url}
                          alt={item.legenda}
                          className="h-full w-full object-cover"
                        />
                      </div>
                      <p className="mt-2 text-xs text-aura-anthracite">{item.legenda}</p>
                    </div>
                  ))}
                  <button
                    type="button"
                    className="flex aspect-square flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed border-[#f1dfd4] text-aura-muted transition-colors hover:bg-[#efdcd1]/30 hover:text-[#7d5141]"
                  >
                    <ImagePlus className="h-8 w-8" />
                    <span className="text-sm">Adicionar foto</span>
                  </button>
                </div>
              </div>
            )}

            {tab === 'notes' && (
              <div className="space-y-4 p-4 sm:p-5">
                {notas.length === 0 ? (
                  <p className="text-sm text-aura-muted">Nenhuma nota interna ainda.</p>
                ) : (
                  notas.map((n) => (
                    <div key={n.id} className="rounded-lg bg-[#f4f3f2] p-4">
                      <div className="mb-1 flex items-center justify-between gap-2">
                        <p className="text-sm font-semibold text-[#7d5141]">Nota por {n.autor}</p>
                        <p className="text-xs text-aura-muted">{formatDateShortBR(n.data)}</p>
                      </div>
                      <p className="text-sm italic text-aura-anthracite">&ldquo;{n.texto}&rdquo;</p>
                    </div>
                  ))
                )}
                <button
                  type="button"
                  onClick={() => {
                    setNotaTexto('')
                    setError('')
                    setNotaOpen(true)
                  }}
                  className="flex w-full items-center justify-center gap-2 rounded-lg border border-[#f1dfd4] py-3 text-sm text-aura-muted transition-colors hover:bg-[#efdcd1]/30 hover:text-[#7d5141]"
                >
                  <NotebookPen className="h-4 w-4" />
                  Escrever nova nota
                </button>
              </div>
            )}
          </div>
        </div>
      </section>

      {/* Footer metrics bar */}
      <div className="fixed bottom-16 left-0 right-0 z-30 border-t border-[#efdcd1] bg-white/90 px-4 py-3 backdrop-blur-md lg:bottom-0 lg:left-64">
        <div className="mx-auto flex max-w-6xl flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex flex-wrap gap-6 sm:gap-8">
            <div>
              <p className="text-[10px] font-bold uppercase tracking-wider text-aura-muted">
                Lifetime Value (LTV)
              </p>
              <p className="font-display text-lg font-semibold text-[#7d5141]">
                {formatBRL(metrics.ltv)}
              </p>
            </div>
            <div className="border-l border-[#efdcd1] pl-6 sm:pl-8">
              <p className="text-[10px] font-bold uppercase tracking-wider text-aura-muted">
                Frequência mensal
              </p>
              <p className="font-display text-lg font-semibold text-[#7d5141]">
                {metrics.frequenciaMensal}x
              </p>
            </div>
            <div className="border-l border-[#efdcd1] pl-6 sm:pl-8">
              <p className="text-[10px] font-bold uppercase tracking-wider text-aura-muted">
                Ticket médio
              </p>
              <p className="font-display text-lg font-semibold text-[#7d5141]">
                {formatBRL(metrics.ticketMedio)}
              </p>
            </div>
          </div>
          {metrics.proximoAgendamento && (
            <div className="hidden items-center gap-2 md:flex">
              <span className="text-sm text-aura-muted">Próximo agendamento:</span>
              <span className="rounded-full bg-[#7d5141] px-3 py-1 text-sm font-semibold text-white">
                {formatDateShortBR(metrics.proximoAgendamento.data)},{' '}
                {formatTimeBR(metrics.proximoAgendamento.hora)}
              </span>
            </div>
          )}
        </div>
      </div>

      <DonaFooter />

      <Modal
        open={editOpen}
        onClose={() => setEditOpen(false)}
        title="Editar perfil"
        footer={
          <>
            <Button variant="secondary" onClick={() => setEditOpen(false)}>
              Cancelar
            </Button>
            <Button className="bg-[#7d5141] hover:bg-[#996958]" onClick={saveEdit}>
              Salvar
            </Button>
          </>
        }
      >
        {error && editOpen && <p className="mb-3 text-sm text-red-600">{error}</p>}
        <div className="space-y-4">
          <Input label="Nome completo" value={nome} onChange={(e) => setNome(e.target.value)} />
          <Input
            label="WhatsApp"
            type="tel"
            inputMode="numeric"
            autoComplete="tel"
            maxLength={15}
            value={telefone}
            onChange={(e) => setTelefone(maskPhoneBRInput(e.target.value))}
            placeholder="(11) 99999-9999"
          />
          <Input label="E-mail" type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
        </div>
      </Modal>

      <Modal
        open={notaOpen}
        onClose={() => setNotaOpen(false)}
        title="Nova nota interna"
        footer={
          <>
            <Button variant="secondary" onClick={() => setNotaOpen(false)}>
              Cancelar
            </Button>
            <Button className="bg-[#7d5141] hover:bg-[#996958]" onClick={saveNota}>
              Salvar nota
            </Button>
          </>
        }
      >
        {error && notaOpen && <p className="mb-3 text-sm text-red-600">{error}</p>}
        <textarea
          value={notaTexto}
          onChange={(e) => setNotaTexto(e.target.value)}
          rows={4}
          placeholder="Observações visíveis apenas para a equipe…"
          className="w-full rounded-lg border border-aura-border px-3 py-2 text-sm focus:border-[#7d5141] focus:outline-none focus:ring-2 focus:ring-[#7d5141]/15"
        />
      </Modal>
    </DonaLayout>
  )
}
