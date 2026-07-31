import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it, vi } from 'vitest'
import { ToastFeedback } from './ToastFeedback'

describe('ToastFeedback', () => {
  it('não renderiza quando message é null', () => {
    const html = renderToStaticMarkup(
      <ToastFeedback message={null} onDismiss={() => undefined} />,
    )
    expect(html).toBe('')
  })

  it('expõe aria-live polite e fica acima da BottomNav no mobile', () => {
    const html = renderToStaticMarkup(
      <ToastFeedback message="Salvo com sucesso." onDismiss={() => undefined} />,
    )
    expect(html).toContain('aria-live="polite"')
    expect(html).toContain('role="status"')
    expect(html).toContain('z-50')
    expect(html).toContain('bottom-[calc(5.5rem+env(safe-area-inset-bottom,0px))]')
    expect(html).toContain('lg:bottom-6')
    expect(html).toContain('Salvo com sucesso.')
  })

  it('usa hit area mínima no botão fechar', () => {
    const html = renderToStaticMarkup(
      <ToastFeedback message="Ok" onDismiss={vi.fn()} />,
    )
    expect(html).toContain('min-h-touch-min')
    expect(html).toContain('aria-label="Fechar"')
  })
})
