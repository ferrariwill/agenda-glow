import { useEffect, useRef } from 'react'
import { syncTenantBootstrap } from '../data/sync'
import { IS_MOCK } from '../lib/config'
import type { UserRole } from '../types'

const REFRESH_INTERVAL_MS = 45_000

export function useSilentTenantBootstrapRefresh(role?: UserRole) {
  const inFlight = useRef(false)

  useEffect(() => {
    if (IS_MOCK) return

    const refresh = async () => {
      if (document.visibilityState === 'hidden' || inFlight.current) return

      inFlight.current = true
      try {
        await syncTenantBootstrap(role)
      } catch {
        // Mantém os dados atuais; a próxima sincronização tenta novamente.
      } finally {
        inFlight.current = false
      }
    }

    const onFocus = () => {
      void refresh()
    }
    const onVisibilityChange = () => {
      if (document.visibilityState === 'visible') void refresh()
    }

    const interval = window.setInterval(() => void refresh(), REFRESH_INTERVAL_MS)
    window.addEventListener('focus', onFocus)
    document.addEventListener('visibilitychange', onVisibilityChange)

    return () => {
      window.clearInterval(interval)
      window.removeEventListener('focus', onFocus)
      document.removeEventListener('visibilitychange', onVisibilityChange)
    }
  }, [role])
}
