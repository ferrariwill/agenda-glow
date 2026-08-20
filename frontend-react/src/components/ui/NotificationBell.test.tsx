import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import { NotificationBell } from './NotificationBell'

describe('NotificationBell', () => {
  it('não renderiza badge quando unreadCount é 0', () => {
    const html = renderToStaticMarkup(
      <NotificationBell items={[]} unreadCount={0} />,
    )
    expect(html).toContain('aria-label="Notificações"')
    expect(html).not.toContain('bg-red-500')
  })

  it('renderiza contador quando há não lidas', () => {
    const html = renderToStaticMarkup(
      <NotificationBell
        items={[
          {
            id: '1',
            title: 'Aviso',
            created_at: new Date().toISOString(),
            read_at: null,
          },
        ]}
        unreadCount={3}
      />,
    )
    expect(html).toContain('bg-red-500')
    expect(html).toContain('>3<')
    expect(html).toContain('aria-label="3 notificações não lidas"')
  })

  it('exibe 9+ acima de nove não lidas', () => {
    const html = renderToStaticMarkup(<NotificationBell items={[]} unreadCount={12} />)
    expect(html).toContain('>9+<')
  })
})
