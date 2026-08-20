import { useCallback, useEffect, useState, type ReactNode } from 'react'
import { useParams } from 'react-router-dom'
import { Button } from '../components/ui/Button'
import { IS_MOCK } from '../lib/config'
import { syncPublicCatalog } from './sync'

type CatalogStatus = 'loading' | 'ready' | 'error'

export function PublicCatalogLoader({ children }: { children: ReactNode }) {
  const { slug } = useParams<{ slug: string }>()
  const [status, setStatus] = useState<CatalogStatus>(IS_MOCK || !slug ? 'ready' : 'loading')
  const [retryToken, setRetryToken] = useState(0)

  const retry = useCallback(() => {
    setRetryToken((n) => n + 1)
  }, [])

  useEffect(() => {
    if (IS_MOCK || !slug) {
      setStatus('ready')
      return
    }
    let cancelled = false
    setStatus('loading')
    syncPublicCatalog(slug)
      .then(() => {
        if (!cancelled) setStatus('ready')
      })
      .catch(() => {
        if (!cancelled) setStatus('error')
      })
    return () => {
      cancelled = true
    }
  }, [slug, retryToken])

  if (status === 'loading') {
    return (
      <div className="flex min-h-screen items-center justify-center bg-[#faf9f8]">
        <p className="text-sm text-[#514440]">Carregando catálogo…</p>
      </div>
    )
  }

  if (status === 'error') {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-4 bg-[#faf9f8] px-6 text-center">
        <p className="text-sm text-[#514440]">
          Não foi possível carregar o catálogo deste salão. Verifique a conexão e tente de novo.
        </p>
        <Button variant="secondary" onClick={retry}>
          Tentar de novo
        </Button>
      </div>
    )
  }

  return children
}
