import { useEffect, useState } from 'react'
import type { CategoriaServico } from '../../types'
import { listCategoriasServico } from '../../utils/mockDb'

interface ServicoCategoriaSelectProps {
  tenantId: string
  value: string
  onChange: (categoriaId: string) => void
  className?: string
  id?: string
}

export function ServicoCategoriaSelect({
  tenantId,
  value,
  onChange,
  className,
  id,
}: ServicoCategoriaSelectProps) {
  const [categorias, setCategorias] = useState<CategoriaServico[]>([])

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const list = await listCategoriasServico(tenantId)
        if (!cancelled) setCategorias(list)
      } catch {
        if (!cancelled) setCategorias([])
      }
    })()
    return () => {
      cancelled = true
    }
  }, [tenantId])

  return (
    <select
      id={id}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className={
        className ??
        'w-full rounded-lg border-none bg-[#f4f3f2] px-4 py-2.5 text-base focus:ring-1 focus:ring-[#7d5141]'
      }
    >
      <option value="">Sem categoria</option>
      {categorias.map((c) => (
        <option key={c.id} value={c.id}>
          {c.nome}
        </option>
      ))}
    </select>
  )
}
