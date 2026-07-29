import { useState } from 'react'
import { Flower2, Leaf, Plus, Scissors, Sparkles } from 'lucide-react'
import { DonaLayout, DonaFooter, PageHeader, MetricCard } from '../../components/dona/DonaLayout'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Input'
import { Modal } from '../../components/ui/Modal'
import { useAuth } from '../../contexts/AuthContext'
import { countByEspecialidade, createEspecialidade, getDb, updateEspecialidade } from '../../utils/mockDb'

const iconMap: Record<string, typeof Scissors> = {
  scissors: Scissors,
  sparkles: Sparkles,
  flower: Flower2,
  leaf: Leaf,
}

export function EspecialidadesDona() {
  const { session } = useAuth()
  const tenantId = session?.user.tenant_id ?? ''
  const [db, setDb] = useState(getDb())
  const [selected, setSelected] = useState<string | null>(null)
  const [modalOpen, setModalOpen] = useState(false)
  const [editId, setEditId] = useState<string | null>(null)
  const [nome, setNome] = useState('')

  const refresh = () => setDb(getDb())
  const especialidades = db.especialidades.filter((e) => e.tenant_id === tenantId)

  const openCreate = () => {
    setEditId(null)
    setNome('')
    setModalOpen(true)
  }

  const openEdit = (id: string, current: string) => {
    setEditId(id)
    setNome(current)
    setModalOpen(true)
  }

  const save = async () => {
    if (!nome.trim()) return
    if (editId) await updateEspecialidade(editId, nome)
    else await createEspecialidade(tenantId, nome)
    refresh()
    setModalOpen(false)
  }

  return (
    <DonaLayout searchPlaceholder="Buscar especialidades…">
      <PageHeader
        title="Gestão de Especialidades"
        subtitle="Organize as áreas de atuação da sua equipe."
        action={
          <Button onClick={openCreate}>
            <Plus className="h-4 w-4" />
            Nova Especialidade
          </Button>
        }
      />

      <div className="mb-6 max-w-xs">
        <MetricCard
          label="Especialidades cadastradas"
          value={String(especialidades.length).padStart(2, '0')}
        />
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {especialidades.map((esp, i) => {
          const { profs, servicos } = countByEspecialidade(tenantId, esp.id)
          const Icon = iconMap[['scissors', 'sparkles', 'flower', 'leaf'][i % 4]] ?? Sparkles
          const isSelected = selected === esp.id
          return (
            <button
              key={esp.id}
              type="button"
              onClick={() => setSelected(esp.id)}
              onDoubleClick={() => openEdit(esp.id, esp.nome)}
              className={[
                'rounded-lg border p-6 text-left shadow-sm transition-all',
                isSelected
                  ? 'border-aura-primary bg-aura-primary/10 shadow-md'
                  : 'border-aura-border bg-white hover:border-aura-primary/40',
              ].join(' ')}
            >
              <Icon className={`mb-4 h-8 w-8 ${isSelected ? 'text-aura-primary' : 'text-aura-muted'}`} />
              <p className="font-display text-lg font-semibold">{esp.nome}</p>
              <p className="mt-2 text-sm text-aura-muted">
                {servicos} Serviços · {profs} Profissionais
              </p>
            </button>
          )
        })}
      </div>

      <DonaFooter />

      <Modal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        title={editId ? 'Editar especialidade' : 'Nova especialidade'}
        footer={
          <>
            <Button variant="secondary" onClick={() => setModalOpen(false)}>Cancelar</Button>
            <Button onClick={save}>Salvar</Button>
          </>
        }
      >
        <Input label="Nome" value={nome} onChange={(e) => setNome(e.target.value)} placeholder="Ex: Cabelo, Unhas…" />
      </Modal>
    </DonaLayout>
  )
}
