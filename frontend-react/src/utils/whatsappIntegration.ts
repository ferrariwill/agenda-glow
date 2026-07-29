import { IS_MOCK } from '../lib/config'
import { getDb, persistDb } from './mockDb'

export type WhatsAppIntegrationStatus = 'DESCONECTADO' | 'PENDENTE' | 'CONECTADO'

export type WhatsAppIntegrationView = {
  estabelecimento_id: string
  status: WhatsAppIntegrationStatus
  state: string
  signup_url: string
  waba_id?: string | null
  phone_number_id?: string | null
  connected_at?: string | null
}

/** URL oficial do Embedded Signup (Meta). O state do salão é concatenado no final. */
export const META_WHATSAPP_EMBEDDED_SIGNUP_BASE =
  (import.meta.env.VITE_META_WHATSAPP_EMBEDDED_SIGNUP_URL as string | undefined)?.trim() ||
  'https://business.facebook.com/messaging/whatsapp/onboard/?app_id=1743419366842637&config_id=3724327417718926&extras=%7B%22version%22%3A%22v4%22%2C%22sessionInfoVersion%22%3A%223%22%2C%22featureType%22%3A%22whatsapp_business_app_onboarding%22%7D&redirect_uri=https%3A%2F%2Fpreliminary-entrepreneur-cad-denver.trycloudflare.com%2Fmeta%2Fembedded-signup%2Fcallback'

const STATE_PREFIX = 'beleza'

export function buildWhatsAppState(tenantId: string): string {
  return `${STATE_PREFIX}_${tenantId}`
}

/**
 * URL final = base + '&state=beleza_' + idDoSalao
 * (igual ao pedido: base + '&state=beleza_' + usuarioLogado.idSalao)
 */
export function buildWhatsAppEmbeddedSignupURL(idSalao: string): string {
  let base = META_WHATSAPP_EMBEDDED_SIGNUP_BASE
  const ampState = base.indexOf('&state=')
  const qState = base.indexOf('?state=')
  if (ampState >= 0) base = base.slice(0, ampState)
  else if (qState >= 0) base = base.slice(0, qState)

  const sep = base.includes('?') ? '&' : '?'
  return `${base}${sep}state=beleza_${idSalao}`
}

export function getWhatsAppIntegration(tenantId: string): WhatsAppIntegrationView {
  const tenant = getDb().tenants.find((t) => t.id === tenantId)
  const status = tenant?.whatsapp_status ?? 'DESCONECTADO'
  return {
    estabelecimento_id: tenantId,
    status,
    state: buildWhatsAppState(tenantId),
    signup_url: buildWhatsAppEmbeddedSignupURL(tenantId),
    waba_id: tenant?.whatsapp_waba_id,
    phone_number_id: tenant?.whatsapp_phone_number_id,
    connected_at: tenant?.whatsapp_connected_at,
  }
}

export function markWhatsAppPending(tenantId: string): void {
  if (!IS_MOCK) return
  const db = getDb()
  const idx = db.tenants.findIndex((t) => t.id === tenantId)
  if (idx < 0) return
  if (db.tenants[idx].whatsapp_status === 'CONECTADO') return
  db.tenants[idx] = {
    ...db.tenants[idx],
    whatsapp_status: 'PENDENTE',
  }
  persistDb(db)
}

export function markWhatsAppConnected(
  tenantId: string,
  meta?: { waba_id?: string; phone_number_id?: string },
): void {
  if (!IS_MOCK) return
  const db = getDb()
  const idx = db.tenants.findIndex((t) => t.id === tenantId)
  if (idx < 0) return
  db.tenants[idx] = {
    ...db.tenants[idx],
    whatsapp_status: 'CONECTADO',
    whatsapp_waba_id: meta?.waba_id,
    whatsapp_phone_number_id: meta?.phone_number_id,
    whatsapp_connected_at: new Date().toISOString(),
  }
  persistDb(db)
}
