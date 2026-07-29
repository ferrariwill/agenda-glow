import { useEffect, useState, type ReactNode } from 'react'
import { IS_MOCK } from '../lib/config'
import { useAuth } from '../contexts/AuthContext'
import { subscribeStore } from './store'
import { syncAdminBootstrap, syncTenantBootstrap } from './sync'

export function DataProvider({ children }: { children: ReactNode }) {
  const { session, isAuthenticated } = useAuth()
  const [ready, setReady] = useState(IS_MOCK)
  const [tick, setTick] = useState(0)

  useEffect(() => subscribeStore(() => setTick((t) => t + 1)), [])

  useEffect(() => {
    if (IS_MOCK) {
      setReady(true)
      return
    }
    if (!isAuthenticated || !session) {
      setReady(true)
      return
    }

    let cancelled = false
    setReady(false)
    ;(async () => {
      try {
        if (session.user.role === 'SUPER_ADMIN') {
          await syncAdminBootstrap()
        } else if (
          session.user.role === 'DONA' ||
          session.user.role === 'PROFISSIONAL' ||
          session.user.role === 'SECRETARIA'
        ) {
          await syncTenantBootstrap(session.user.role)
        }
      } finally {
        if (!cancelled) setReady(true)
      }
    })()

    return () => {
      cancelled = true
    }
  }, [isAuthenticated, session?.user.id, session?.user.role])

  void tick

  if (!ready) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-[#faf9f8]">
        <p className="text-sm text-[#514440]">Carregando dados do salão…</p>
      </div>
    )
  }

  return children
}
