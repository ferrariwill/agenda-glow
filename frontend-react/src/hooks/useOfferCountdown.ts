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

/**
 * Reconcile server seed with expires_at for display only.
 * Validity is never decided here — a zero value only triggers a GET.
 */
export function reconcileSecondsRemaining(
  expiresAt: string | undefined,
  secondsRemainingSeed: number | undefined,
  now = Date.now(),
): number | null {
  const fromExpires = secondsUntil(expiresAt, now)
  if (fromExpires === null) return null
  if (typeof secondsRemainingSeed === 'number' && Number.isFinite(secondsRemainingSeed)) {
    return Math.min(fromExpires, Math.max(0, Math.floor(secondsRemainingSeed)))
  }
  return fromExpires
}

/** After a zero-crossing GET, keep the gate closed while still at 0s. */
export function nextExpiryRefreshGate(reconciledSeconds: number | null): boolean {
  return reconciledSeconds === null || reconciledSeconds <= 0
}

interface OfferCountdown {
  secondsRemaining: number | null
  label: string
  expired: boolean
}

/**
 * Contador regressivo visual.
 * `seconds_remaining` é semente; `expires_at` reconcilia a exibição.
 * Ao zerar chama `onZero` uma única vez por ciclo positivo→zero.
 * Quem decide o estado final é o GET, nunca o relógio do browser.
 */
export function useOfferCountdown(
  expiresAt: string | undefined,
  onZero?: () => void,
  secondsRemainingSeed?: number,
): OfferCountdown {
  const [secondsRemaining, setSecondsRemaining] = useState<number | null>(() =>
    reconcileSecondsRemaining(expiresAt, secondsRemainingSeed),
  )
  const onZeroRef = useRef(onZero)
  const firedRef = useRef(false)

  useEffect(() => {
    onZeroRef.current = onZero
  }, [onZero])

  useEffect(() => {
    const initial = reconcileSecondsRemaining(expiresAt, secondsRemainingSeed)
    // Re-sincroniza o display quando a oferta (expires_at / seed) muda.
    // eslint-disable-next-line react-hooks/set-state-in-effect -- semente do relógio visual a partir do payload
    setSecondsRemaining(initial)
    firedRef.current = nextExpiryRefreshGate(initial)

    if (initial === null) return

    const fireIfNeeded = (next: number | null) => {
      if (next === 0 && !firedRef.current) {
        firedRef.current = true
        onZeroRef.current?.()
      }
    }

    if (initial === 0) {
      fireIfNeeded(initial)
      return
    }

    const tick = () => {
      const next = reconcileSecondsRemaining(expiresAt, undefined)
      setSecondsRemaining(next)
      fireIfNeeded(next)
    }

    const id = window.setInterval(tick, 1000)
    const onVisibility = () => {
      if (document.visibilityState !== 'visible') return
      tick()
    }
    document.addEventListener('visibilitychange', onVisibility)
    return () => {
      window.clearInterval(id)
      document.removeEventListener('visibilitychange', onVisibility)
    }
  }, [expiresAt, secondsRemainingSeed])

  return {
    secondsRemaining,
    label: secondsRemaining === null ? '' : formatCountdown(secondsRemaining),
    expired: secondsRemaining === 0,
  }
}
