import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { IS_MOCK } from '../lib/config'
import { useAuth } from '../contexts/AuthContext'
import { isSubscriptionBlockedError, onApiError } from '../lib/api'
import type { BootstrapPhase } from '../utils/subscriptionGate'
import { BootstrapContext } from './BootstrapContext'
import { phaseFromError } from './bootstrapPhase'
import { subscribeStore } from './store'
import { syncAdminBootstrap, syncTenantBootstrap } from './sync'

function messageFromError(err: unknown): string {
  return err instanceof Error && err.message
    ? err.message
    : 'Tente novamente em alguns instantes.'
}

export function DataProvider({ children }: { children: ReactNode }) {
  const { session, isAuthenticated } = useAuth()
  const [ready, setReady] = useState(IS_MOCK)
  const [phase, setPhase] = useState<BootstrapPhase>('idle')
  const [errorMessage, setErrorMessage] = useState<string>()
  const [reloadKey, setReloadKey] = useState(0)
  const [tick, setTick] = useState(0)
  const retry = useCallback(() => setReloadKey((key) => key + 1), [])

  useEffect(() => subscribeStore(() => setTick((t) => t + 1)), [])

  /* eslint-disable react-hooks/set-state-in-effect -- o efeito controla o ciclo da carga assíncrona */
  useEffect(() => {
    const role = session?.user.role
    if (role !== 'DONA' && role !== 'PROFISSIONAL' && role !== 'SECRETARIA') return

    return onApiError((err) => {
      if (isSubscriptionBlockedError(err)) {
        setErrorMessage(undefined)
        setPhase('blocked')
      }
    })
  }, [session?.user.role])

  useEffect(() => {
    if (IS_MOCK) {
      setPhase('idle')
      setErrorMessage(undefined)
      setReady(true)
      return
    }
    if (!isAuthenticated || !session) {
      setPhase('idle')
      setErrorMessage(undefined)
      setReady(true)
      return
    }

    let cancelled = false
    setReady(false)
    setPhase('loading')
    setErrorMessage(undefined)
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
        if (!cancelled) setPhase('ready')
      } catch (err) {
        if (!cancelled) {
          const nextPhase = phaseFromError(err)
          setPhase(nextPhase)
          setErrorMessage(nextPhase === 'error' ? messageFromError(err) : undefined)
        }
      } finally {
        if (!cancelled) setReady(true)
      }
    })()

    return () => {
      cancelled = true
    }
  }, [isAuthenticated, session, reloadKey])
  /* eslint-enable react-hooks/set-state-in-effect */

  void tick
  const value = useMemo(
    () => ({ phase, errorMessage, retry }),
    [phase, errorMessage, retry],
  )

  if (!ready) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-[#faf9f8]">
        <p className="text-sm text-[#514440]">Carregando dados do salão…</p>
      </div>
    )
  }

  return <BootstrapContext.Provider value={value}>{children}</BootstrapContext.Provider>
}
