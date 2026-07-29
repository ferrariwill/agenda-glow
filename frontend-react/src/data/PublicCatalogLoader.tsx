import { useEffect, useState, type ReactNode } from 'react'
import { useParams } from 'react-router-dom'
import { IS_MOCK } from '../lib/config'
import { syncPublicCatalog } from './sync'

export function PublicCatalogLoader({ children }: { children: ReactNode }) {
  const { slug } = useParams<{ slug: string }>()
  const [ready, setReady] = useState(IS_MOCK || !slug)

  useEffect(() => {
    if (IS_MOCK || !slug) {
      setReady(true)
      return
    }
    let cancelled = false
    setReady(false)
    syncPublicCatalog(slug)
      .catch(() => undefined)
      .finally(() => {
        if (!cancelled) setReady(true)
      })
    return () => {
      cancelled = true
    }
  }, [slug])

  if (!ready) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-[#faf9f8]">
        <p className="text-sm text-[#514440]">Carregando catálogo…</p>
      </div>
    )
  }

  return children
}
