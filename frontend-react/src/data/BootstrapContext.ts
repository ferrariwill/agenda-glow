import { createContext, useContext } from 'react'
import type { BootstrapPhase } from '../utils/subscriptionGate'

export interface BootstrapState {
  phase: BootstrapPhase
  errorMessage?: string
  retry: () => void
}

export const BootstrapContext = createContext<BootstrapState | null>(null)

export function useBootstrapState(): BootstrapState {
  const context = useContext(BootstrapContext)
  if (!context) throw new Error('useBootstrapState deve ser usado dentro de DataProvider')
  return context
}
