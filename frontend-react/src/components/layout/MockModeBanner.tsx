import { IS_MOCK } from '../../lib/config'

export function MockModeBanner() {
  if (!IS_MOCK) return null
  return (
    <div className="bg-amber-100 px-3 py-1.5 text-center text-xs text-amber-900">
      Modo protótipo (dados locais). Para usar o backend:{' '}
      <code className="rounded bg-amber-200/60 px-1">VITE_DATA_SOURCE=api</code> + API na porta 8081.
    </div>
  )
}
