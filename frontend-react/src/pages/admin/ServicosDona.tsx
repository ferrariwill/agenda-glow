import { useMemo, useState } from 'react'
import { ChevronLeft, ChevronRight, Plus } from 'lucide-react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { DonaLayout, DonaFooter } from '../../components/dona/DonaLayout'
import { ServicoActionsMenu } from '../../components/dona/ServicoActionsMenu'
import { Alert } from '../../components/ui/Alert'
import { ConfirmModal } from '../../components/ui/Modal'
import { useAuth } from '../../contexts/AuthContext'
import type { Servico } from '../../types'
import {
  deleteServico,
  getDb,
  updateServico,
} from '../../utils/mockDb'
import { formatBRL } from '../../utils/format'

const PAGE_SIZE = 8
const GLASS =
  'rounded-xl border border-[#e5d3c8]/30 bg-white shadow-[0px_4px_20px_rgba(183,132,114,0.08)]'

const DEFAULT_IMG =
  'https://images.unsplash.com/photo-1560066984-138dadb4c035?w=400&q=80'

function statusBadge(ativo: boolean) {
  return ativo
    ? 'bg-[#E7F3E7] text-[#2D5A2D]'
    : 'bg-[#F5F5F5] text-[#555555]'
}

export function ServicosDona() {
  const navigate = useNavigate()
  const location = useLocation()
  const { session } = useAuth()
  const tenantId = session?.user.tenant_id ?? ''
  const [db, setDb] = useState(getDb())
  const [search, setSearch] = useState('')
  const [categoriaId, setCategoriaId] = useState<string | 'all'>('all')
  const [page, setPage] = useState(1)
  const [deleteTarget, setDeleteTarget] = useState<Servico | null>(null)
  const [success, setSuccess] = useState(
    (location.state as { success?: string } | null)?.success ?? '',
  )

  const refresh = () => setDb(getDb())

  const categorias = db.categorias.filter((c) => c.tenant_id === tenantId)
  const totalAtivos = db.servicos.filter((s) => s.tenant_id === tenantId && s.ativo).length

  const servicos = useMemo(() => {
    const q = search.trim().toLowerCase()
    let list = db.servicos.filter((s) => s.tenant_id === tenantId)
    if (categoriaId !== 'all') list = list.filter((s) => s.categoria_id === categoriaId)
    if (q) {
      list = list.filter((s) => {
        const cat = categorias.find((c) => c.id === s.categoria_id)?.nome ?? ''
        return (
          s.nome.toLowerCase().includes(q) ||
          s.descricao?.toLowerCase().includes(q) ||
          cat.toLowerCase().includes(q) ||
          String(s.preco).includes(q)
        )
      })
    }
    return list.sort((a, b) => a.nome.localeCompare(b.nome))
  }, [db.servicos, tenantId, categoriaId, search, categorias])

  const totalPages = Math.max(1, Math.ceil(servicos.length / PAGE_SIZE))
  const pageSafe = Math.min(page, totalPages)
  const pageItems = servicos.slice((pageSafe - 1) * PAGE_SIZE, pageSafe * PAGE_SIZE)

  const toggleAtivo = (s: Servico) => {
    updateServico(s.id, { ativo: !s.ativo })
    refresh()
    setSuccess(s.ativo ? 'Serviço desativado.' : 'Serviço reativado.')
  }

  const confirmDelete = () => {
    if (!deleteTarget) return
    deleteServico(deleteTarget.id)
    refresh()
    setDeleteTarget(null)
    setSuccess('Serviço removido do catálogo.')
  }

  const getCategoriaNome = (id?: string) =>
    categorias.find((c) => c.id === id)?.nome ?? '—'

  return (
    <DonaLayout
      searchPlaceholder="Buscar serviços, preços ou categorias…"
      searchValue={search}
      onSearchChange={(v) => {
        setSearch(v)
        setPage(1)
      }}
    >
      <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="font-display text-2xl font-semibold text-[#7d5141] sm:text-3xl">
            Gestão de serviços
          </h1>
          <p className="mt-1 text-[#615b58]">Configure o cardápio de luxo do seu salão.</p>
        </div>
        <Link
          to="/admin/servicos/novo"
          className="flex items-center justify-center gap-2 rounded-xl bg-[#7d5141] px-6 py-3 text-sm font-semibold text-white shadow-md transition-all hover:opacity-90 active:scale-95"
        >
          <Plus className="h-4 w-4" />
          Novo serviço
        </Link>
      </div>

      {success && (
        <Alert variant="info" className="mb-4" onDismiss={() => setSuccess('')}>
          {success}
        </Alert>
      )}

      <div className="mb-8 grid grid-cols-12 gap-6">
        <div className={`col-span-12 flex flex-wrap items-center gap-3 p-6 xl:col-span-8 ${GLASS}`}>
          <span className="mr-1 text-xs font-bold uppercase tracking-widest text-[#695c53]">
            Filtrar por:
          </span>
          <button
            type="button"
            onClick={() => {
              setCategoriaId('all')
              setPage(1)
            }}
            className={[
              'rounded-full px-4 py-2 text-sm font-semibold transition-colors',
              categoriaId === 'all'
                ? 'bg-[#7d5141] text-white'
                : 'bg-[#f4f3f2] text-[#615b58] hover:bg-[#efdcd1]',
            ].join(' ')}
          >
            Todos
          </button>
          {categorias.map((c) => (
            <button
              key={c.id}
              type="button"
              onClick={() => {
                setCategoriaId(c.id)
                setPage(1)
              }}
              className={[
                'rounded-full px-4 py-2 text-sm font-semibold transition-colors',
                categoriaId === c.id
                  ? 'bg-[#7d5141] text-white'
                  : 'bg-[#f4f3f2] text-[#615b58] hover:bg-[#efdcd1]',
              ].join(' ')}
            >
              {c.nome}
            </button>
          ))}
        </div>
        <div
          className={`col-span-12 flex flex-col justify-center p-6 text-[#fffbff] xl:col-span-4 ${GLASS} bg-[#996958] !border-[#996958]/20`}
        >
          <p className="mb-1 text-xs font-bold uppercase tracking-widest opacity-80">
            Total de serviços
          </p>
          <div className="flex items-baseline gap-2">
            <span className="font-display text-4xl font-bold leading-none">{totalAtivos}</span>
            <span className="text-sm opacity-90">Ativos no catálogo</span>
          </div>
        </div>
      </div>

      <div className={`overflow-hidden ${GLASS}`}>
        <div className="overflow-x-auto">
          <table className="w-full border-collapse text-left">
            <thead>
              <tr className="border-b border-[#d6c2bd]/30 bg-[#f4f3f2]/50">
                <th className="px-6 py-4 text-xs font-bold uppercase tracking-wider text-[#695c53]">
                  Nome do serviço
                </th>
                <th className="px-6 py-4 text-xs font-bold uppercase tracking-wider text-[#695c53]">
                  Categoria
                </th>
                <th className="px-6 py-4 text-center text-xs font-bold uppercase tracking-wider text-[#695c53]">
                  Duração
                </th>
                <th className="px-6 py-4 text-right text-xs font-bold uppercase tracking-wider text-[#695c53]">
                  Preço base
                </th>
                <th className="px-6 py-4 text-center text-xs font-bold uppercase tracking-wider text-[#695c53]">
                  Status
                </th>
                <th className="px-6 py-4" />
              </tr>
            </thead>
            <tbody className="divide-y divide-[#d6c2bd]/10">
              {pageItems.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-6 py-12 text-center text-sm text-[#83746f]">
                    Nenhum serviço encontrado.
                  </td>
                </tr>
              ) : (
                pageItems.map((s) => (
                  <tr
                    key={s.id}
                    className="group cursor-pointer transition-colors hover:bg-[#f4f3f2] hover:translate-x-1"
                    onClick={() => navigate(`/admin/servicos/${s.id}/edit`)}
                  >
                    <td className="px-6 py-5">
                      <div className="flex items-center gap-4">
                        <div className="h-12 w-12 shrink-0 overflow-hidden rounded-lg bg-[#efdcd1]/20">
                          <img
                            src={s.imagem_url ?? DEFAULT_IMG}
                            alt=""
                            className="h-full w-full object-cover"
                          />
                        </div>
                        <div className="min-w-0">
                          <p className="font-semibold text-[#7d5141]">{s.nome}</p>
                          {s.descricao && (
                            <p className="truncate text-xs text-[#615b58]">{s.descricao}</p>
                          )}
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-5">
                      <span className="rounded bg-[#eeeeed] px-2 py-1 text-xs font-bold uppercase tracking-wider text-[#695c53]">
                        {getCategoriaNome(s.categoria_id)}
                      </span>
                    </td>
                    <td className="px-6 py-5 text-center text-sm text-[#615b58]">
                      {s.duracao_minutos} min
                    </td>
                    <td className="px-6 py-5 text-right font-semibold text-[#7d5141]">
                      {formatBRL(s.preco)}
                    </td>
                    <td className="px-6 py-5 text-center">
                      <span
                        className={`inline-flex items-center rounded-full px-2 py-1 text-xs font-semibold ${statusBadge(s.ativo)}`}
                      >
                        {s.ativo ? 'Ativo' : 'Inativo'}
                      </span>
                    </td>
                    <td className="px-6 py-5 text-right" onClick={(e) => e.stopPropagation()}>
                      <ServicoActionsMenu
                        ativo={s.ativo}
                        onEdit={() => navigate(`/admin/servicos/${s.id}/edit`)}
                        onToggleAtivo={() => toggleAtivo(s)}
                        onDelete={() => setDeleteTarget(s)}
                      />
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        <div className="flex flex-col items-center justify-between gap-4 border-t border-[#d6c2bd]/10 bg-[#f4f3f2]/30 px-6 py-4 sm:flex-row">
          <p className="text-sm text-[#615b58]">
            Mostrando{' '}
            <span className="font-semibold">{pageItems.length}</span> de{' '}
            <span className="font-semibold">{servicos.length}</span> serviços
          </p>
          <div className="flex items-center gap-2">
            <button
              type="button"
              disabled={pageSafe <= 1}
              onClick={() => setPage((p) => p - 1)}
              className="flex h-10 w-10 items-center justify-center rounded-xl border border-[#d6c2bd]/30 transition-all hover:bg-[#e9e8e7] disabled:opacity-30"
              aria-label="Página anterior"
            >
              <ChevronLeft className="h-4 w-4" />
            </button>
            {Array.from({ length: Math.min(totalPages, 5) }, (_, i) => i + 1).map((p) => (
              <button
                key={p}
                type="button"
                onClick={() => setPage(p)}
                className={[
                  'flex h-10 w-10 items-center justify-center rounded-xl border border-[#d6c2bd]/30 text-sm font-bold transition-all',
                  p === pageSafe
                    ? 'bg-white text-[#7d5141]'
                    : 'hover:bg-[#e9e8e7] text-[#615b58]',
                ].join(' ')}
              >
                {p}
              </button>
            ))}
            <button
              type="button"
              disabled={pageSafe >= totalPages}
              onClick={() => setPage((p) => p + 1)}
              className="flex h-10 w-10 items-center justify-center rounded-xl border border-[#d6c2bd]/30 transition-all hover:bg-[#e9e8e7] disabled:opacity-30"
              aria-label="Próxima página"
            >
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      </div>

      <DonaFooter />

      <ConfirmModal
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={confirmDelete}
        title="Excluir serviço"
        message={
          deleteTarget
            ? `Deseja remover "${deleteTarget.nome}" permanentemente do catálogo?`
            : ''
        }
        confirmLabel="Excluir"
      />
    </DonaLayout>
  )
}
