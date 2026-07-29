import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react'
import type { AuthSession, UserRole } from '../types'
import { IS_MOCK } from '../lib/config'
import { apiAuthErrorMessage, loginWithApi } from '../services/authApi'
import {
  ClienteAuthError,
  getDb,
  loginContaCliente,
  registerContaCliente,
  registerOrLoginCliente,
} from '../utils/mockDb'

const SESSION_KEY = 'agendaglow_session'

function encodeToken(payload: object): string {
  return btoa(JSON.stringify(payload))
}

function decodeToken(token: string): AuthSession | null {
  try {
    return JSON.parse(atob(token)) as AuthSession
  } catch {
    return null
  }
}

function loadSession(): AuthSession | null {
  const raw = localStorage.getItem(SESSION_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as AuthSession
  } catch {
    return decodeToken(raw)
  }
}

function buildSessionFromConta(conta: {
  id: string
  email: string
  nome: string
  telefone: string
}): AuthSession {
  return {
    token: encodeToken({ sub: conta.id, role: 'CLIENTE', exp: Date.now() + 86400000 * 30 }),
    user: {
      id: conta.id,
      email: conta.email,
      role: 'CLIENTE',
      nome: conta.nome,
      telefone: conta.telefone,
    },
  }
}

interface AuthContextValue {
  session: AuthSession | null
  login: (email: string, password: string, expectedRole?: UserRole) => Promise<string>
  loginCliente: (email: string, password: string, slug: string) => Promise<string>
  registerCliente: (
    data: { nome: string; telefone: string; email: string; password: string },
    slug: string,
  ) => Promise<string>
  registerOrLoginCliente: (
    data: { nome: string; telefone: string; email: string },
    slug: string,
  ) => Promise<string>
  logout: () => void
  refreshSession: () => void
  isAuthenticated: boolean
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<AuthSession | null>(() => loadSession())

  const login = useCallback(async (email: string, password: string, expectedRole?: UserRole) => {
    if (!IS_MOCK) {
      try {
        const authSession = await loginWithApi(email, password, expectedRole)
        setSession(authSession)
        return redirectForRole(authSession.user.role)
      } catch (err) {
        throw new Error(apiAuthErrorMessage(err))
      }
    }

    const db = getDb()
    const user = db.users.find(
      (u) => u.email.toLowerCase() === email.trim().toLowerCase() && u.password === password,
    )
    if (!user) throw new Error('E-mail ou senha inválidos')
    if (expectedRole && user.role !== expectedRole) {
      throw new Error('Perfil incorreto para esta tela de login')
    }

    const authSession: AuthSession = {
      token: encodeToken({ sub: user.id, role: user.role, exp: Date.now() + 86400000 }),
      user: {
        id: user.id,
        email: user.email,
        role: user.role,
        tenant_id: user.tenant_id,
        profissional_id: user.profissional_id,
        nome: user.nome,
        telefone: user.telefone,
      },
    }
    localStorage.setItem(SESSION_KEY, JSON.stringify(authSession))
    setSession(authSession)
    return redirectForRole(user.role)
  }, [])

  const loginCliente = useCallback(async (email: string, password: string, slug: string) => {
    const conta = loginContaCliente(email, password)
    const authSession = buildSessionFromConta(conta)
    localStorage.setItem(SESSION_KEY, JSON.stringify(authSession))
    setSession(authSession)
    return `/${slug}/agendar`
  }, [])

  const registerCliente = useCallback(
    async (
      data: { nome: string; telefone: string; email: string; password: string },
      slug: string,
    ) => {
      try {
        const conta = registerContaCliente(data)
        const authSession = buildSessionFromConta(conta)
        localStorage.setItem(SESSION_KEY, JSON.stringify(authSession))
        setSession(authSession)
        return `/${slug}/agendar`
      } catch (err) {
        if (err instanceof ClienteAuthError) throw err
        throw new Error('Erro ao cadastrar')
      }
    },
    [],
  )

  const registerOrLoginClienteFn = useCallback(
    async (data: { nome: string; telefone: string; email: string }, slug: string) => {
      try {
        const conta = registerOrLoginCliente(data)
        const authSession = buildSessionFromConta(conta)
        localStorage.setItem(SESSION_KEY, JSON.stringify(authSession))
        setSession(authSession)
        return `/${slug}/agendar`
      } catch (err) {
        if (err instanceof ClienteAuthError) throw err
        throw new Error('Erro ao cadastrar')
      }
    },
    [],
  )

  const logout = useCallback(() => {
    localStorage.removeItem(SESSION_KEY)
    setSession(null)
  }, [])

  const refreshSession = useCallback(() => {
    setSession((current) => {
      if (!current) return null
      if (current.user.role === 'CLIENTE') {
        const conta = getDb().contas_cliente.find((c) => c.id === current.user.id)
        if (!conta) return current
        const updated = buildSessionFromConta(conta)
        localStorage.setItem(SESSION_KEY, JSON.stringify(updated))
        return updated
      }
      const user = getDb().users.find((u) => u.id === current.user.id)
      if (!user) return current
      const updated: AuthSession = {
        ...current,
        user: {
          id: user.id,
          email: user.email,
          role: user.role,
          tenant_id: user.tenant_id,
          profissional_id: user.profissional_id,
          nome: user.nome,
          telefone: user.telefone,
        },
      }
      localStorage.setItem(SESSION_KEY, JSON.stringify(updated))
      return updated
    })
  }, [])

  const value = useMemo(
    () => ({
      session,
      login,
      loginCliente,
      registerCliente,
      registerOrLoginCliente: registerOrLoginClienteFn,
      logout,
      refreshSession,
      isAuthenticated: !!session,
    }),
    [session, login, loginCliente, registerCliente, registerOrLoginClienteFn, logout, refreshSession],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth deve ser usado dentro de AuthProvider')
  return ctx
}

export function redirectForRole(role: UserRole, slug?: string): string {
  switch (role) {
    case 'SUPER_ADMIN':
      return '/superadmin/dashboard'
    case 'DONA':
      return '/admin/dashboard'
    case 'PROFISSIONAL':
      return '/profissional/dashboard'
    case 'SECRETARIA':
      return '/secretaria/agenda'
    case 'CLIENTE':
      return slug ? `/${slug}/agendar` : '/login'
    default:
      return '/login'
  }
}
