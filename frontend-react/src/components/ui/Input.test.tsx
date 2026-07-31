import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import { Input } from './Input'
import { Button } from './Button'

describe('Input a11y (DEV-114)', () => {
  it('expõe aria-invalid e aria-describedby quando há error', () => {
    const html = renderToStaticMarkup(
      <Input id="nome" label="Nome" error="Obrigatório" />,
    )
    expect(html).toContain('aria-invalid="true"')
    expect(html).toContain('aria-describedby="nome-error"')
    expect(html).toContain('id="nome-error"')
    expect(html).toContain('Obrigatório')
  })

  it('não força aria-invalid sem error', () => {
    const html = renderToStaticMarkup(<Input id="email" label="Email" />)
    expect(html).not.toContain('aria-invalid')
    expect(html).not.toContain('aria-describedby')
  })

  it('usa text-control (16px) e min-h-touch-min no campo', () => {
    const html = renderToStaticMarkup(<Input id="tel" />)
    expect(html).toContain('text-control')
    expect(html).toContain('min-h-touch-min')
    expect(html).toContain('touch-manipulation')
  })
})

describe('Button touch (DEV-114)', () => {
  it('aplica piso de toque e touch-manipulation', () => {
    const html = renderToStaticMarkup(<Button>Salvar</Button>)
    expect(html).toContain('min-h-touch-min')
    expect(html).toContain('touch-manipulation')
  })
})
