import { useEffect, useMemo, useState, type ReactNode } from 'react'
import {
  ChevronRight,
  DollarSign,
  Edit3,
  ImagePlus,
  Info,
  Link2,
  Percent,
  User,
  Utensils,
  Wallet,
  Briefcase,
  Clock,
} from 'lucide-react'
import { Link, Navigate, useNavigate, useParams } from 'react-router-dom'
import { DonaLayout, DonaFooter } from '../../components/dona/DonaLayout'
import { QuickSpecialtyModal } from '../../components/dona/QuickSpecialtyModal'
import { StickyActionBar, STICKY_ACTION_BAR_SPACE } from '../../components/dona/StickyActionBar'
import { Alert } from '../../components/ui/Alert'
import { ToastFeedback } from '../../components/ui/ToastFeedback'
import { useAuth } from '../../contexts/AuthContext'
import { PlanLimitExceededError, type Expediente, type Profissional } from '../../types'
import {
  createProfissional,
  getDb,
  getProfissionalById,
  updateProfissional,
} from '../../utils/mockDb'
import {
  initials,
  maskPhoneBRInput,
  phoneDigitsToMaskInput,
  toWhatsAppDigits,
} from '../../utils/format'
import { validateAllExpedientes } from './ExpedienteForm'

const GLASS =
  'rounded-xl border border-[#e5d3c8]/30 bg-white shadow-[0px_4px_20px_rgba(183,132,114,0.08)]'

const STOCK_AVATARS = [
  'https://images.unsplash.com/photo-1595476108010-b4d1f102b1b1?w=400&q=80',
  'https://images.unsplash.com/photo-1503951914875-452162b0f3f1?w=400&q=80',
  'https://images.unsplash.com/photo-1604654894610-df63bc536371?w=400&q=80',
  'https://images.unsplash.com/photo-1570172619644-dfd03ed5d881?w=400&q=80',
]

const WEEK_DAYS: { dia: number; label: string }[] = [
  { dia: 1, label: 'SEG' },
  { dia: 2, label: 'TER' },
  { dia: 3, label: 'QUA' },
  { dia: 4, label: 'QUI' },
  { dia: 5, label: 'SEX' },
  { dia: 6, label: 'SAB' },
  { dia: 0, label: 'DOM' },
]

type DayRow = { active: boolean; entrada: string; saida: string }

function defaultWeek(): DayRow[] {
  return WEEK_DAYS.map(({ dia }) => ({
    active: dia >= 1 && dia <= 5,
    entrada: dia === 6 ? '08:00' : '09:00',
    saida: dia === 5 ? '20:00' : dia === 6 ? '14:00' : '18:00',
  }))
}

function expedientesFromWeek(
  days: DayRow[],
  almoco: { inicio: string; fim: string },
): Expediente[] {
  const out: Expediente[] = []
  WEEK_DAYS.forEach(({ dia }, i) => {
    const row = days[i]
    if (!row.active) return
    out.push({
      dia_semana: dia,
      horario_entrada: row.entrada,
      horario_saida: row.saida,
      inicio_almoco: almoco.inicio,
      fim_almoco: almoco.fim,
    })
  })
  return out
}

function weekFromExpedientes(expedientes: Expediente[]): {
  days: DayRow[]
  almoco: { inicio: string; fim: string }
} {
  const days = WEEK_DAYS.map(({ dia }) => {
    const exp = expedientes.find((e) => e.dia_semana === dia)
    if (!exp) return { active: false, entrada: '09:00', saida: '18:00' }
    return {
      active: true,
      entrada: exp.horario_entrada,
      saida: exp.horario_saida,
    }
  })
  const first = expedientes[0]
  return {
    days,
    almoco: {
      inicio: first?.inicio_almoco ?? '12:00',
      fim: first?.fim_almoco ?? '13:00',
    },
  }
}

function SectionHeader({
  icon: Icon,
  title,
}: {
  icon: typeof User
  title: string
}) {
  return (
    <div className="mb-6 flex items-center gap-4">
      <div className="flex h-10 w-10 items-center justify-center rounded-full bg-[#efdcd1] text-[#695c53]">
        <Icon className="h-5 w-5" />
      </div>
      <h2 className="font-display text-xl font-semibold text-[#1a1c1c]">{title}</h2>
    </div>
  )
}

export function ProfissionalFormDona() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { session } = useAuth()
  const tenantId = session?.user.tenant_id ?? ''
  const isEdit = Boolean(id)

  const db = getDb()
  const existing = isEdit ? getProfissionalById(tenantId, id!) : undefined
  const especialidades = db.especialidades.filter((e) => e.tenant_id === tenantId)

  const [nome, setNome] = useState('')
  const [email, setEmail] = useState('')
  const [telefone, setTelefone] = useState('')
  const [dataNascimento, setDataNascimento] = useState('')
  const [biografia, setBiografia] = useState('')
  const [fotoUrl, setFotoUrl] = useState('')
  const [espIds, setEspIds] = useState<string[]>([])
  const [portfolioUrl, setPortfolioUrl] = useState('')
  const [dataContratacao, setDataContratacao] = useState('')
  const [modeloPagamento, setModeloPagamento] = useState<'PERCENTUAL' | 'FIXO'>('PERCENTUAL')
  const [comissao, setComissao] = useState(30)
  const [valorFixo, setValorFixo] = useState(50)
  const [weekDays, setWeekDays] = useState<DayRow[]>(defaultWeek)
  const [almoco, setAlmoco] = useState({ inicio: '12:00', fim: '13:00' })
  const [ativo, setAtivo] = useState(true)
  const [avatarPickerOpen, setAvatarPickerOpen] = useState(false)
  const [error, setError] = useState('')
  const [loaded, setLoaded] = useState(!isEdit)
  const [isQuickModalOpen, setIsQuickModalOpen] = useState(false)
  const [isSaving, setIsSaving] = useState(false)
  const [toast, setToast] = useState('')
  const [formBaseline, setFormBaseline] = useState<string | null>(null)

  useEffect(() => {
    if (!isEdit) return
    if (!existing) {
      setLoaded(true)
      return
    }
    setNome(existing.nome)
    setEmail(existing.email ?? '')
    setTelefone(existing.telefone ? phoneDigitsToMaskInput(existing.telefone) : '')
    setDataNascimento(existing.data_nascimento ?? '')
    setBiografia(existing.biografia ?? '')
    setFotoUrl(existing.foto_url ?? '')
    setEspIds(
      existing.especialidade_ids?.length
        ? existing.especialidade_ids
        : existing.especialidade_id
          ? [existing.especialidade_id]
          : [],
    )
    setPortfolioUrl(existing.portfolio_url ?? '')
    setDataContratacao(existing.data_contratacao ?? '')
    setModeloPagamento(existing.modelo_pagamento ?? 'PERCENTUAL')
    setComissao(existing.comissao_percent)
    setValorFixo(existing.valor_fixo_atendimento ?? 50)
    const { days, almoco: a } = weekFromExpedientes(existing.expedientes)
    setWeekDays(days)
    setAlmoco(a)
    setAtivo(existing.ativo)
    setLoaded(true)
  }, [isEdit, existing])

  useEffect(() => {
    if (!isEdit && especialidades.length && espIds.length === 0) {
      setEspIds([especialidades[0].id])
    }
  }, [isEdit, especialidades, espIds.length])

  const formSnapshot = useMemo(
    () =>
      JSON.stringify({
        nome,
        email,
        telefone,
        dataNascimento,
        biografia,
        fotoUrl,
        espIds,
        portfolioUrl,
        dataContratacao,
        modeloPagamento,
        comissao,
        valorFixo,
        weekDays,
        almoco,
        ativo,
      }),
    [
      nome,
      email,
      telefone,
      dataNascimento,
      biografia,
      fotoUrl,
      espIds,
      portfolioUrl,
      dataContratacao,
      modeloPagamento,
      comissao,
      valorFixo,
      weekDays,
      almoco,
      ativo,
    ],
  )

  useEffect(() => {
    if (!loaded || formBaseline !== null) return
    // Aguarda pré-seleção da primeira especialidade em cadastro novo
    if (!isEdit && especialidades.length > 0 && espIds.length === 0) return
    setFormBaseline(formSnapshot)
  }, [loaded, formBaseline, formSnapshot, isEdit, especialidades.length, espIds.length])

  const hasUnsavedChanges = formBaseline !== null && formSnapshot !== formBaseline

  const expedientes = useMemo(
    () => expedientesFromWeek(weekDays, almoco),
    [weekDays, almoco],
  )

  const espNomes = espIds
    .map((eid) => especialidades.find((e) => e.id === eid)?.nome)
    .filter(Boolean) as string[]

  if (isEdit && !existing && loaded) {
    return <Navigate to="/admin/equipe" replace />
  }

  const toggleEsp = (espId: string) => {
    setEspIds((prev) =>
      prev.includes(espId) ? prev.filter((x) => x !== espId) : [...prev, espId],
    )
  }

  const updateDay = (index: number, patch: Partial<DayRow>) => {
    setWeekDays((rows) => rows.map((r, i) => (i === index ? { ...r, ...patch } : r)))
  }

  const handleSpecialtyCreated = (newSpecialty: { id: string; nome: string }) => {
    setEspIds((prev) => (prev.includes(newSpecialty.id) ? prev : [...prev, newSpecialty.id]))
    setToast(`Especialidade '${newSpecialty.nome}' criada e selecionada!`)
  }

  const cancel = () => {
    navigate('/admin/equipe')
  }

  const save = async () => {
    setError('')
    if (!nome.trim()) {
      setError('Informe o nome completo do profissional.')
      requestAnimationFrame(() => {
        document.getElementById('form-error-alert')?.scrollIntoView({ behavior: 'smooth' })
      })
      return
    }
    if (espIds.length === 0) {
      setError('Selecione ao menos uma especialidade.')
      requestAnimationFrame(() => {
        document.getElementById('form-error-alert')?.scrollIntoView({ behavior: 'smooth' })
      })
      return
    }
    const expErr = validateAllExpedientes(expedientes)
    if (expErr) {
      setError(expErr)
      requestAnimationFrame(() => {
        document.getElementById('form-error-alert')?.scrollIntoView({ behavior: 'smooth' })
      })
      return
    }
    if (modeloPagamento === 'PERCENTUAL' && (comissao < 0 || comissao > 100)) {
      setError('A comissão deve estar entre 0% e 100%.')
      requestAnimationFrame(() => {
        document.getElementById('form-error-alert')?.scrollIntoView({ behavior: 'smooth' })
      })
      return
    }
    if (modeloPagamento === 'FIXO' && valorFixo <= 0) {
      setError('Informe um valor fixo válido por atendimento.')
      requestAnimationFrame(() => {
        document.getElementById('form-error-alert')?.scrollIntoView({ behavior: 'smooth' })
      })
      return
    }

    const payload: Omit<Profissional, 'id' | 'tenant_id'> = {
      nome: nome.trim(),
      email: email.trim() || undefined,
      telefone: telefone.replace(/\D/g, '')
        ? toWhatsAppDigits(telefone)
        : undefined,
      data_nascimento: dataNascimento || undefined,
      biografia: biografia.trim() || undefined,
      foto_url: fotoUrl || undefined,
      especialidade_id: espIds[0],
      especialidade_ids: espIds,
      portfolio_url: portfolioUrl.trim() || undefined,
      data_contratacao: dataContratacao || undefined,
      modelo_pagamento: modeloPagamento,
      comissao_percent: modeloPagamento === 'PERCENTUAL' ? comissao : 0,
      valor_fixo_atendimento: modeloPagamento === 'FIXO' ? valorFixo : undefined,
      ativo: existing?.pendente_aprovacao ? false : ativo,
      pendente_aprovacao: existing?.pendente_aprovacao,
      expedientes,
    }

    setIsSaving(true)
    try {
      if (isEdit && existing) {
        await updateProfissional(existing.id, payload)
        navigate('/admin/equipe', { state: { success: 'Profissional atualizado.' } })
      } else {
        await createProfissional({ tenant_id: tenantId, ...payload, ativo: true })
        navigate('/admin/equipe', { state: { success: 'Profissional cadastrado.' } })
      }
    } catch (err) {
      if (err instanceof PlanLimitExceededError) {
        setError(
          'Limite de profissionais do plano atingido. Faça upgrade do plano SaaS em Configurações.',
        )
      } else {
        setError(
          err instanceof Error
            ? err.message
            : 'Não foi possível salvar os dados. Verifique sua conexão e tente novamente.',
        )
      }
      requestAnimationFrame(() => {
        document.getElementById('form-error-alert')?.scrollIntoView({ behavior: 'smooth' })
      })
    } finally {
      setIsSaving(false)
    }
  }

  const pageTitle = isEdit ? 'Editar Profissional' : 'Adicionar Novo Profissional'
  const breadcrumbLabel = isEdit ? nome || 'Editar' : 'Novo Profissional'
  const subtitle = isEdit
    ? 'Atualize o perfil, especialidades e modelo de remuneração deste membro da equipe.'
    : 'Configure o perfil, especialidades e modelo de remuneração para o novo membro da equipe.'

  return (
    <DonaLayout searchPlaceholder="Buscar profissionais…">
      <header className="mb-8 flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <nav className="mb-2 flex items-center gap-2 text-sm text-[#514440]">
            <Link to="/admin/equipe" className="transition-colors hover:text-[#7d5141]">
              Equipe
            </Link>
            <ChevronRight className="h-4 w-4" />
            <span className="font-bold text-[#7d5141]">{breadcrumbLabel}</span>
          </nav>
          <h1 className="font-display text-3xl font-bold text-[#1a1c1c] sm:text-4xl">
            {pageTitle}
          </h1>
          <p className="mt-1 text-[#514440]">{subtitle}</p>
        </div>
        <div className="flex flex-wrap gap-4">
          <button
            type="button"
            onClick={cancel}
            disabled={isSaving}
            className="rounded-lg border border-[#d6c2bd] px-6 py-2.5 text-sm font-semibold text-[#514440] transition-colors hover:bg-[#f4f3f2] disabled:opacity-50"
          >
            Cancelar
          </button>
          <button
            type="button"
            onClick={() => void save()}
            disabled={isSaving}
            aria-busy={isSaving}
            className="rounded-lg bg-[#7d5141] px-8 py-2.5 text-sm font-semibold text-white shadow-md transition-all hover:opacity-90 active:scale-[0.98] disabled:opacity-50"
          >
            {isSaving
              ? isEdit
                ? 'Salvar alterações...'
                : 'Salvar Profissional...'
              : isEdit
                ? 'Salvar alterações'
                : 'Salvar Profissional'}
          </button>
        </div>
      </header>

      {error && (
        <div id="form-error-alert">
          <Alert variant="error" className="mb-6" onDismiss={() => setError('')}>
            {error}
            {error.includes('upgrade') && (
              <>
                {' '}
                <Link
                  to="/admin/configuracoes"
                  className="font-semibold text-red-900 underline underline-offset-2"
                >
                  Ir para Configurações
                </Link>
              </>
            )}
          </Alert>
        </div>
      )}

      <div className={STICKY_ACTION_BAR_SPACE}>
      <div className="grid grid-cols-12 gap-8">
        {/* Coluna esquerda */}
        <div className="col-span-12 space-y-8 xl:col-span-8">
          {/* Informações Pessoais */}
          <section className={`p-6 sm:p-8 ${GLASS}`}>
            <SectionHeader icon={User} title="Informações Pessoais" />
            <div className="grid grid-cols-2 gap-6">
              <div className="col-span-2 mb-2 flex flex-col items-start gap-6 sm:flex-row sm:items-center">
                <button
                  type="button"
                  onClick={() => setAvatarPickerOpen((o) => !o)}
                  className="group relative shrink-0"
                >
                  <div className="flex h-24 w-24 items-center justify-center overflow-hidden rounded-full border-2 border-dashed border-[#d6c2bd] bg-[#f4f3f2] transition-colors group-hover:border-[#7d5141]">
                    {fotoUrl ? (
                      <img src={fotoUrl} alt="" className="h-full w-full object-cover" />
                    ) : (
                      <ImagePlus className="h-10 w-10 text-[#83746f] group-hover:text-[#7d5141]" />
                    )}
                  </div>
                  <span className="absolute bottom-0 right-0 rounded-full border-2 border-white bg-[#7d5141] p-1 text-white shadow-sm">
                    <Edit3 className="h-3.5 w-3.5" />
                  </span>
                </button>
                <div className="flex-1">
                  <p className="text-xs font-bold uppercase tracking-widest text-[#514440]">
                    Foto do Perfil
                  </p>
                  <p className="mt-1 text-sm text-[#514440]">
                    Recomendado: JPG ou PNG, min. 400×400px. Máx 5MB.
                  </p>
                  <button
                    type="button"
                    onClick={() => setAvatarPickerOpen((o) => !o)}
                    className="mt-2 text-sm font-bold text-[#7d5141] hover:underline"
                  >
                    Carregar imagem
                  </button>
                </div>
              </div>
              {avatarPickerOpen && (
                <div className="col-span-2 flex flex-wrap gap-2">
                  {STOCK_AVATARS.map((url) => (
                    <button
                      key={url}
                      type="button"
                      onClick={() => {
                        setFotoUrl(url)
                        setAvatarPickerOpen(false)
                      }}
                      className={[
                        'h-14 w-14 overflow-hidden rounded-full border-2',
                        fotoUrl === url ? 'border-[#7d5141]' : 'border-transparent',
                      ].join(' ')}
                    >
                      <img src={url} alt="" className="h-full w-full object-cover" />
                    </button>
                  ))}
                  <button
                    type="button"
                    onClick={() => {
                      setFotoUrl('')
                      setAvatarPickerOpen(false)
                    }}
                    className="rounded-lg border border-dashed border-[#d6c2bd] px-3 py-1 text-xs text-[#514440]"
                  >
                    Remover
                  </button>
                </div>
              )}

              <Field label="Nome Completo" className="col-span-2 sm:col-span-1">
                <input
                  type="text"
                  value={nome}
                  onChange={(e) => setNome(e.target.value)}
                  placeholder="Ex: Maria Silva"
                  className={fieldInputClass}
                />
              </Field>
              <Field label="E-mail Profissional" className="col-span-2 sm:col-span-1">
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="maria@agendaglow.com"
                  className={fieldInputClass}
                />
              </Field>
              <Field label="Telefone / WhatsApp" className="col-span-2 sm:col-span-1">
                <input
                  type="tel"
                  inputMode="numeric"
                  autoComplete="tel"
                  maxLength={15}
                  value={telefone}
                  onChange={(e) => setTelefone(maskPhoneBRInput(e.target.value))}
                  placeholder="(11) 99999-9999"
                  className={fieldInputClass}
                />
              </Field>
              <Field label="Data de Nascimento" className="col-span-2 sm:col-span-1">
                <input
                  type="date"
                  value={dataNascimento}
                  onChange={(e) => setDataNascimento(e.target.value)}
                  className={fieldInputClass}
                />
              </Field>
              <Field label="Biografia Curta (Exibida no App)" className="col-span-2">
                <textarea
                  value={biografia}
                  onChange={(e) => setBiografia(e.target.value)}
                  rows={3}
                  placeholder="Conte um pouco sobre a experiência e estilo de atendimento..."
                  className="w-full resize-none rounded-lg border border-[#d6c2bd] bg-transparent p-3 text-sm focus:border-[#7d5141] focus:ring-2 focus:ring-[#7d5141]/10"
                />
              </Field>
            </div>
          </section>

          {/* Informações Profissionais */}
          <section className={`p-6 sm:p-8 ${GLASS}`}>
            <SectionHeader icon={Briefcase} title="Informações Profissionais" />
            <div className="space-y-6">
              <div>
                <p className="mb-3 text-xs font-bold uppercase tracking-widest text-[#514440]">
                  Especialidades
                </p>
                {especialidades.length === 0 ? (
                  <div className="rounded-xl border border-dashed border-[#d6c2bd] bg-[#f4f3f2]/50 p-6 text-center">
                    <p className="text-sm font-medium text-[#514440]">
                      Nenhuma especialidade cadastrada ainda. Clique abaixo para cadastrar a
                      primeira.
                    </p>
                    <button
                      type="button"
                      onClick={() => setIsQuickModalOpen(true)}
                      className="mt-3 inline-flex items-center gap-2 rounded-lg bg-[#7d5141] px-4 py-2 text-sm font-semibold text-white hover:opacity-90"
                    >
                      + Cadastrar Primeira Especialidade
                    </button>
                  </div>
                ) : (
                  <div className="flex flex-wrap gap-2">
                    {especialidades.map((esp) => {
                      const selected = espIds.includes(esp.id)
                      return (
                        <button
                          key={esp.id}
                          type="button"
                          onClick={() => toggleEsp(esp.id)}
                          className={[
                            'rounded-full border px-4 py-2 text-sm transition-all',
                            selected
                              ? 'border-[#7d5141] bg-[#996958]/10 text-[#7d5141]'
                              : 'border-[#d6c2bd] text-[#514440] hover:border-[#7d5141]/50',
                          ].join(' ')}
                        >
                          {esp.nome}
                        </button>
                      )
                    })}
                    <button
                      type="button"
                      onClick={() => setIsQuickModalOpen(true)}
                      className="flex items-center gap-1 rounded-full border border-dashed border-[#d6c2bd] px-4 py-2 text-sm text-[#514440] transition-all hover:border-[#7d5141]"
                    >
                      + Adicionar Outra
                    </button>
                  </div>
                )}
              </div>
              <div className="grid grid-cols-2 gap-6">
                <Field label="Link do Portfólio (Instagram/Behance)">
                  <div className="flex items-center gap-2 border-b border-[#d6c2bd] py-2">
                    <Link2 className="h-5 w-5 shrink-0 text-[#514440]" />
                    <input
                      type="url"
                      value={portfolioUrl}
                      onChange={(e) => setPortfolioUrl(e.target.value)}
                      placeholder="https://"
                      className="flex-1 border-none bg-transparent p-0 text-sm focus:ring-0"
                    />
                  </div>
                </Field>
                <Field label="Data de Contratação">
                  <input
                    type="date"
                    value={dataContratacao}
                    onChange={(e) => setDataContratacao(e.target.value)}
                    className={fieldInputClass}
                  />
                </Field>
              </div>
              {isEdit && existing && !existing.pendente_aprovacao && (
                <label className="flex items-center gap-3 rounded-lg border border-[#d6c2bd]/30 bg-[#f4f3f2]/50 p-4">
                  <input
                    type="checkbox"
                    checked={ativo}
                    onChange={(e) => setAtivo(e.target.checked)}
                    className="rounded text-[#7d5141] focus:ring-[#7d5141]"
                  />
                  <span className="text-sm text-[#514440]">
                    Profissional ativo no salão
                  </span>
                </label>
              )}
              {isEdit && existing?.pendente_aprovacao && (
                <div className="rounded-lg border border-[#f1dfd4] bg-[#f1dfd4]/30 p-4 text-sm text-[#50443c]">
                  Este profissional aguarda aprovação. Use a listagem de equipe para aprovar o
                  cadastro.
                </div>
              )}
            </div>
          </section>

          {/* Comissão */}
          <section className={`p-6 sm:p-8 ${GLASS}`}>
            <SectionHeader icon={Wallet} title="Comissão & Modelo de Pagamento" />
            <div className="space-y-8">
              <div className="grid grid-cols-2 gap-6">
                <label className="cursor-pointer">
                  <input
                    type="radio"
                    name="payment_type"
                    value="PERCENTUAL"
                    checked={modeloPagamento === 'PERCENTUAL'}
                    onChange={() => setModeloPagamento('PERCENTUAL')}
                    className="peer hidden"
                  />
                  <div className="flex flex-col items-center gap-4 rounded-xl border p-6 transition-all peer-checked:border-[#7d5141] peer-checked:bg-[#996958]/5 hover:border-[#7d5141]/50">
                    <Percent
                      className={[
                        'h-10 w-10',
                        modeloPagamento === 'PERCENTUAL' ? 'text-[#7d5141]' : 'text-[#514440]',
                      ].join(' ')}
                    />
                    <span className="text-center font-bold text-[#1a1c1c]">
                      Percentual sobre Procedimento
                    </span>
                    <p className="text-center text-sm text-[#514440]">
                      Comissão baseada no valor total de cada serviço realizado.
                    </p>
                  </div>
                </label>
                <label className="cursor-pointer">
                  <input
                    type="radio"
                    name="payment_type"
                    value="FIXO"
                    checked={modeloPagamento === 'FIXO'}
                    onChange={() => setModeloPagamento('FIXO')}
                    className="peer hidden"
                  />
                  <div className="flex flex-col items-center gap-4 rounded-xl border p-6 transition-all peer-checked:border-[#7d5141] peer-checked:bg-[#996958]/5 hover:border-[#7d5141]/50">
                    <DollarSign
                      className={[
                        'h-10 w-10',
                        modeloPagamento === 'FIXO' ? 'text-[#7d5141]' : 'text-[#514440]',
                      ].join(' ')}
                    />
                    <span className="text-center font-bold text-[#1a1c1c]">
                      Valor Fixo por Atendimento
                    </span>
                    <p className="text-center text-sm text-[#514440]">
                      Um valor pré-definido pago por cada agendamento concluído.
                    </p>
                  </div>
                </label>
              </div>
              <div className="mx-auto max-w-md">
                <p className="mb-3 text-center text-xs font-bold uppercase tracking-widest text-[#514440]">
                  {modeloPagamento === 'PERCENTUAL'
                    ? 'Valor da Comissão Padrão'
                    : 'Valor Fixo por Atendimento'}
                </p>
                <div className="flex items-center justify-center gap-4">
                  <div className="relative w-40">
                    <span className="absolute left-4 top-1/2 -translate-y-1/2 font-bold text-[#514440]">
                      {modeloPagamento === 'PERCENTUAL' ? '%' : 'R$'}
                    </span>
                    <input
                      type="number"
                      min={0}
                      max={modeloPagamento === 'PERCENTUAL' ? 100 : undefined}
                      value={modeloPagamento === 'PERCENTUAL' ? comissao : valorFixo}
                      onChange={(e) => {
                        const v = Number(e.target.value)
                        if (modeloPagamento === 'PERCENTUAL') setComissao(v)
                        else setValorFixo(v)
                      }}
                      className="w-full rounded-xl border border-[#d6c2bd] py-4 pl-10 pr-4 text-center text-2xl font-bold focus:border-[#7d5141] focus:ring-2 focus:ring-[#7d5141]/10"
                    />
                  </div>
                  <span title="Exceções por serviço podem ser definidas depois.">
                    <Info className="h-5 w-5 text-[#514440]" aria-hidden />
                  </span>
                </div>
                <p className="mt-2 text-center text-sm text-[#514440]">
                  Você poderá definir exceções por serviço posteriormente.
                </p>
              </div>
            </div>
          </section>
        </div>

        {/* Coluna direita */}
        <div className="col-span-12 space-y-8 xl:col-span-4">
          <section className={`p-6 sm:p-8 ${GLASS}`}>
            <SectionHeader icon={Clock} title="Escala de Trabalho" />
            <div className="space-y-2">
              {WEEK_DAYS.map(({ label }, index) => {
                const row = weekDays[index]
                return (
                  <div
                    key={label}
                    className={[
                      'flex items-center justify-between rounded-lg p-3',
                      row.active
                        ? 'border border-[#7d5141]/20 bg-[#f4f3f2]'
                        : 'bg-[#e3e2e1]/30 opacity-60',
                    ].join(' ')}
                  >
                    <div className="flex items-center gap-4">
                      <input
                        type="checkbox"
                        checked={row.active}
                        onChange={(e) => updateDay(index, { active: e.target.checked })}
                        className="rounded text-[#7d5141] focus:ring-[#7d5141]"
                      />
                      <span
                        className={[
                          'w-10 text-sm font-bold',
                          row.active ? 'text-[#1a1c1c]' : 'text-[#514440]',
                        ].join(' ')}
                      >
                        {label}
                      </span>
                    </div>
                    {row.active ? (
                      <div className="flex items-center gap-2 text-sm">
                        <input
                          type="time"
                          value={row.entrada}
                          onChange={(e) => updateDay(index, { entrada: e.target.value })}
                          className="w-[4.5rem] border-none bg-transparent p-0 text-[#1a1c1c] focus:ring-0"
                        />
                        <span>—</span>
                        <input
                          type="time"
                          value={row.saida}
                          onChange={(e) => updateDay(index, { saida: e.target.value })}
                          className="w-[4.5rem] border-none bg-transparent p-0 text-[#1a1c1c] focus:ring-0"
                        />
                      </div>
                    ) : (
                      <span className="text-xs font-bold uppercase italic text-[#514440]">
                        Fechado
                      </span>
                    )}
                  </div>
                )
              })}
            </div>
            <div className="mt-6 border-t border-[#d6c2bd]/30 pt-4">
              <h4 className="mb-2 text-lg font-semibold text-[#1a1c1c]">
                Intervalo de Almoço Padrão
              </h4>
              <div className="flex items-center justify-between rounded-lg border border-[#d6c2bd]/30 p-3">
                <Utensils className="h-5 w-5 text-[#514440]" />
                <div className="flex items-center gap-2 text-sm">
                  <input
                    type="time"
                    value={almoco.inicio}
                    onChange={(e) => setAlmoco((a) => ({ ...a, inicio: e.target.value }))}
                    className="w-[4.5rem] border-none bg-transparent p-0 focus:ring-0"
                  />
                  <span>—</span>
                  <input
                    type="time"
                    value={almoco.fim}
                    onChange={(e) => setAlmoco((a) => ({ ...a, fim: e.target.value }))}
                    className="w-[4.5rem] border-none bg-transparent p-0 focus:ring-0"
                  />
                </div>
              </div>
            </div>
          </section>

          {/* Preview */}
          <section className="overflow-hidden rounded-xl border border-[#7d5141]/10 bg-[#7d5141]/5">
            <div className="h-24 bg-[#7d5141]/10" />
            <div className="relative z-10 -mt-10 px-6 pb-6 text-center">
              <div className="mx-auto mb-4 flex h-20 w-20 items-center justify-center overflow-hidden rounded-full border-4 border-[#faf9f8] bg-[#eeeeed]">
                {fotoUrl ? (
                  <img src={fotoUrl} alt="" className="h-full w-full object-cover" />
                ) : nome.trim() ? (
                  <span className="text-2xl font-bold text-[#7d5141]">{initials(nome)}</span>
                ) : (
                  <User className="h-10 w-10 text-[#83746f]" />
                )}
              </div>
              <h4 className="font-display text-lg font-semibold leading-tight text-[#1a1c1c]">
                {nome.trim() || 'Visualização do Perfil'}
              </h4>
              <p className="mt-1 text-sm text-[#514440]">
                Como os clientes verão este profissional no app de agendamento.
              </p>
              <div className="mt-6 space-y-4 text-left">
                <div className="flex flex-wrap gap-2">
                  {espNomes.slice(0, 3).map((n) => (
                    <span
                      key={n}
                      className="rounded border border-[#7d5141]/10 bg-white/50 px-2 py-1 text-[10px] font-bold uppercase text-[#7d5141]"
                    >
                      {n}
                    </span>
                  ))}
                </div>
                {biografia ? (
                  <p className="line-clamp-3 text-sm text-[#514440]">{biografia}</p>
                ) : (
                  <div className="space-y-2">
                    <div className="h-3 w-3/4 animate-pulse rounded-full bg-white/50" />
                    <div className="h-2 w-1/2 animate-pulse rounded-full bg-white/50" />
                  </div>
                )}
                <button
                  type="button"
                  className="mt-2 w-full rounded-lg border border-[#7d5141] bg-white py-2 text-sm font-bold text-[#7d5141]"
                >
                  Ver Perfil Completo
                </button>
              </div>
            </div>
          </section>
        </div>
      </div>

      <DonaFooter />
      </div>

      <StickyActionBar
        onSave={() => void save()}
        onCancel={cancel}
        isSaving={isSaving}
        isEdit={isEdit}
        hasUnsavedChanges={hasUnsavedChanges}
      />

      <QuickSpecialtyModal
        isOpen={isQuickModalOpen}
        onClose={() => setIsQuickModalOpen(false)}
        onSuccess={handleSpecialtyCreated}
        tenantId={tenantId}
      />

      <ToastFeedback message={toast || null} onDismiss={() => setToast('')} />
    </DonaLayout>
  )
}

const fieldInputClass =
  'w-full border-b border-[#d6c2bd] bg-transparent py-2 text-sm transition-colors focus:border-[#7d5141] focus:ring-0'

function Field({
  label,
  children,
  className = '',
}: {
  label: string
  children: ReactNode
  className?: string
}) {
  return (
    <div className={`space-y-2 ${className}`}>
      <label className="block text-xs font-bold uppercase tracking-widest text-[#514440]">
        {label}
      </label>
      {children}
    </div>
  )
}
