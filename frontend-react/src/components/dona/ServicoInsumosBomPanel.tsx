import { useEffect, useImperativeHandle, useState, forwardRef } from 'react'
import { Trash2 } from 'lucide-react'
import type { ServicoBomItem, ServicoBomInput } from '../../types'
import { listServiceSupplies } from '../../utils/mockDb'
import { InsumoQuickSearch } from './InsumoQuickSearch'

export interface ServicoInsumosBomPanelHandle {
  getItens: () => ServicoBomInput[]
}

interface ServicoInsumosBomPanelProps {
  servicoId?: string
  className?: string
}

export const ServicoInsumosBomPanel = forwardRef<
  ServicoInsumosBomPanelHandle,
  ServicoInsumosBomPanelProps
>(function ServicoInsumosBomPanel({ servicoId, className }, ref) {
  const [itens, setItens] = useState<ServicoBomItem[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!servicoId) {
      setItens([])
      return
    }
    let cancelled = false
    void (async () => {
      setLoading(true)
      setError('')
      try {
        const list = await listServiceSupplies(servicoId)
        if (!cancelled) setItens(list)
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Falha ao carregar ficha técnica.')
          setItens([])
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [servicoId])

  useImperativeHandle(ref, () => ({
    getItens: () =>
      itens
        .filter((i) => i.quantidade_uso > 0)
        .map((i) => ({
          insumo_id: i.insumo_id,
          quantidade_uso: i.quantidade_uso,
        })),
  }))

  const excludeIds = itens.map((i) => i.insumo_id)

  return (
    <div className={className}>
      <h2 className="mb-4 border-b border-[#d6c2bd]/10 pb-2 font-semibold text-[#7d5141]">
        Ficha técnica (insumos)
      </h2>

      {error && <p className="mb-3 text-sm text-red-600">{error}</p>}
      {loading && <p className="mb-3 text-sm text-[#83746f]">Carregando ficha…</p>}

      <div className="mb-4">
        <InsumoQuickSearch
          excludeIds={excludeIds}
          onSelect={(insumo) => {
            setItens((prev) => [
              ...prev,
              {
                insumo_id: insumo.id,
                nome: insumo.nome,
                unidade: insumo.unidade,
                quantidade_uso: 1,
              },
            ])
          }}
        />
      </div>

      <div className="space-y-3">
        {itens.length === 0 && !loading ? (
          <p className="text-sm text-[#83746f]">
            Nenhum insumo na ficha. Busque e adicione para montar o BOM.
          </p>
        ) : (
          itens.map((item) => (
            <div key={item.insumo_id} className="flex items-center gap-2">
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium text-[#7d5141]">{item.nome}</p>
                <p className="text-xs text-[#83746f]">{item.unidade}</p>
              </div>
              <input
                type="number"
                min={0.01}
                step="any"
                value={item.quantidade_uso}
                onChange={(e) => {
                  const qty = Number(e.target.value)
                  setItens((prev) =>
                    prev.map((row) =>
                      row.insumo_id === item.insumo_id
                        ? { ...row, quantidade_uso: qty }
                        : row,
                    ),
                  )
                }}
                className="w-24 rounded-lg bg-[#f4f3f2] px-3 py-2 text-sm focus:ring-1 focus:ring-[#7d5141]"
                aria-label={`Quantidade de ${item.nome}`}
              />
              <button
                type="button"
                onClick={() =>
                  setItens((prev) => prev.filter((row) => row.insumo_id !== item.insumo_id))
                }
                className="rounded p-1.5 text-[#83746f] hover:bg-red-50 hover:text-red-600"
                aria-label={`Remover ${item.nome}`}
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
          ))
        )}
      </div>
    </div>
  )
})
