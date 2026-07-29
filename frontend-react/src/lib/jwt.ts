export interface JwtPayload {
  user_id?: string
  email?: string
  role?: string
  estabelecimento_id?: string
  profissional_id?: string
  exp?: number
}

export function parseJwtPayload(token: string): JwtPayload | null {
  try {
    const part = token.split('.')[1]
    if (!part) return null
    const json = atob(part.replace(/-/g, '+').replace(/_/g, '/'))
    return JSON.parse(json) as JwtPayload
  } catch {
    return null
  }
}
