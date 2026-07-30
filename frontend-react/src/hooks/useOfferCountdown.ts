import { useEffect, useRef, useState } from 'react'

/** Segundos → mm:ss (nunca negativo). */
export function formatCountdown(seconds: number): string {
  const safe = Math.max(0, Math.floor(seconds))
  const mm = String(Math.floor(safe / 60)).padStart(2, '0')
  const ss = String(safe % 60).padStart(2, '0')
  return `${mm}:${ss}`
}

export function secondsUntil(expiresAt: string | undefined, now = Date.now()): number | null {
  if (!expiresAt) return null
  const target = new Date(expiresAt).getTime()
  if (Number.isNaN(target)) return null
  return Math.max(0, Math.ceil((target - now) / 1000))
}

interface OfferCountdown {
  secondsRemaining: number | null
  label: string
  expired: boolean
}

/**
 * Contador regressivo derivado de `expires_at` (relógio do servidor).
 * Ao zerar chama `onZero` uma única vez — quem decide o estado final é o GET,
 * nunca o relógio do browser.
 */
export function useOfferCountdown(
  expiresAt: string | undefined,
  onZero?: () => void,
): OfferCountdown {
  const [secondsRemaining, setSecondsRemaining] = useState(() => secondsUntil(expiresAt))
  const onZeroRef = useRef(onZero)
  onZeroRef.current = onZero

  useEffect(() => {
    const initial = secondsUntil(expiresAt)
    setSecondsRemaining(initial)
    if (initial === null) return

    if (initial === 0) {
      onZeroRef.current?.()
      return
    }

    let fired = false
    const id = setInterval(() => {
      const next = secondsUntil(expiresAt)
      setSecondsRemaining(next)
      if (next === 0 && !fired) {
        fired = true
        clearInterval(id)
        onZeroRef.current?.()
      }
    }, 1000)

    return () => clearInterval(id)
  }, [expiresAt])

  return {
    secondsRemaining,
    label: secondsRemaining === null ? '' : formatCountdown(secondsRemaining),
    expired: secondsRemaining === 0,
  }
}
