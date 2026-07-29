import { isSubscriptionBlockedError } from '../lib/api'

export function phaseFromError(err: unknown): 'blocked' | 'error' {
  return isSubscriptionBlockedError(err) ? 'blocked' : 'error'
}
