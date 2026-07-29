import type { AuthSession, UserRole } from '../types'
import { apiFetch, ApiError } from '../lib/api'
import { parseJwtPayload } from '../lib/jwt'

interface LoginApiResponse {
  token: string
  role: string
  user?: {
    id: string
    email: string
    estabelecimento_id?: string
    profissional_id?: string
  }
}

const SESSION_KEY = 'agendaglow_session'

function roleFromApi(role: string): UserRole {
  const r = role.toUpperCase()
  if (r === 'SUPER_ADMIN' || r === 'DONA' || r === 'PROFISSIONAL' || r === 'SECRETARIA' || r === 'CLIENTE') {
    return r
  }
  throw new Error('Perfil não reconhecido')
}

function buildSession(token: string, role: string, user?: LoginApiResponse['user']): AuthSession {
  const claims = parseJwtPayload(token)
  const email = user?.email ?? claims?.email ?? ''
  const userRole = roleFromApi(role || claims?.role || '')

  return {
    token,
    user: {
      id: user?.id ?? claims?.user_id ?? '',
      email,
      role: userRole,
      tenant_id: user?.estabelecimento_id ?? claims?.estabelecimento_id,
      profissional_id: user?.profissional_id ?? claims?.profissional_id,
      nome: email.split('@')[0] ?? 'Usuário',
    },
  }
}

export async function loginWithApi(
  email: string,
  password: string,
  expectedRole?: UserRole,
): Promise<AuthSession> {
  const data = await apiFetch<LoginApiResponse>('/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email: email.trim(), password }),
  })

  const session = buildSession(data.token, data.role, data.user)
  if (expectedRole && session.user.role !== expectedRole) {
    throw new Error('Perfil incorreto para esta tela de login')
  }
  localStorage.setItem(SESSION_KEY, JSON.stringify(session))
  return session
}

export function apiAuthErrorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    switch (err.code) {
      case 'invalid_credentials':
        return 'E-mail ou senha inválidos'
      case 'user_inactive':
        return 'Usuário inativo. Entre em contato com o suporte.'
      default:
        return err.message
    }
  }
  if (err instanceof Error) return err.message
  return 'Falha no login'
}
