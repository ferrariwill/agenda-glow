import { useEffect, useRef, useState } from 'react'
import type { InsumoEstoque } from '../../types'
import { searchSupplies } from '../../utils/mockDb'

interface InsumoQuickSearchProps {
  excludeIds: string[]
  onSelect: (insumo: InsumoEstoque) => void
  disabled?: boolean
}

export function InsumoQuickSearch({
  excludeIds,
  onSelect,
  disabled = false,
}: InsumoQuickSearchProps) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<InsumoEstoque[]>([])
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const wrapRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    const onDoc = (e: MouseEvent) => {
      if (!wrapRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    return () => document.removeEventListener('mousedown', onDoc)
  }, [])

  const excludeKey = excludeIds.join(',')

  useEffect(() => {
    if (disabled) return
    let cancelled = false
    const timer = window.setTimeout(() => {
      void (async () => {
        setLoading(true)
        try {
          const list = await searchSupplies(query, 20)
          if (cancelled) return
          const exclude = new Set(excludeKey ? excludeKey.split(',') : [])
          setResults(list.filter((i) => i.ativo && !exclude.has(i.id)))
          // Só abre se o input estiver focado — evita dropdown no mount.
          if (document.activeElement === inputRef.current) setOpen(true)
        } catch {
          if (!cancelled) setResults([])
        } finally {
          if (!cancelled) setLoading(false)
        }
      })()
    }, 300)
    return () => {
      cancelled = true
      window.clearTimeout(timer)
    }
  }, [query, excludeKey, disabled])

  return (
    <div ref={wrapRef} className="relative">
      <input
        ref={inputRef}
        type="search"
        value={query}
        disabled={disabled}
        onChange={(e) => setQuery(e.target.value)}
        onFocus={() => setOpen(true)}
        placeholder="Buscar insumo…"
        className="w-full rounded-lg bg-[#f4f3f2] px-3 py-2 text-sm focus:ring-1 focus:ring-[#7d5141] disabled:opacity-50"
        aria-label="Busca rápida de insumos"
      />
      {open && !disabled && (
        <div className="absolute z-20 mt-1 max-h-48 w-full overflow-y-auto rounded-lg border border-[#e5d3c8]/40 bg-white shadow-lg">
          {loading && (
            <p className="px-3 py-2 text-xs text-[#83746f]">Buscando…</p>
          )}
          {!loading && results.length === 0 && (
            <p className="px-3 py-2 text-xs text-[#83746f]">Nenhum insumo encontrado.</p>
          )}
          {!loading &&
            results.map((item) => (
              <button
                key={item.id}
                type="button"
                className="flex w-full items-center justify-between px-3 py-2 text-left text-sm hover:bg-[#f4f3f2]"
                onClick={() => {
                  onSelect(item)
                  setQuery('')
                  setOpen(false)
                }}
              >
                <span className="font-medium text-[#7d5141]">{item.nome}</span>
                <span className="text-xs text-[#83746f]">{item.unidade}</span>
              </button>
            ))}
        </div>
      )}
    </div>
  )
}
