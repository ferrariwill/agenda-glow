import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Lock, Mail } from 'lucide-react'
import { Alert } from '../../components/ui/Alert'
import { Button } from '../../components/ui/Button'
import { useAuth } from '../../contexts/AuthContext'

const REMEMBER_KEY = 'agendaglow_remember_email'

function loadRememberedEmail(): string {
  try {
    return localStorage.getItem(REMEMBER_KEY) ?? ''
  } catch {
    return ''
  }
}

export function LoginForm() {
  const navigate = useNavigate()
  const { login } = useAuth()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [remember, setRemember] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    const saved = loadRememberedEmail()
    if (saved) {
      setEmail(saved)
      setRemember(true)
    }
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      if (remember) {
        localStorage.setItem(REMEMBER_KEY, email.trim())
      } else {
        localStorage.removeItem(REMEMBER_KEY)
      }
      const redirect = await login(email, password)
      navigate(redirect)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Falha no login')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="relative flex min-h-screen items-center justify-center p-4 sm:p-6">
      {/* Fundo — interior luxuoso com blur */}
      <div
        className="absolute inset-0 bg-cover bg-center bg-no-repeat"
        style={{
          backgroundImage:
            "url('https://images.unsplash.com/photo-1560066984-138dadb4c035?auto=format&fit=crop&w=1920&q=80')",
        }}
        aria-hidden
      />
      <div className="absolute inset-0 bg-aura-anthracite/25 backdrop-blur-sm" aria-hidden />

      {/* Card central */}
      <div className="relative z-10 w-full max-w-[420px] rounded-xl bg-[#faf9f8] px-6 py-8 shadow-2xl sm:px-10 sm:py-12">
        <header className="mb-8 text-center">
          <h1 className="font-display text-[2rem] font-semibold tracking-tight text-aura-anthracite sm:text-[2.25rem]">
            AgendaGlow
          </h1>
          <p className="mt-2 text-sm text-aura-muted">Bem-vinda de volta</p>
        </header>

        <form onSubmit={handleSubmit} className="space-y-5">
          {error && <Alert variant="error">{error}</Alert>}

          <div className="space-y-1.5">
            <label htmlFor="email" className="block text-sm font-medium text-aura-anthracite">
              E-mail
            </label>
            <div className="relative">
              <Mail className="pointer-events-none absolute left-3.5 top-1/2 h-[18px] w-[18px] -translate-y-1/2 text-aura-muted/70" />
              <input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="seu@email.com"
                required
                autoComplete="email"
                className="w-full rounded-lg border border-aura-border bg-white py-3 pl-11 pr-4 text-sm text-aura-anthracite placeholder:text-aura-muted/60 focus:border-aura-primary focus:outline-none focus:ring-2 focus:ring-aura-primary/20"
              />
            </div>
          </div>

          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <label htmlFor="password" className="text-sm font-medium text-aura-anthracite">
                Senha
              </label>
              <button
                type="button"
                className="text-xs font-normal text-aura-muted transition-colors hover:text-aura-primary"
                onClick={() => setError('Recuperação de senha em breve. Contate o suporte.')}
              >
                Esqueci minha senha
              </button>
            </div>
            <div className="relative">
              <Lock className="pointer-events-none absolute left-3.5 top-1/2 h-[18px] w-[18px] -translate-y-1/2 text-aura-muted/70" />
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                required
                autoComplete="current-password"
                className="w-full rounded-lg border border-aura-border bg-white py-3 pl-11 pr-4 text-sm text-aura-anthracite placeholder:text-aura-muted/60 focus:border-aura-primary focus:outline-none focus:ring-2 focus:ring-aura-primary/20"
              />
            </div>
          </div>

          <label className="flex cursor-pointer items-center gap-2.5">
            <input
              type="checkbox"
              checked={remember}
              onChange={(e) => setRemember(e.target.checked)}
              className="h-4 w-4 rounded border-aura-border text-aura-primary focus:ring-aura-primary/30"
            />
            <span className="text-sm text-aura-muted">Lembrar-me</span>
          </label>

          <Button
            type="submit"
            fullWidth
            loading={loading}
            className="mt-2 py-3.5 text-xs font-bold uppercase tracking-[0.12em]"
          >
            Entrar no sistema
          </Button>
        </form>

        <p className="mt-8 text-center text-sm text-aura-muted">
          Ainda não tem conta?{' '}
          <Link
            to="/login"
            className="font-semibold text-aura-primary transition-colors hover:text-aura-primary-dark"
            onClick={(e) => {
              e.preventDefault()
              setError('Cadastro disponível em breve. Fale com o suporte AgendaGlow.')
            }}
          >
            Cadastre-se
          </Link>
        </p>
      </div>
    </div>
  )
}
