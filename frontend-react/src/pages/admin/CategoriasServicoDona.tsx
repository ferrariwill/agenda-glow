import { useEffect, useState } from 'react'
import { Flower2, Leaf, Plus, Scissors, Sparkles } from 'lucide-react'
import { DonaLayout, DonaFooter, PageHeader, MetricCard } from '../../components/dona/DonaLayout'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Input'
import { ConfirmModal, Modal } from '../../components/ui/Modal'
import { Alert } from '../../components/ui/Alert'
import { useAuth } from '../../contexts/AuthContext'
import { ApiError } from '../../lib/api'
import type { CategoriaServico } from '../../types'
import {
  createCategoriaServico,
  deleteCategoriaServico,
  getDb,
  listCategoriasServico,
  updateCategoriaServico,
} from '../../utils/mockDb'

const ICON_OPTIONS = [
  { value: 'scissors', label: 'Tesoura', Icon: Scissors },
  { value: 'sparkles', label: 'Brilho', Icon: Sparkles },
  { value: 'flower', label: 'Flor', Icon: Flower2 },
  { value: 'leaf', label: 'Folha', Icon: Leaf },
] as const

const iconMap: Record<string, typeof Scissors> = {
  scissors: Scissors,
  sparkles: Sparkles,
  flower: Flower2,
  leaf: Leaf,
}

export function CategoriasServicoDona() {
  const { session } = useAuth()
  const tenantId = session?.user.tenant_id ?? ''
  const [categorias, setCategorias] = useState<CategoriaServico[]>([])
  const [selected, setSelected] = useState<string | null>(null)
  const [modalOpen, setModalOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<CategoriaServico | null>(null)
  const [editId, setEditId] = useState<string | null>(null)
  const [nome, setNome] = useState('')
  const [icone, setIcone] = useState<string>('scissors')
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  const refresh = async () => {
    const list = await listCategoriasServico(tenantId)
    setCategorias(list)
  }

  useEffect(() => {
    void refresh().catch(() => setCategorias(getDb().categorias.filter((c) => c.tenant_id === tenantId)))
  }, [tenantId])

  const openCreate = () => {
    setEditId(null)
    setNome('')
    setIcone('scissors')
    setError('')
    setModalOpen(true)
  }

  const openEdit = (cat: CategoriaServico) => {
    setEditId(cat.id)
    setNome(cat.nome)
    setIcone(cat.icone || 'scissors')
    setError('')
    setModalOpen(true)
  }

  const save = async () => {
    if (!nome.trim()) {
      setError('Informe o nome da categoria.')
      return
    }
    setSaving(true)
    setError('')
    try {
      if (editId) {
        await updateCategoriaServico(editId, { nome: nome.trim(), icone })
      } else {
        await createCategoriaServico(tenantId, { nome: nome.trim(), icone })
      }
      await refresh()
      setModalOpen(false)
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        setError('Já existe uma categoria com este nome.')
      } else {
        setError(err instanceof Error ? err.message : 'Não foi possível salvar.')
      }
    } finally {
      setSaving(false)
    }
  }

  const confirmDelete = async () => {
    if (!deleteTarget) return
    setSaving(true)
    try {
      await deleteCategoriaServico(deleteTarget.id)
      await refresh()
      setDeleteTarget(null)
      if (selected === deleteTarget.id) setSelected(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Não foi possível excluir.')
      setDeleteTarget(null)
    } finally {
      setSaving(false)
    }
  }

  const servicosCount = (categoriaId: string) =>
    getDb().servicos.filter((s) => s.tenant_id === tenantId && s.categoria_id === categoriaId).length

  return (
    <DonaLayout searchPlaceholder="Buscar categorias…">
      <PageHeader
        title="Categorias de serviços"
        subtitle="Organize o cardápio por áreas (Cabelo, Unhas, Estética…)."
        action={
          <Button onClick={openCreate}>
            <Plus className="h-4 w-4" />
            Nova categoria
          </Button>
        }
      />

      {error && !modalOpen && (
        <Alert variant="error" className="mb-6" onDismiss={() => setError('')}>
          {error}
        </Alert>
      )}

      <div className="mb-6 max-w-xs">
        <MetricCard
          label="Categorias cadastradas"
          value={String(categorias.length).padStart(2, '0')}
        />
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {categorias.map((cat) => {
          const Icon = iconMap[cat.icone ?? ''] ?? Sparkles
          const isSelected = selected === cat.id
          return (
            <button
              key={cat.id}
              type="button"
              onClick={() => setSelected(cat.id)}
              onDoubleClick={() => openEdit(cat)}
              className={[
                'rounded-lg border p-6 text-left shadow-sm transition-all',
                isSelected
                  ? 'border-aura-primary bg-aura-primary/10 shadow-md'
                  : 'border-aura-border bg-white hover:border-aura-primary/40',
              ].join(' ')}
            >
              <Icon className={`mb-4 h-8 w-8 ${isSelected ? 'text-aura-primary' : 'text-aura-muted'}`} />
              <p className="font-display text-lg font-semibold">{cat.nome}</p>
              <p className="mt-2 text-sm text-aura-muted">
                {servicosCount(cat.id)} serviços vinculados
              </p>
              <div className="mt-4 flex gap-2">
                <span
                  role="button"
                  tabIndex={0}
                  className="text-xs font-semibold text-aura-primary hover:underline"
                  onClick={(e) => {
                    e.stopPropagation()
                    openEdit(cat)
                  }}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.stopPropagation()
                      openEdit(cat)
                    }
                  }}
                >
                  Editar
                </span>
                <span
                  role="button"
                  tabIndex={0}
                  className="text-xs font-semibold text-red-600 hover:underline"
                  onClick={(e) => {
                    e.stopPropagation()
                    setDeleteTarget(cat)
                  }}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.stopPropagation()
                      setDeleteTarget(cat)
                    }
                  }}
                >
                  Excluir
                </span>
              </div>
            </button>
          )
        })}
      </div>

      {categorias.length === 0 && (
        <p className="mt-4 text-sm text-aura-muted">
          Nenhuma categoria ainda. Crie a primeira para filtrar o menu de serviços.
        </p>
      )}

      <DonaFooter />

      <Modal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        title={editId ? 'Editar categoria' : 'Nova categoria'}
        footer={
          <>
            <Button variant="secondary" onClick={() => setModalOpen(false)}>
              Cancelar
            </Button>
            <Button onClick={save} disabled={saving}>
              {saving ? 'Salvando…' : 'Salvar'}
            </Button>
          </>
        }
      >
        {error && (
          <Alert variant="error" className="mb-4" onDismiss={() => setError('')}>
            {error}
          </Alert>
        )}
        <div className="space-y-4">
          <Input
            label="Nome"
            value={nome}
            onChange={(e) => setNome(e.target.value)}
            placeholder="Ex: Cabelo, Unhas…"
          />
          <div>
            <p className="mb-2 text-xs font-bold uppercase tracking-wider text-aura-muted">
              Ícone
            </p>
            <div className="flex flex-wrap gap-2">
              {ICON_OPTIONS.map(({ value, label, Icon }) => (
                <button
                  key={value}
                  type="button"
                  onClick={() => setIcone(value)}
                  className={[
                    'flex items-center gap-2 rounded-lg border px-3 py-2 text-sm transition-colors',
                    icone === value
                      ? 'border-aura-primary bg-aura-primary/10 text-aura-primary'
                      : 'border-aura-border hover:border-aura-primary/40',
                  ].join(' ')}
                >
                  <Icon className="h-4 w-4" />
                  {label}
                </button>
              ))}
            </div>
          </div>
        </div>
      </Modal>

      <ConfirmModal
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={confirmDelete}
        title="Excluir categoria"
        message={
          deleteTarget
            ? `Excluir "${deleteTarget.nome}"? Serviços vinculados ficam sem categoria.`
            : ''
        }
        confirmLabel="Excluir"
      />
    </DonaLayout>
  )
}
