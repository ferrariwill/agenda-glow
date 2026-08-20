import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import { TIME_PICKER_POPOVER_LAYER, TimePicker, TimePickerPopover } from './TimePicker'

describe('TimePicker (DEV-197)', () => {
  it('lista horários em passos de 30 min com hit area touch', () => {
    const html = renderToStaticMarkup(
      <TimePicker value="09:00" onChange={() => {}} stepMinutes={30} />,
    )
    expect(html).toContain('role="listbox"')
    expect(html).toContain('09:00')
    expect(html).toContain('09:30')
    expect(html).toContain('10:00')
    expect(html).toContain('min-h-touch-min')
    expect(html).toContain('bg-aura-primary')
  })

  it('respeita minTime e maxTime', () => {
    const html = renderToStaticMarkup(
      <TimePicker value="10:00" onChange={() => {}} minTime="10:00" maxTime="11:00" />,
    )
    expect(html).toContain('10:00')
    expect(html).toContain('10:30')
    expect(html).toContain('11:00')
    expect(html).not.toContain('09:30')
    expect(html).not.toContain('11:30')
  })
})

describe('TimePickerPopover portal (DEV-197)', () => {
  it('usa camada fixed z-[200] e retorna null quando fechado', () => {
    expect(TIME_PICKER_POPOVER_LAYER).toContain('fixed')
    expect(TIME_PICKER_POPOVER_LAYER).toContain('z-[200]')
    const closed = renderToStaticMarkup(
      <TimePickerPopover
        open={false}
        onClose={() => {}}
        anchorRef={{ current: null }}
        value="09:00"
        onChange={() => {}}
      />,
    )
    expect(closed).toBe('')
  })
})
