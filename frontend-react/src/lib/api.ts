import { API_BASE } from './config'

export class ApiError extends Error {
  status: number
  code?: string

  constructor(message: string, status: number, code?: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

type ApiErrorListener = (err: ApiError) => void
const errorListeners = new Set<ApiErrorListener>()

export const SUBSCRIPTION_BLOCKED_CODE = 'subscription_expired_or_suspended'

export function onApiError(listener: ApiErrorListener): () => void {
  errorListeners.add(listener)
  return () => errorListeners.delete(listener)
}

export function isSubscriptionBlockedError(err: unknown): boolean {
  return (
    err instanceof ApiError &&
    err.status === 402 &&
    err.code === SUBSCRIPTION_BLOCKED_CODE
  )
}

export async function apiFetch<T = unknown>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const url = path.startsWith('http') ? path : `${API_BASE}${path}`
  const headers = new Headers(init.headers)
  // FormData precisa do boundary gerado pelo browser — não forçar JSON.
  if (init.body && !(init.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  const res = await fetch(url, {
    ...init,
    headers,
    credentials: 'include',
  })

  if (!res.ok) {
    const text = await res.text()
    let code: string | undefined
    let message = text || res.statusText
    try {
      const body = JSON.parse(text) as { error?: string; message?: string }
      code = body.error
      message = body.message ?? body.error ?? message
    } catch {
      /* mantém text */
    }
    const error = new ApiError(message, res.status, code)
    errorListeners.forEach((listener) => listener(error))
    throw error
  }

  if (res.status === 204) return undefined as T
  const ct = res.headers.get('content-type') ?? ''
  if (!ct.includes('application/json')) return undefined as T
  return res.json() as Promise<T>
}
