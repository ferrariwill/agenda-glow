import { useEffect, useRef } from 'react'

export function nextAgendaExpiryDelay(
  expiresAtValues: Array<string | null | undefined>,
  now = Date.now(),
): number | null {
  const delays = expiresAtValues
    .map((value) => (value ? new Date(value).getTime() - now : Number.NaN))
    .filter(Number.isFinite)

  if (delays.length === 0) return null
  return Math.max(0, Math.min(...delays))
}

/**
 * Um único timer por página de agenda. Cards só apresentam o prazo recebido.
 */
export function useEarlySlotAgendaExpiry(
  expiresAtValues: Array<string | null | undefined>,
  onExpire: () => void,
) {
  const onExpireRef = useRef(onExpire)
  const firedSignatureRef = useRef('')
  const signature = expiresAtValues.filter(Boolean).sort().join('|')

  useEffect(() => {
    onExpireRef.current = onExpire
  }, [onExpire])

  useEffect(() => {
    if (!signature) return
    const values = signature.split('|')
    const delay = nextAgendaExpiryDelay(values)
    if (delay === null || (delay === 0 && firedSignatureRef.current === signature)) return

    const timer = window.setTimeout(() => {
      firedSignatureRef.current = signature
      onExpireRef.current()
    }, delay)
    return () => window.clearTimeout(timer)
  }, [signature])
}
