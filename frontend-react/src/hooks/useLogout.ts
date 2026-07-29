import { useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'

export function useLogout(redirectTo = '/login') {
  const { logout } = useAuth()
  const navigate = useNavigate()

  return useCallback(() => {
    logout()
    navigate(redirectTo, { replace: true })
  }, [logout, navigate, redirectTo])
}
