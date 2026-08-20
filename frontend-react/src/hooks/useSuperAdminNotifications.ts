import { useCallback, useState } from 'react'
import type { SuperAdminNotification } from '../types'

export type SuperAdminNotificationsState = {
  items: SuperAdminNotification[]
  unreadCount: number
  loading: boolean
  markRead: (id: string) => void
}

/**
 * Fonte de dados MVP do sino SuperAdmin.
 * Hoje retorna lista vazia; trocar o stub pelo fetch quando a API existir.
 */
export function useSuperAdminNotifications(): SuperAdminNotificationsState {
  const [items, setItems] = useState<SuperAdminNotification[]>([])
  const [loading] = useState(false)

  const unreadCount = items.reduce((n, item) => (item.read_at ? n : n + 1), 0)

  const markRead = useCallback((id: string) => {
    setItems((prev) =>
      prev.map((item) =>
        item.id === id && !item.read_at
          ? { ...item, read_at: new Date().toISOString() }
          : item,
      ),
    )
  }, [])

  return { items, unreadCount, loading, markRead }
}
