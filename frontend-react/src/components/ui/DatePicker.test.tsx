import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import { DATE_PICKER_POPOVER_LAYER, DatePicker, DatePickerPopover } from './DatePicker'

describe('DatePicker touch (DEV-114)', () => {
  it('preserva hit area 44×44 + gap-2 sem comprimir no viewport 360', () => {
    const html = renderToStaticMarkup(
      <DatePicker value="2026-07-15" onChange={() => {}} />,
    )
    expect(html).toContain('overflow-x-auto')
    expect(html).toContain('w-[382px]')
    expect(html).toContain('gap-2')
    expect(html).toContain('size-touch-min')
  })
})

describe('DatePickerPopover portal (DEV-208)', () => {
  it('usa camada fixed z-[200] e retorna null quando fechado', () => {
    expect(DATE_PICKER_POPOVER_LAYER).toContain('fixed')
    expect(DATE_PICKER_POPOVER_LAYER).toContain('z-[200]')
    const closed = renderToStaticMarkup(
      <DatePickerPopover
        open={false}
        onClose={() => {}}
        anchorRef={{ current: null }}
        value="2026-07-15"
        onChange={() => {}}
      />,
    )
    expect(closed).toBe('')
  })
})
