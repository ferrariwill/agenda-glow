import { useMemo, useState } from 'react'
import { MessageCircle, Plus, Search, Users } from 'lucide-react'
import { SecretariaLayout } from '../../components/secretaria/SecretariaLayout'
import { Alert } from '../../components/ui/Alert'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Input'
import { Modal } from '../../components/ui/Modal'
import { useAuth } from '../../contexts/AuthContext'
import { createCliente, getTenantClients } from '../../utils/mockDb'
import { formatBRL, formatDateShortBR, initials } from '../../utils/format'

function whatsappUrl(telefone: string) {
  const tel = telefone.replace(/\D/g, '')
  return `https://wa.me/55${tel}`
}

export function SecretariaClientes() {
  const { session } = useAuth()
  const tenantId = session?.user.tenant_id ?? ''
  const [search, setSearch] = useState('')
  const [refreshKey, setRefreshKey] = useState(0)
  const [modalOpen, setModalOpen] = useState(false)
  const [nome, setNome] = useState('')
  const [telefone, setTelefone] = useState('')
  const [email, setEmail] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [loading, setLoading] = useState(false)

  const clientes = useMemo(() => {
    void refreshKey
    const q = search.trim().toLowerCase()
    return getTenantClients(tenantId).filter(
      (c) =>
        !q ||
        c.nome.toLowerCase().includes(q) ||
        c.telefone.includes(q) ||
        (c.email?.toLowerCase().includes(q) ?? false),
    )
  }, [tenantId, search, refreshKey])

  const totalAtivos = clientes.filter((c) => c.ativo).length

  const resetForm = () => {
    setNome('')
    setTelefone('')
    setEmail('')
    setError('')
  }

  const salvarCliente = async () => {
    setError('')
    setLoading(true)
    try {
      if (!nome.trim()) throw new Error('Informe o nome do cliente.')
      if (telefone.replace(/\D/g, '').length < 10) {
        throw new Error('Informe um telefone válido.')
      }
      await createCliente(tenantId, {
        nome: nome.trim(),
        telefone,
        email: email.trim() || undefined,
      })
      setSuccess('Cliente cadastrado com sucesso.')
      setModalOpen(false)
      resetForm()
      setRefreshKey((k) => k + 1)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erro ao cadastrar')
    } finally {
      setLoading(false)
    }
  }

  return (
    <SecretariaLayout
      searchPlaceholder="Buscar cliente por nome ou telefone…"
      searchValue={search}
      onSearchChange={setSearch}
      headerAction={
        <Button
          className="hidden bg-[#7d5141] hover:bg-[#996958] sm:inline-flex"
          onClick={() => {
            resetForm()
            setModalOpen(true)
          }}
        >
          <Plus className="h-4 w-4" />
          Novo cliente
        </Button>
      }
    >
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="font-display text-2xl font-semibold text-[#1a1c1c] sm:text-3xl">
            Clientes
          </h1>
          <p className="mt-1 text-sm text-[#514440]">
            Cadastre e consulte clientes para agendamentos e cobranças.
          </p>
        </div>
        <Button
          className="bg-[#7d5141] hover:bg-[#996958] sm:hidden"
          onClick={() => {
            resetForm()
            setModalOpen(true)
          }}
        >
          <Plus className="h-4 w-4" />
          Novo cliente
        </Button>
      </div>

      {success && (
        <Alert variant="info" className="mb-4" onDismiss={() => setSuccess('')}>
          {success}
        </Alert>
      )}

      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <div className="rounded-xl border border-[#e5d3c8]/30 bg-white/70 p-4 shadow-sm backdrop-blur-md">
          <div className="flex items-center justify-between">
            <p className="text-xs font-bold uppercase tracking-widest text-[#514440]">Total</p>
            <Users className="h-5 w-5 text-[#7d5141]/40" />
          </div>
          <p className="mt-2 font-display text-2xl font-bold text-[#7d5141]">{clientes.length}</p>
        </div>
        <div className="rounded-xl border border-[#e5d3c8]/30 bg-white/70 p-4 shadow-sm backdrop-blur-md">
          <p className="text-xs font-bold uppercase tracking-widest text-[#514440]">Ativos</p>
          <p className="mt-2 font-display text-2xl font-bold text-emerald-700">{totalAtivos}</p>
        </div>
        <div className="rounded-xl border border-[#e5d3c8]/30 bg-white/70 p-4 shadow-sm backdrop-blur-md">
          <p className="text-xs font-bold uppercase tracking-widest text-[#514440]">Receita acumulada</p>
          <p className="mt-2 font-display text-2xl font-bold text-[#695c53]">
            {formatBRL(clientes.reduce((s, c) => s + c.gasto_total, 0))}
          </p>
        </div>
      </div>

      <div className="overflow-hidden rounded-xl border border-[#e5d3c8]/30 bg-white shadow-sm">
        <div className="hidden border-b border-[#efdcd1]/30 bg-[#faf9f8] px-4 py-3 text-xs font-bold uppercase tracking-widest text-[#514440] sm:grid sm:grid-cols-[1fr_140px_1fr_100px_80px] sm:gap-4">
          <span>Cliente</span>
          <span>Telefone</span>
          <span>E-mail</span>
          <span>Última visita</span>
          <span className="text-right">Ações</span>
        </div>
        {clientes.length === 0 ? (
          <div className="px-4 py-12 text-center text-sm text-[#514440]">
            <Search className="mx-auto mb-2 h-8 w-8 text-[#7d5141]/30" />
            Nenhum cliente encontrado.
          </div>
        ) : (
          <ul className="divide-y divide-[#efdcd1]/30">
            {clientes.map((c) => (
              <li
                key={c.id}
                className="flex flex-col gap-2 px-4 py-4 sm:grid sm:grid-cols-[1fr_140px_1fr_100px_80px] sm:items-center sm:gap-4"
              >
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-[#f1dfd4] text-sm font-bold text-[#7d5141]">
                    {initials(c.nome)}
                  </div>
                  <div>
                    <p className="font-semibold text-[#1a1c1c]">{c.nome}</p>
                    <p className="text-xs text-[#514440] sm:hidden">{c.telefone}</p>
                  </div>
                </div>
                <p className="hidden text-sm text-[#514440] sm:block">{c.telefone}</p>
                <p className="truncate text-sm text-[#514440]">{c.email ?? '—'}</p>
                <p className="text-sm text-[#514440]">
                  {c.ultima_visita ? formatDateShortBR(c.ultima_visita) : '—'}
                </p>
                <div className="flex justify-end">
                  <a
                    href={whatsappUrl(c.telefone)}
                    target="_blank"
                    rel="noreferrer"
                    className="inline-flex items-center gap-1 rounded-lg px-2 py-1.5 text-sm text-[#7d5141] hover:bg-[#efdcd1]/40"
                    aria-label={`WhatsApp ${c.nome}`}
                  >
                    <MessageCircle className="h-4 w-4" />
                  </a>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>

      <Modal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        title="Novo cliente"
        maxWidthClass="max-w-md"
      >
        <div className="space-y-4">
          {error && <Alert variant="error">{error}</Alert>}
          <Input label="Nome completo" value={nome} onChange={(e) => setNome(e.target.value)} />
          <Input
            label="Telefone (WhatsApp)"
            value={telefone}
            onChange={(e) => setTelefone(e.target.value)}
            placeholder="(11) 99999-9999"
          />
          <Input
            label="E-mail (opcional)"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
          <div className="flex gap-2 border-t border-[#d6c2bd]/20 pt-4">
            <Button onClick={salvarCliente} loading={loading} className="flex-1">
              Cadastrar cliente
            </Button>
            <Button variant="ghost" onClick={() => setModalOpen(false)}>
              Cancelar
            </Button>
          </div>
        </div>
      </Modal>
    </SecretariaLayout>
  )
}
