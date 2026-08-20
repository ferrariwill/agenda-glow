import { useEffect, useRef, useState } from 'react'
import {
  AlertCircle,
  CheckCircle2,
  ChevronRight,
  FileText,
  ImagePlus,
  Info,
  Loader2,
  Package,
  XCircle,
} from 'lucide-react'
import { Link, Navigate, useNavigate, useParams } from 'react-router-dom'
import { DonaLayout, DonaFooter } from '../../components/dona/DonaLayout'
import { Alert } from '../../components/ui/Alert'
import { useAuth } from '../../contexts/AuthContext'
import type { InsumoCategoria } from '../../types'
import {
  createInsumo,
  getInsumoById,
  updateInsumo,
} from '../../utils/mockDb'
import { maskBRLInput, parseBRLInput } from '../../utils/format'

const GLASS =
  'rounded-xl border border-[#e5d3c8]/30 bg-white shadow-[0px_4px_20px_rgba(183,132,114,0.08)]'

const DEFAULT_IMG =
  'https://images.unsplash.com/photo-1560066984-138dadb4c035?w=800&q=80'

const CATEGORIA_LABEL: Record<InsumoCategoria, string> = {
  CUIDADOS_CAPILARES: 'Cuidados Capilares',
  TECNICA_UNHAS: 'Técnica em Unhas',
  ESTETICA: 'Estética',
  GERAL: 'Consumíveis (Algodão, Toalhas)',
}

const UNIDADES = [
  { value: 'ml', label: 'ml' },
  { value: 'g', label: 'g' },
  { value: 'un', label: 'unidade' },
  { value: 'kg', label: 'kg' },
  { value: 'oz', label: 'oz' },
]

const fieldClass =
  'w-full border-0 border-b border-[#efdcd1]/40 bg-transparent py-2.5 text-base transition-colors focus:border-[#7d5141] focus:ring-0'

export function InsumoFormDona() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { session } = useAuth()
  const tenantId = session?.user.tenant_id ?? ''
  const isEdit = Boolean(id)
  const fileRef = useRef<HTMLInputElement>(null)

  const existing = isEdit ? getInsumoById(tenantId, id!) : undefined

  const [nome, setNome] = useState('')
  const [marca, setMarca] = useState('')
  const [categoria, setCategoria] = useState<InsumoCategoria>('ESTETICA')
  const [unidade, setUnidade] = useState('un')
  const [valorUnit, setValorUnit] = useState('')
  const [quantidade, setQuantidade] = useState<number | ''>('')
  const [estoqueMin, setEstoqueMin] = useState(10)
  const [instrucoes, setInstrucoes] = useState('')
  const [imagemUrl, setImagemUrl] = useState<string | null>(null)
  const [previewFile, setPreviewFile] = useState<string | null>(null)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [loaded, setLoaded] = useState(!isEdit)

  useEffect(() => {
    if (!isEdit) return
    if (!existing) {
      setLoaded(true)
      return
    }
    setNome(existing.nome)
    setMarca(existing.marca ?? '')
    setCategoria(existing.categoria)
    setUnidade(existing.unidade)
    setValorUnit(
      existing.valor_unitario.toLocaleString('pt-BR', {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
      }),
    )
    setQuantidade(existing.quantidade)
    setEstoqueMin(existing.estoque_minimo)
    setInstrucoes(existing.instrucoes_uso ?? '')
    setImagemUrl(existing.imagem_url ?? null)
    setLoaded(true)
  }, [isEdit, existing])

  if (isEdit && !existing && loaded) {
    return <Navigate to="/admin/insumos" replace />
  }

  const displayImage = previewFile ?? imagemUrl ?? null

  const handleFile = (file: File | null) => {
    if (!file || !file.type.startsWith('image/')) return
    const reader = new FileReader()
    reader.onload = (e) => {
      const url = e.target?.result as string
      setPreviewFile(url)
      setImagemUrl(url)
    }
    reader.readAsDataURL(file)
  }

  const save = async () => {
    const valor = parseBRLInput(valorUnit.replace('$', '').trim())
    if (!nome.trim()) {
      setError('Informe o nome do produto.')
      return
    }
    if (Number.isNaN(valor) || valor < 0) {
      setError('Informe um custo unitário válido.')
      return
    }

    setSaving(true)
    setError('')

    const quantidadeNum = typeof quantidade === 'number' ? quantidade : 0
    const estoqueIdeal = Math.max(estoqueMin * 2, quantidadeNum, 10)
    const payload = {
      nome: nome.trim(),
      marca: marca.trim() || undefined,
      categoria,
      quantidade: Math.max(0, quantidadeNum),
      estoque_minimo: estoqueMin,
      estoque_ideal: estoqueIdeal,
      valor_unitario: valor,
      unidade,
      instrucoes_uso: instrucoes.trim() || undefined,
      imagem_url: imagemUrl ?? DEFAULT_IMG,
    }

    await new Promise((r) => setTimeout(r, 400))

    if (isEdit && existing) {
      updateInsumo(existing.id, payload)
    } else {
      createInsumo({ tenant_id: tenantId, ...payload, ativo: true })
    }

    setSaving(false)
    setSaved(true)
    setTimeout(() => {
      navigate('/admin/insumos', {
        state: {
          success: isEdit ? 'Insumo atualizado.' : 'Insumo cadastrado no estoque.',
        },
      })
    }, 800)
  }

  const pageTitle = isEdit ? 'Editar insumo' : 'Novo insumo'
  const breadcrumb = isEdit ? nome || 'Insumo' : 'Novo insumo'

  return (
    <DonaLayout searchPlaceholder="Pesquisar insumos ou marcas…">
      <div className="relative">
      <div className="pointer-events-none absolute -right-24 -top-24 -z-10 h-96 w-96 rounded-full bg-[#f3baa6]/10 blur-3xl" />
      <div className="pointer-events-none absolute -bottom-24 -left-24 -z-10 h-72 w-72 rounded-full bg-[#d5c3b8]/20 blur-3xl" />

      <div className="mb-8">
        <nav className="mb-2 flex items-center gap-1 text-xs text-[#50443c]">
          <Link to="/admin/insumos" className="transition-colors hover:text-[#7d5141]">
            Insumos
          </Link>
          <ChevronRight className="h-3 w-3" />
          <span className="font-bold text-[#7d5141]">{breadcrumb}</span>
        </nav>
        <h1 className="font-display text-2xl font-semibold text-[#7d5141] sm:text-3xl">
          {pageTitle}
        </h1>
        <p className="mt-1 text-sm text-[#50443c]">
          Cadastre um novo produto ou consumível para o estoque do seu salão ou spa.
        </p>
      </div>

      {error && (
        <Alert variant="error" className="mb-6" onDismiss={() => setError('')}>
          {error}
        </Alert>
      )}

      <form
        className="grid grid-cols-1 gap-8 lg:grid-cols-12"
        onSubmit={(e) => {
          e.preventDefault()
          save()
        }}
      >
        <div className="space-y-6 lg:col-span-8">
          <div className={`p-6 sm:p-8 ${GLASS}`}>
            <h2 className="mb-6 flex items-center gap-2 font-semibold text-[#7d5141]">
              <Info className="h-5 w-5" />
              Informações gerais
            </h2>
            <div className="grid gap-6 md:grid-cols-2">
              <div className="md:col-span-2">
                <label className="mb-1 block text-xs font-bold uppercase tracking-wider text-[#514440]">
                  Nome do produto
                </label>
                <input
                  type="text"
                  required
                  value={nome}
                  onChange={(e) => setNome(e.target.value)}
                  placeholder="Ex.: Óleo Facial Hidratante"
                  className={fieldClass}
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-bold uppercase tracking-wider text-[#514440]">
                  Marca
                </label>
                <input
                  type="text"
                  value={marca}
                  onChange={(e) => setMarca(e.target.value)}
                  placeholder="Ex.: Aura Botanics"
                  className={fieldClass}
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-bold uppercase tracking-wider text-[#514440]">
                  Categoria
                </label>
                <select
                  value={categoria}
                  onChange={(e) => setCategoria(e.target.value as InsumoCategoria)}
                  className={`${fieldClass} appearance-none`}
                >
                  {(Object.keys(CATEGORIA_LABEL) as InsumoCategoria[]).map((k) => (
                    <option key={k} value={k}>
                      {CATEGORIA_LABEL[k]}
                    </option>
                  ))}
                </select>
              </div>
            </div>
          </div>

          <div className={`p-6 sm:p-8 ${GLASS}`}>
            <h2 className="mb-6 flex items-center gap-2 font-semibold text-[#7d5141]">
              <Package className="h-5 w-5" />
              Estoque e custos
            </h2>
            <div className="grid gap-6 md:grid-cols-3">
              <div>
                <label className="mb-1 block text-xs font-bold uppercase tracking-wider text-[#514440]">
                  Unidade de medida
                </label>
                <select
                  value={unidade}
                  onChange={(e) => setUnidade(e.target.value)}
                  className={`${fieldClass} appearance-none`}
                >
                  {UNIDADES.map((u) => (
                    <option key={u.value} value={u.value}>
                      {u.label}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="mb-1 block text-xs font-bold uppercase tracking-wider text-[#514440]">
                  Custo por unidade
                </label>
                <div className="relative">
                  <span className="absolute left-0 top-1/2 -translate-y-1/2 font-bold text-[#514440]">
                    R$
                  </span>
                  <input
                    type="text"
                    inputMode="decimal"
                    value={valorUnit}
                    onChange={(e) => setValorUnit(maskBRLInput(e.target.value))}
                    placeholder="0,00"
                    className={`${fieldClass} pl-8`}
                  />
                </div>
              </div>
              <div>
                <label className="mb-1 block text-xs font-bold uppercase tracking-wider text-[#514440]">
                  Estoque atual
                </label>
                <input
                  type="number"
                  min={0}
                  value={quantidade}
                  onFocus={() => {
                    if (quantidade === 0 || quantidade === '') setQuantidade('')
                  }}
                  onChange={(e) => {
                    const raw = e.target.value
                    setQuantidade(raw === '' ? '' : Math.max(0, Number(raw)))
                  }}
                  placeholder="0"
                  className={fieldClass}
                />
              </div>
              <div className="md:col-span-3">
                <div className="flex items-start gap-4 rounded-lg border border-[#efdcd1]/30 bg-[#efdcd1]/10 p-4">
                  <AlertCircle className="mt-0.5 h-5 w-5 shrink-0 fill-[#7d5141] text-white" />
                  <div className="flex-1">
                    <label className="mb-1 block text-xs font-bold uppercase tracking-wider text-[#514440]">
                      Alerta de estoque mínimo
                    </label>
                    <p className="mb-3 text-[10px] text-[#50443c]">
                      Receba um aviso quando o estoque ficar abaixo deste nível.
                    </p>
                    <input
                      type="range"
                      min={0}
                      max={100}
                      value={estoqueMin}
                      onChange={(e) => setEstoqueMin(Number(e.target.value))}
                      className="h-1 w-full cursor-pointer accent-[#7d5141]"
                    />
                    <div className="mt-1 flex justify-between text-xs font-bold text-[#7d5141]">
                      <span>0 {unidade}</span>
                      <span>
                        {estoqueMin} {unidade}
                      </span>
                      <span>100 {unidade}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div className={`p-6 sm:p-8 ${GLASS}`}>
            <h2 className="mb-4 flex items-center gap-2 font-semibold text-[#7d5141]">
              <FileText className="h-5 w-5" />
              Instruções de uso
            </h2>
            <textarea
              value={instrucoes}
              onChange={(e) => setInstrucoes(e.target.value)}
              rows={4}
              placeholder="Descreva como o insumo deve ser usado nos tratamentos, dosagem por serviço ou requisitos de armazenamento…"
              className="w-full resize-none rounded-lg border border-[#efdcd1]/40 p-4 text-sm transition-colors focus:border-[#7d5141] focus:ring-1 focus:ring-[#7d5141]/10"
            />
          </div>
        </div>

        <div className="space-y-6 lg:col-span-4">
          <div className={`flex flex-col items-center p-6 sm:p-8 ${GLASS}`}>
            <h2 className="mb-4 w-full self-start font-semibold text-[#7d5141]">
              Foto do produto
            </h2>
            <button
              type="button"
              onClick={() => fileRef.current?.click()}
              onDragOver={(e) => e.preventDefault()}
              onDrop={(e) => {
                e.preventDefault()
                handleFile(e.dataTransfer.files[0] ?? null)
              }}
              className="group relative flex aspect-square w-full cursor-pointer flex-col items-center justify-center overflow-hidden rounded-xl border-2 border-dashed border-[#efdcd1]/50 bg-[#f4f3f2] transition-colors hover:border-[#7d5141]/50"
            >
              {displayImage ? (
                <>
                  <img
                    src={displayImage}
                    alt=""
                    className="absolute inset-0 h-full w-full object-cover"
                  />
                  <div className="absolute inset-0 flex items-center justify-center bg-black/30 opacity-0 transition-opacity group-hover:opacity-100">
                    <span className="rounded-lg bg-white/90 px-3 py-1.5 text-xs font-bold text-[#7d5141]">
                      Trocar imagem
                    </span>
                  </div>
                </>
              ) : (
                <div className="flex flex-col items-center transition-transform group-hover:scale-105">
                  <ImagePlus className="h-10 w-10 text-[#695c53]/40 transition-colors group-hover:text-[#7d5141]" />
                  <p className="mt-4 text-sm text-[#50443c]">Arraste a imagem ou clique para enviar</p>
                  <p className="mt-1 text-[10px] text-[#50443c]/60">PNG ou JPG em alta resolução (máx. 5MB)</p>
                </div>
              )}
            </button>
            <input
              ref={fileRef}
              type="file"
              accept="image/*"
              className="hidden"
              onChange={(e) => handleFile(e.target.files?.[0] ?? null)}
            />
            <p className="mt-4 w-full rounded-lg border border-[#615b58]/10 bg-[#7a7371]/5 p-3 text-left text-[11px italic leading-relaxed text-[#4b4543]">
              Imagens de qualidade ajudam a equipe a identificar itens rapidamente no estoque.
            </p>
          </div>

          <div className="sticky top-24 space-y-3">
            <button
              type="submit"
              disabled={saving || saved}
              className={[
                'flex w-full items-center justify-center gap-2 rounded-xl py-4 text-sm font-semibold text-white shadow-lg transition-all',
                saved
                  ? 'bg-[#7a7371]'
                  : 'bg-[#7d5141] shadow-[#7d5141]/20 hover:scale-[1.02] active:scale-95',
                saving || saved ? 'opacity-90' : '',
              ].join(' ')}
            >
              {saving ? (
                <>
                  <Loader2 className="h-5 w-5 animate-spin" />
                  Salvando…
                </>
              ) : saved ? (
                <>
                  <CheckCircle2 className="h-5 w-5" />
                  Salvo com sucesso
                </>
              ) : (
                <>
                  <CheckCircle2 className="h-5 w-5" />
                  {isEdit ? 'Salvar insumo' : 'Cadastrar insumo'}
                </>
              )}
            </button>
            <Link
              to="/admin/insumos"
              className="flex w-full items-center justify-center gap-2 rounded-xl border border-[#7d5141] py-4 text-sm font-semibold text-[#7d5141] transition-all hover:bg-[#7d5141]/5 active:opacity-80"
            >
              <XCircle className="h-5 w-5" />
              Cancelar
            </Link>
            <div className="border-l-2 border-[#efdcd1]/40 pl-3 pt-4">
              <p className="flex items-center gap-2 text-[10px] text-[#50443c]">
                <Package className="h-3.5 w-3.5" />
                {isEdit && existing
                  ? `Estoque atual: ${existing.quantidade} ${existing.unidade}`
                  : 'Última contagem: ainda não registrada'}
              </p>
            </div>
          </div>
        </div>
      </form>

      <DonaFooter />
      </div>
    </DonaLayout>
  )
}
