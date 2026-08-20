import { useEffect, useRef, useState } from 'react'
import {
  ChevronRight,
  Clock,
  Plus,
  X,
} from 'lucide-react'
import { Link, Navigate, useNavigate, useParams } from 'react-router-dom'
import { DonaLayout, DonaFooter } from '../../components/dona/DonaLayout'
import { ServicoCategoriaSelect } from '../../components/dona/ServicoCategoriaSelect'
import {
  ServicoInsumosBomPanel,
  type ServicoInsumosBomPanelHandle,
} from '../../components/dona/ServicoInsumosBomPanel'
import { Alert } from '../../components/ui/Alert'
import { Modal } from '../../components/ui/Modal'
import { useAuth } from '../../contexts/AuthContext'
import { ApiError } from '../../lib/api'
import {
  createServico,
  getDb,
  getServicoById,
  replaceServiceSupplies,
  updateServico,
} from '../../utils/mockDb'
import { initials, parseBRLInput } from '../../utils/format'

const GLASS =
  'rounded-xl border border-[#e5d3c8]/30 bg-white shadow-[0px_4px_20px_rgba(183,132,114,0.08)]'

const DEFAULT_IMG =
  'https://images.unsplash.com/photo-1560066984-138dadb4c035?w=400&q=80'

const STOCK_IMAGES = [
  'https://images.unsplash.com/photo-1522337360788-8b13dee7a37e?w=800&q=80',
  'https://images.unsplash.com/photo-1604654894610-df63bc536371?w=800&q=80',
  'https://images.unsplash.com/photo-1560066984-138dadb4c035?w=800&q=80',
  'https://images.unsplash.com/photo-1570172619644-dfd03ed5d881?w=800&q=80',
]

export function ServicoFormDona() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { session } = useAuth()
  const tenantId = session?.user.tenant_id ?? ''
  const isEdit = Boolean(id)
  const bomRef = useRef<ServicoInsumosBomPanelHandle>(null)

  const db = getDb()
  const existing = isEdit ? getServicoById(tenantId, id!) : undefined
  const profissionais = db.profissionais.filter((p) => p.tenant_id === tenantId && p.ativo)

  const [nome, setNome] = useState('')
  const [descricao, setDescricao] = useState('')
  const [categoriaId, setCategoriaId] = useState('')
  const [duracao, setDuracao] = useState(60)
  const [preco, setPreco] = useState('')
  const [ativo, setAtivo] = useState(true)
  const [exibirCatalogo, setExibirCatalogo] = useState(true)
  const [agendamentoOnline, setAgendamentoOnline] = useState(true)
  const [profissionalIds, setProfissionalIds] = useState<string[]>([])
  const [imagemUrl, setImagemUrl] = useState(DEFAULT_IMG)
  const [profPickerOpen, setProfPickerOpen] = useState(false)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [loaded, setLoaded] = useState(!isEdit)

  useEffect(() => {
    if (!isEdit) return
    if (!existing) {
      setLoaded(true)
      return
    }
    setNome(existing.nome)
    setDescricao(existing.descricao ?? '')
    setCategoriaId(existing.categoria_id ?? '')
    setDuracao(existing.duracao_minutos)
    setPreco(existing.preco.toFixed(2).replace('.', ','))
    setAtivo(existing.ativo)
    setExibirCatalogo(existing.exibir_catalogo_publico ?? true)
    setAgendamentoOnline(existing.permitir_agendamento_online ?? true)
    setProfissionalIds(existing.profissional_ids ?? [])
    setImagemUrl(existing.imagem_url ?? DEFAULT_IMG)
    setLoaded(true)
  }, [isEdit, existing])

  const profsDisponiveis = profissionais.filter((p) => !profissionalIds.includes(p.id))
  const profsSelecionados = profissionais.filter((p) => profissionalIds.includes(p.id))

  if (isEdit && !existing && loaded) {
    return <Navigate to="/admin/servicos" replace />
  }

  const save = async () => {
    const precoNum = parseBRLInput(preco)
    if (!nome.trim()) {
      setError('Informe o nome do serviço.')
      return
    }
    if (Number.isNaN(precoNum) || precoNum < 0) {
      setError('Informe um preço válido (≥ 0).')
      return
    }
    if (duracao <= 0) {
      setError('A duração deve ser maior que zero.')
      return
    }
    if (duracao < 15) {
      setError('A duração mínima recomendada é 15 minutos.')
      return
    }

    const payload = {
      nome: nome.trim(),
      descricao: descricao.trim() || undefined,
      categoria_id: categoriaId || null,
      duracao_minutos: duracao,
      preco: precoNum,
      ativo,
      exibir_catalogo_publico: exibirCatalogo,
      permitir_agendamento_online: agendamentoOnline,
      profissional_ids: profissionalIds,
      imagem_url: imagemUrl,
    }

    setSaving(true)
    setError('')
    try {
      let servicoId = existing?.id
      if (isEdit && existing) {
        await updateServico(existing.id, payload)
      } else {
        const created = await createServico({ tenant_id: tenantId, ...payload })
        servicoId = created.id
      }

      const bomItens = bomRef.current?.getItens() ?? []
      if (servicoId) {
        await replaceServiceSupplies(servicoId, bomItens)
      }

      navigate('/admin/servicos', {
        state: {
          success: isEdit ? 'Serviço atualizado.' : 'Serviço criado no catálogo.',
        },
      })
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.status === 409) setError('Conflito ao salvar. Verifique os dados.')
        else if (err.status === 400) setError(err.message || 'Dados inválidos.')
        else if (err.status === 404) setError('Serviço ou insumo não encontrado.')
        else setError(err.message || 'Falha ao salvar o serviço.')
      } else {
        setError(err instanceof Error ? err.message : 'Falha ao salvar o serviço.')
      }
    } finally {
      setSaving(false)
    }
  }

  const pageTitle = isEdit ? 'Editar serviço' : 'Novo serviço'
  const breadcrumbLabel = isEdit ? nome || 'Serviço' : 'Novo serviço'

  return (
    <DonaLayout searchPlaceholder="Buscar serviços, preços ou categorias…">
      <div className="mb-8 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <nav className="mb-2 flex items-center gap-2 text-sm text-[#615b58]">
            <Link to="/admin/servicos" className="transition-colors hover:text-[#7d5141]">
              Serviços
            </Link>
            <ChevronRight className="h-3.5 w-3.5" />
            <span className="font-semibold text-[#7d5141]">{breadcrumbLabel}</span>
          </nav>
          <h1 className="font-display text-2xl font-semibold text-[#7d5141] sm:text-3xl">
            {pageTitle}
          </h1>
        </div>
        <div className="flex flex-wrap gap-3">
          <Link
            to="/admin/servicos"
            className="rounded-xl border border-[#d6c2bd]/40 px-6 py-3 text-sm font-semibold text-[#615b58] transition-colors hover:bg-[#f4f3f2]"
          >
            Voltar
          </Link>
          <button
            type="button"
            onClick={save}
            disabled={saving}
            className="rounded-xl bg-[#7d5141] px-6 py-3 text-sm font-semibold text-white shadow-md transition-all hover:opacity-90 active:scale-95 disabled:opacity-60"
          >
            {saving ? 'Salvando…' : isEdit ? 'Salvar alterações' : 'Criar serviço'}
          </button>
        </div>
      </div>

      {error && (
        <Alert variant="error" className="mb-6" onDismiss={() => setError('')}>
          {error}
        </Alert>
      )}

      <div className="grid grid-cols-12 gap-6">
        <div className="col-span-12 space-y-6 xl:col-span-8">
          <div className={`space-y-6 p-6 sm:p-8 ${GLASS}`}>
            <h2 className="border-b border-[#d6c2bd]/10 pb-2 font-semibold text-[#7d5141]">
              Informações gerais
            </h2>
            <div className="space-y-4">
              <div>
                <label className="mb-1 block text-xs font-bold uppercase tracking-wider text-[#695c53]">
                  Nome do serviço
                </label>
                <input
                  type="text"
                  value={nome}
                  onChange={(e) => setNome(e.target.value)}
                  className="w-full rounded-lg border-none bg-[#f4f3f2] px-4 py-2.5 text-base focus:ring-1 focus:ring-[#7d5141]"
                  placeholder="Ex.: Mechas Californianas"
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-bold uppercase tracking-wider text-[#695c53]">
                  Descrição detalhada
                </label>
                <textarea
                  value={descricao}
                  onChange={(e) => setDescricao(e.target.value)}
                  rows={6}
                  className="min-h-[150px] w-full resize-y rounded-lg border-none bg-[#f4f3f2] px-4 py-3 text-base text-[#615b58] focus:ring-1 focus:ring-[#7d5141]"
                  placeholder="Descreva o procedimento, inclusões e diferenciais…"
                />
              </div>
              <div className="grid gap-4 sm:grid-cols-3">
                <div>
                  <label className="mb-1 block text-xs font-bold uppercase tracking-wider text-[#695c53]">
                    Categoria
                  </label>
                  <ServicoCategoriaSelect
                    tenantId={tenantId}
                    value={categoriaId}
                    onChange={setCategoriaId}
                  />
                </div>
                <div>
                  <label className="mb-1 block text-xs font-bold uppercase tracking-wider text-[#695c53]">
                    Duração
                  </label>
                  <div className="relative">
                    <Clock className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[#695c53]" />
                    <input
                      type="number"
                      min={15}
                      step={15}
                      value={duracao}
                      onChange={(e) => setDuracao(Number(e.target.value))}
                      className="w-full rounded-lg border-none bg-[#f4f3f2] py-2.5 pl-10 pr-12 text-base focus:ring-1 focus:ring-[#7d5141]"
                    />
                    <span className="absolute right-3 top-1/2 -translate-y-1/2 text-sm text-[#83746f]">
                      min
                    </span>
                  </div>
                </div>
                <div>
                  <label className="mb-1 block text-xs font-bold uppercase tracking-wider text-[#695c53]">
                    Preço base
                  </label>
                  <div className="relative">
                    <span className="absolute left-3 top-1/2 -translate-y-1/2 text-[#695c53]">
                      R$
                    </span>
                    <input
                      type="text"
                      inputMode="decimal"
                      value={preco}
                      onChange={(e) => setPreco(e.target.value)}
                      placeholder="450,00"
                      className="w-full rounded-lg border-none bg-[#f4f3f2] py-2.5 pl-10 pr-4 text-base focus:ring-1 focus:ring-[#7d5141]"
                    />
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div className={`p-6 sm:p-8 ${GLASS}`}>
            <h2 className="mb-4 border-b border-[#d6c2bd]/10 pb-2 font-semibold text-[#7d5141]">
              Profissionais habilitados
            </h2>
            <div className="flex flex-wrap items-center gap-3">
              {profsSelecionados.map((p) => (
                <div
                  key={p.id}
                  className="flex items-center gap-2 rounded-lg border border-[#d6c2bd]/20 bg-[#f4f3f2] p-2 pr-3"
                >
                  <div className="flex h-8 w-8 items-center justify-center rounded-full bg-[#efdcd1] text-xs font-bold text-[#7d5141]">
                    {initials(p.nome)}
                  </div>
                  <span className="text-sm font-medium">{p.nome}</span>
                  <button
                    type="button"
                    onClick={() => setProfissionalIds((ids) => ids.filter((x) => x !== p.id))}
                    className="ml-1 rounded-full p-0.5 text-[#83746f] hover:bg-[#e9e8e7] hover:text-[#7d5141]"
                    aria-label={`Remover ${p.nome}`}
                  >
                    <X className="h-3.5 w-3.5" />
                  </button>
                </div>
              ))}
              {profsDisponiveis.length > 0 && (
                <button
                  type="button"
                  onClick={() => setProfPickerOpen(true)}
                  className="flex h-8 w-8 items-center justify-center rounded-full border-2 border-dashed border-[#d6c2bd]/40 text-[#695c53] transition-colors hover:border-[#7d5141] hover:text-[#7d5141]"
                  aria-label="Adicionar profissional"
                >
                  <Plus className="h-4 w-4" />
                </button>
              )}
            </div>
            {profsSelecionados.length === 0 && (
              <p className="mt-3 text-sm text-[#83746f]">
                Nenhum profissional vinculado. Adicione quem pode realizar este serviço.
              </p>
            )}
          </div>
        </div>

        <div className="col-span-12 space-y-6 xl:col-span-4">
          <div className={`space-y-4 p-6 ${GLASS}`}>
            <div className="flex items-center justify-between">
              <span className="font-semibold text-[#7d5141]">Status do serviço</span>
              <button
                type="button"
                role="switch"
                aria-checked={ativo}
                onClick={() => setAtivo(!ativo)}
                className={[
                  'relative h-6 w-12 rounded-full transition-colors',
                  ativo ? 'bg-[#7d5141]' : 'bg-[#d6c2bd]',
                ].join(' ')}
              >
                <span
                  className={[
                    'absolute top-1 h-4 w-4 rounded-full bg-white transition-all',
                    ativo ? 'right-1' : 'left-1',
                  ].join(' ')}
                />
              </button>
            </div>
            <div className="space-y-3 border-t border-[#d6c2bd]/10 pt-3">
              <label className="flex cursor-pointer items-center gap-2">
                <input
                  type="checkbox"
                  checked={exibirCatalogo}
                  onChange={(e) => setExibirCatalogo(e.target.checked)}
                  className="rounded text-[#7d5141] focus:ring-[#7d5141]"
                />
                <span className="text-sm text-[#615b58]">Exibir no catálogo público</span>
              </label>
              <label className="flex cursor-pointer items-center gap-2">
                <input
                  type="checkbox"
                  checked={agendamentoOnline}
                  onChange={(e) => setAgendamentoOnline(e.target.checked)}
                  className="rounded text-[#7d5141] focus:ring-[#7d5141]"
                />
                <span className="text-sm text-[#615b58]">Permitir agendamento online</span>
              </label>
            </div>
          </div>

          <div className={`space-y-4 p-6 ${GLASS}`}>
            <ServicoInsumosBomPanel ref={bomRef} servicoId={isEdit ? id : undefined} />
          </div>

          <div className={`space-y-4 p-6 ${GLASS}`}>
            <h2 className="border-b border-[#d6c2bd]/10 pb-2 font-semibold text-[#7d5141]">
              Imagem de capa
            </h2>
            <div className="group relative aspect-video overflow-hidden rounded-lg bg-[#f4f3f2]">
              <img
                src={imagemUrl}
                alt=""
                className="h-full w-full object-cover opacity-80 transition-opacity group-hover:opacity-100"
              />
              <div className="absolute inset-0 flex items-center justify-center bg-black/20 opacity-0 transition-opacity group-hover:opacity-100">
                <button
                  type="button"
                  onClick={() => {
                    const idx = STOCK_IMAGES.indexOf(imagemUrl)
                    const next = STOCK_IMAGES[(idx + 1) % STOCK_IMAGES.length]
                    setImagemUrl(next)
                  }}
                  className="rounded-lg bg-white/90 px-4 py-2 text-xs font-bold text-[#7d5141] shadow-lg"
                >
                  Alterar foto
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <DonaFooter />

      <Modal
        open={profPickerOpen}
        onClose={() => setProfPickerOpen(false)}
        title="Adicionar profissional"
      >
        <div className="space-y-2">
          {profsDisponiveis.length === 0 ? (
            <p className="text-sm text-[#83746f]">Todos os profissionais já foram adicionados.</p>
          ) : (
            profsDisponiveis.map((p) => (
              <button
                key={p.id}
                type="button"
                onClick={() => {
                  setProfissionalIds((ids) => [...ids, p.id])
                  setProfPickerOpen(false)
                }}
                className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left transition-colors hover:bg-[#f4f3f2]"
              >
                <div className="flex h-9 w-9 items-center justify-center rounded-full bg-[#efdcd1] text-xs font-bold text-[#7d5141]">
                  {initials(p.nome)}
                </div>
                <span className="font-medium">{p.nome}</span>
              </button>
            ))
          )}
        </div>
      </Modal>
    </DonaLayout>
  )
}
