export const PT_BR_LOCALE = 'pt-BR'

const DATE_OPTS: Intl.DateTimeFormatOptions = {
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
}

const TIME_OPTS: Intl.DateTimeFormatOptions = {
  hour: '2-digit',
  minute: '2-digit',
  hour12: false,
}

export function todayISO(): string {
  const now = new Date()
  const y = now.getFullYear()
  const m = String(now.getMonth() + 1).padStart(2, '0')
  const d = String(now.getDate()).padStart(2, '0')
  return `${y}-${m}-${d}`
}

/** Soma dias a uma data ISO (yyyy-mm-dd). */
export function addDaysISO(iso: string, days: number): string {
  const d = new Date(`${iso}T12:00:00`)
  d.setDate(d.getDate() + days)
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

export function formatBRL(v: number) {
  return v.toLocaleString(PT_BR_LOCALE, { style: 'currency', currency: 'BRL' })
}

/** R$ 14.200 → R$ 14,2k */
export function formatCompactBRL(v: number) {
  if (v >= 1000) {
    const k = v / 1000
    const label = k >= 10 ? k.toFixed(0) : k.toFixed(1).replace('.0', '')
    return `R$ ${label}k`
  }
  return formatBRL(v)
}

/** Converte texto digitado (180,50 · 180.50 · 1.234,56) em número. */
export function parseBRLInput(raw: string): number {
  let s = raw.trim().replace(/[R$\s]/g, '')
  if (!s) return NaN
  if (s.includes(',') && s.includes('.')) {
    s = s.replace(/\./g, '').replace(',', '.')
  } else if (s.includes(',')) {
    s = s.replace(',', '.')
  }
  return Number(s)
}

/** ISO → "14 mar. de 2024" */
export function formatDateShortBR(iso: string) {
  if (!iso) return ''
  const d = new Date(iso.includes('T') ? iso : `${iso}T12:00:00`)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleDateString(PT_BR_LOCALE, {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })
}

/** Telefone E.164 → (11) 98765-4321 */
export function formatPhoneBR(phone: string) {
  const d = phone.replace(/\D/g, '')
  const local = d.startsWith('55') && d.length >= 12 ? d.slice(2) : d
  if (local.length === 11) {
    return `(${local.slice(0, 2)}) ${local.slice(2, 7)}-${local.slice(7)}`
  }
  if (local.length === 10) {
    return `(${local.slice(0, 2)}) ${local.slice(2, 6)}-${local.slice(6)}`
  }
  return phone
}

/**
 * Máscara progressiva de celular/fixo BR enquanto digita.
 * Aceita colagem com prefixo 55 (≥12 dígitos) e limita a 11 dígitos locais.
 */
export function maskPhoneBRInput(raw: string): string {
  let digits = raw.replace(/\D/g, '')
  if (digits.startsWith('55') && digits.length >= 12) {
    digits = digits.slice(2)
  }
  digits = digits.slice(0, 11)

  if (digits.length === 0) return ''
  if (digits.length <= 2) return `(${digits}`
  if (digits.length <= 6) return `(${digits.slice(0, 2)}) ${digits.slice(2)}`
  if (digits.length <= 10) {
    return `(${digits.slice(0, 2)}) ${digits.slice(2, 6)}-${digits.slice(6)}`
  }
  return `(${digits.slice(0, 2)}) ${digits.slice(2, 7)}-${digits.slice(7)}`
}

/** Normaliza máscara ou raw para dígitos com país (55…) para a API. */
export function toWhatsAppDigits(maskedOrRaw: string): string {
  const digits = maskedOrRaw.replace(/\D/g, '')
  if (!digits) return ''
  if (digits.startsWith('55')) return digits
  return `55${digits}`
}

/** Hidrata o input mascarado a partir do valor persistido (dígitos/E.164). */
export function phoneDigitsToMaskInput(stored: string): string {
  return maskPhoneBRInput(stored)
}

export function formatBirthDateBR(iso: string) {
  if (!iso) return ''
  const d = new Date(`${iso}T12:00:00`)
  if (Number.isNaN(d.getTime())) return iso
  const age = Math.floor((Date.now() - d.getTime()) / (365.25 * 86400000))
  const label = d.toLocaleDateString(PT_BR_LOCALE, {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  })
  return `${label} (${age} anos)`
}

/** ISO (yyyy-mm-dd) → DD/MM/AAAA */
export function formatDateBR(iso: string) {
  if (!iso) return ''
  const d = new Date(iso.includes('T') ? iso : `${iso}T12:00:00`)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleDateString(PT_BR_LOCALE, DATE_OPTS)
}

/** HH:mm em 24h */
export function formatTimeBR(time: string) {
  if (!time) return ''
  const parts = time.split(':')
  if (parts.length < 2) return time
  const h = parts[0].padStart(2, '0')
  const m = parts[1].padStart(2, '0')
  return `${h}:${m}`
}

/** ISO + hora → DD/MM/AAAA às HH:mm */
export function formatDateTimeBR(iso: string, time?: string) {
  const datePart = formatDateBR(iso)
  if (!time) return datePart
  return `${datePart} às ${formatTimeBR(time)}`
}

/** Timestamp RFC3339 → DD/MM/AAAA às HH:mm (no fuso do navegador) */
export function formatTimestampBR(timestamp: string) {
  if (!timestamp) return ''
  const d = new Date(timestamp)
  if (Number.isNaN(d.getTime())) return timestamp
  return `${d.toLocaleDateString(PT_BR_LOCALE, DATE_OPTS)} às ${d.toLocaleTimeString(PT_BR_LOCALE, TIME_OPTS)}`
}

/** Timestamp RFC3339 → HH:mm */
export function formatTimestampTimeBR(timestamp: string) {
  if (!timestamp) return ''
  const d = new Date(timestamp)
  if (Number.isNaN(d.getTime())) return timestamp
  return d.toLocaleTimeString(PT_BR_LOCALE, TIME_OPTS)
}

/** segunda-feira, 21 de jun. */
export function formatWeekdayDateBR(iso: string) {
  const d = new Date(iso.includes('T') ? iso : `${iso}T12:00:00`)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleDateString(PT_BR_LOCALE, {
    weekday: 'long',
    day: 'numeric',
    month: 'short',
  })
}

/** Quarta, 24 de outubro */
export function formatWeekdayDateLongBR(iso: string) {
  const d = new Date(iso.includes('T') ? iso : `${iso}T12:00:00`)
  if (Number.isNaN(d.getTime())) return iso
  const raw = d.toLocaleDateString(PT_BR_LOCALE, {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
  })
  return raw.charAt(0).toUpperCase() + raw.slice(1)
}

/** Data e hora atuais em pt-BR */
export function formatNowBR() {
  return new Date().toLocaleString(PT_BR_LOCALE, {
    ...DATE_OPTS,
    ...TIME_OPTS,
  })
}

export function formatRelativeDate(iso: string) {
  const diff = Math.floor(
    (Date.now() - new Date(iso.includes('T') ? iso : `${iso}T12:00:00`).getTime()) / 86400000,
  )
  if (diff === 0) return 'hoje'
  if (diff === 1) return 'há 1 dia'
  if (diff < 30) return `há ${diff} dias`
  return formatDateBR(iso)
}

/** Timestamp relativo curto (notificações, feeds). */
export function formatRelativeShort(iso: string) {
  const ms = Date.now() - new Date(iso).getTime()
  if (Number.isNaN(ms)) return ''
  const minutes = Math.floor(ms / 60000)
  if (minutes < 1) return 'agora'
  if (minutes < 60) return `há ${minutes} min`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `há ${hours} h`
  const days = Math.floor(hours / 24)
  if (days === 1) return 'há 1 dia'
  if (days < 30) return `há ${days} dias`
  return formatDateBR(iso)
}

/** DD/MM/AAAA → ISO (yyyy-mm-dd) ou null */
export function parseDateBR(text: string): string | null {
  const cleaned = text.trim()
  const match = cleaned.match(/^(\d{1,2})\/(\d{1,2})\/(\d{4})$/)
  if (!match) return null

  const day = Number(match[1])
  const month = Number(match[2])
  const year = Number(match[3])
  if (month < 1 || month > 12 || day < 1 || day > 31) return null

  const iso = `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`
  const check = new Date(`${iso}T12:00:00`)
  if (
    Number.isNaN(check.getTime()) ||
    check.getFullYear() !== year ||
    check.getMonth() + 1 !== month ||
    check.getDate() !== day
  ) {
    return null
  }
  return iso
}

/** HH:mm (24h) ou null */
export function parseTimeBR(text: string): string | null {
  const cleaned = text.trim()
  const match = cleaned.match(/^(\d{1,2}):(\d{2})$/)
  if (!match) return null

  const hour = Number(match[1])
  const minute = Number(match[2])
  if (hour < 0 || hour > 23 || minute < 0 || minute > 59) return null

  return `${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}`
}

/** Máscara parcial enquanto digita data */
export function maskDateBRInput(raw: string): string {
  const digits = raw.replace(/\D/g, '').slice(0, 8)
  if (digits.length <= 2) return digits
  if (digits.length <= 4) return `${digits.slice(0, 2)}/${digits.slice(2)}`
  return `${digits.slice(0, 2)}/${digits.slice(2, 4)}/${digits.slice(4)}`
}

export const MONTH_NAMES_PT = [
  'Janeiro', 'Fevereiro', 'Março', 'Abril', 'Maio', 'Junho',
  'Julho', 'Agosto', 'Setembro', 'Outubro', 'Novembro', 'Dezembro',
]

/** Formata YYYY-MM como "Junho 2026". */
export function formatMonthYearBR(yearMonth: string): string {
  const [y, m] = yearMonth.split('-').map(Number)
  if (!y || !m) return yearMonth
  return `${MONTH_NAMES_PT[m - 1] ?? ''} ${y}`
}

/** Mês atual como YYYY-MM. */
export function currentYearMonth(): string {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
}

/** Últimos N meses (inclui o atual), do mais recente ao mais antigo. */
export function recentYearMonths(count: number): string[] {
  const out: string[] = []
  const now = new Date()
  for (let i = 0; i < count; i++) {
    const d = new Date(now.getFullYear(), now.getMonth() - i, 1)
    out.push(`${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`)
  }
  return out
}

/** ISO → "15 Abr 2024" */
export function formatTableDateBR(iso: string) {
  if (!iso) return ''
  const d = new Date(iso.includes('T') ? iso : `${iso}T12:00:00`)
  if (Number.isNaN(d.getTime())) return iso
  const month = d
    .toLocaleDateString(PT_BR_LOCALE, { month: 'short' })
    .replace('.', '')
  const cap = month.charAt(0).toUpperCase() + month.slice(1)
  return `${d.getDate()} ${cap} ${d.getFullYear()}`
}

export const WEEKDAYS_SHORT_PT = ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb']

/** Resumo legível do expediente semanal (ex.: Seg - Sex, 09:00 - 18:00). */
export function formatProfissionalDisponibilidade(
  expedientes: import('../types').Expediente[],
): string {
  if (!expedientes?.length) return 'Fora de Escala'

  const byDay = new Map<number, import('../types').Expediente>()
  for (const e of expedientes) {
    if (!byDay.has(e.dia_semana)) byDay.set(e.dia_semana, e)
  }
  const days = [...byDay.keys()].sort((a, b) => a - b)
  const first = byDay.get(days[0])!
  const sameHours = days.every((d) => {
    const e = byDay.get(d)!
    return (
      e.horario_entrada === first.horario_entrada &&
      e.horario_saida === first.horario_saida
    )
  })

  let dayLabel: string
  if (days.length === 5 && days.join(',') === '1,2,3,4,5') {
    dayLabel = 'Seg - Sex'
  } else if (days.length === 6 && days.join(',') === '1,2,3,4,5,6') {
    dayLabel = 'Seg - Sáb'
  } else if (days.length === 1) {
    dayLabel = WEEKDAYS_SHORT_PT[days[0]]
  } else {
    dayLabel = `${WEEKDAYS_SHORT_PT[days[0]]} - ${WEEKDAYS_SHORT_PT[days[days.length - 1]]}`
  }

  if (sameHours) {
    return `${dayLabel}, ${first.horario_entrada} - ${first.horario_saida}`
  }
  return days.map((d) => WEEKDAYS_SHORT_PT[d]).join(', ')
}

export function isoFromParts(year: number, month: number, day: number): string {
  return `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`
}

export function parseISOParts(iso: string): { year: number; month: number; day: number } {
  const [y, m, d] = iso.split('-').map(Number)
  return { year: y, month: m, day: d }
}

export interface CalendarDay {
  iso: string | null
  day: number | null
  inMonth: boolean
  isToday: boolean
}

/** Grade de 42 células (6 semanas) para o mês visível. */
export function getCalendarGrid(year: number, month: number): CalendarDay[] {
  const first = new Date(year, month - 1, 1)
  const startWeekday = first.getDay()
  const daysInMonth = new Date(year, month, 0).getDate()
  const today = todayISO()
  const cells: CalendarDay[] = []

  for (let i = 0; i < 42; i++) {
    const dayNum = i - startWeekday + 1
    if (dayNum < 1 || dayNum > daysInMonth) {
      cells.push({ iso: null, day: null, inMonth: false, isToday: false })
    } else {
      const iso = isoFromParts(year, month, dayNum)
      cells.push({
        iso,
        day: dayNum,
        inMonth: true,
        isToday: iso === today,
      })
    }
  }
  return cells
}

export function initials(name: string) {
  return name
    .split(' ')
    .slice(0, 2)
    .map((p) => p[0])
    .join('')
    .toUpperCase()
}
