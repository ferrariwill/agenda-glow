import { ApiError } from '../lib/api'
import { API_BASE, IS_MOCK } from '../lib/config'
import type {
  PublicAppointmentManageResponse,
  PublicCancelAppointmentResponse,
} from '../types'

const mockCancelledTokens = new Set<string>()

async function publicFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    ...init,
    credentials: 'omit',
    headers: { 'Content-Type': 'application/json', ...init.headers },
  })
  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as {
      error?: string
      message?: string
    }
    throw new ApiError(
      body.message ?? body.error ?? response.statusText,
      response.status,
      body.error,
    )
  }
  return response.json() as Promise<T>
}

function mockManage(token: string): PublicAppointmentManageResponse {
  if (token === 'mock-expired') {
    throw new ApiError('management_token_expired', 410, 'management_token_expired')
  }
  if (!['mock-valid', 'mock-reason-required', 'mock-window-closed', 'mock-cancelled'].includes(token)) {
    throw new ApiError('appointment_not_found', 404, 'appointment_not_found')
  }

  const cancelled = token === 'mock-cancelled' || mockCancelledTokens.has(token)
  const windowClosed = token === 'mock-window-closed'
  return {
    appointment: {
      service: 'Corte e finalização',
      professional: 'Ana',
      starts_at: '2026-08-01T14:00:00-03:00',
      status: cancelled ? 'CANCELADO' : 'AGENDADO',
      customer_confirmation: cancelled ? 'CANCELADO_CLIENTE' : 'PENDENTE',
    },
    establishment: {
      name: 'Studio Glow',
      contact_phone: '5515999999999',
    },
    cancellation: {
      allowed: !cancelled && !windowClosed,
      reason_required: token === 'mock-reason-required',
      minimum_notice_hours: 2,
      denial_reason: cancelled
        ? 'appointment_not_cancellable'
        : windowClosed
          ? 'cancellation_window_closed'
          : null,
    },
  }
}

export async function getManage(token: string): Promise<PublicAppointmentManageResponse> {
  if (IS_MOCK) return mockManage(token)
  return publicFetch(`/api/v1/public/appointments/manage/${encodeURIComponent(token)}`)
}

export async function cancelManage(
  token: string,
  body: { motivo?: string },
): Promise<PublicCancelAppointmentResponse> {
  if (IS_MOCK) {
    const current = mockManage(token)
    if (token === 'mock-reason-required' && !body.motivo?.trim()) {
      throw new ApiError('reason_required', 400, 'reason_required')
    }
    if (current.cancellation.denial_reason === 'cancellation_window_closed') {
      throw new ApiError('cancellation_window_closed', 422, 'cancellation_window_closed')
    }
    if (!current.cancellation.allowed && current.appointment.status !== 'CANCELADO') {
      throw new ApiError('appointment_not_cancellable', 409, 'appointment_not_cancellable')
    }
    mockCancelledTokens.add(token)
    return {
      status: 'cancelled',
      appointment_status: 'CANCELADO',
      customer_confirmation: 'CANCELADO_CLIENTE',
      slot_released: true,
    }
  }

  return publicFetch(`/api/v1/public/appointments/manage/${encodeURIComponent(token)}/cancel`, {
    method: 'POST',
    body: JSON.stringify(body.motivo?.trim() ? { motivo: body.motivo.trim() } : {}),
  })
}
