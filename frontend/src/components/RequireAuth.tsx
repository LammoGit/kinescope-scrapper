import { Navigate } from 'react-router-dom'
import { useAuth } from '../auth'
import type { UserRole } from '../types'

export function RequireAuth({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth()
  if (loading) return <div className="loading">Loading...</div>
  if (!user) return <Navigate to="/login" replace />
  return <>{children}</>
}

export function RequireRole({ role, children }: { role: UserRole; children: React.ReactNode }) {
  const { user } = useAuth()
  if (!user) return <Navigate to="/login" replace />
  if (user.role !== role) return <div className="error">Forbidden: requires {role}</div>
  return <>{children}</>
}
