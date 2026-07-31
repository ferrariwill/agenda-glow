import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import { SlotChip } from './SlotChip'

describe('SlotChip (DEV-115)', () => {
  it('expõe aria-pressed e min-h-touch-min', () => {
    const html = renderToStaticMarkup(
      <SlotChip value="09:00" label="09:00" selected={false} onSelect={() => {}} />,
    )
    expect(html).toContain('aria-pressed="false"')
    expect(html).toContain('min-h-touch-min')
    expect(html).toContain('touch-manipulation')
    expect(html).toContain('role="option"')
  })

  it('marca selecionado com aria-pressed e check visual', () => {
    const html = renderToStaticMarkup(
      <SlotChip value="10:30" label="10:30" selected onSelect={() => {}} />,
    )
    expect(html).toContain('aria-pressed="true"')
    expect(html).toContain('aria-selected="true"')
    expect(html).toContain('bg-[#7d5141]')
  })
})
