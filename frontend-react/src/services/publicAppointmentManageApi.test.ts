import { describe, expect, it } from 'vitest'
import { cancelManage, getManage } from './publicAppointmentManageApi'

describe('publicAppointmentManageApi mock contract', () => {
  it('returns the backend-shaped management payload', async () => {
    const response = await getManage('mock-valid')

    expect(response.appointment.customer_confirmation).toBe('PENDENTE')
    expect(response.cancellation).toMatchObject({
      allowed: true,
      reason_required: false,
      minimum_notice_hours: 2,
    })
  })

  it('propagates expired and unknown token status and code', async () => {
    await expect(getManage('mock-expired')).rejects.toMatchObject({
      status: 410,
      code: 'management_token_expired',
    })
    await expect(getManage('unknown')).rejects.toMatchObject({
      status: 404,
      code: 'appointment_not_found',
    })
  })

  it('requires a reason only when the API contract requests one', async () => {
    await expect(cancelManage('mock-reason-required', {})).rejects.toMatchObject({
      status: 400,
      code: 'reason_required',
    })

    await expect(
      cancelManage('mock-reason-required', { motivo: 'Imprevisto' }),
    ).resolves.toMatchObject({
      appointment_status: 'CANCELADO',
      customer_confirmation: 'CANCELADO_CLIENTE',
      slot_released: true,
    })
  })

  it('keeps cancellation-window decisions in the API layer', async () => {
    await expect(
      cancelManage('mock-window-closed', { motivo: 'Imprevisto' }),
    ).rejects.toMatchObject({
      status: 422,
      code: 'cancellation_window_closed',
    })
  })
})
