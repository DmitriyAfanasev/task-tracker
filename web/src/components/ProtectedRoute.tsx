import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'

export function ProtectedRoute({ children }: { children: ReactNode }) {
  const { isAuthenticated, ready } = useAuth()
  const location = useLocation()

  if (!ready) return null
  if (!isAuthenticated) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }
  return children
}
