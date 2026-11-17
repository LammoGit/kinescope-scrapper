import { createContext, useContext, useEffect, useState, ReactNode } from 'react'
import { api, AUTH_EXPIRED_EVENT } from './api'
import type { PublicUser, UserRole } from './types'

function decodeJwt(token: string): any {
  try {
    const p = token.split('.')[1]
    const json = atob(p.replace(/-/g, '+').replace(/_/g, '/'))
    return JSON.parse(json)
  } catch { return {} }
}

function extractClaims(token: string): { userId: number | null; role: UserRole | null } {
  const c = decodeJwt(token)
  const userId = c.user_id ?? c.UserID ?? c.uid ?? c.id ?? c.sub ?? null
  const role   = c.user_role ?? c.UserRole ?? c.role ?? null
  return {
    userId: typeof userId === 'number' ? userId : (userId ? Number(userId) : null),
    role: role as UserRole,
  }
}

interface AuthValue {
  user: { id: number; role: UserRole } | null
  loading: boolean
  login: (email: string, password: string) => Promise<void>
  register: (username: string, email: string, password: string) => Promise<PublicUser>
  logout: () => void
  isAdmin: boolean
  isOperator: boolean
}

const AuthContext = createContext<AuthValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<{ id: number; role: UserRole } | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const t = localStorage.getItem('access_token')
    if (t) {
      const { userId, role } = extractClaims(t)
      if (userId && role) setUser({ id: userId, role })
    }
    setLoading(false)
  }, [])

  // Drop the session when the API layer reports that it cannot be restored
  // (e.g. the access token expired and the refresh token was rejected)
  useEffect(() => {
    const onExpired = () => setUser(null)
    window.addEventListener(AUTH_EXPIRED_EVENT, onExpired)
    return () => window.removeEventListener(AUTH_EXPIRED_EVENT, onExpired)
  }, [])

  const login = async (email: string, password: string) => {
    const { access_token, refresh_token } = await api.login(email, password)
    localStorage.setItem('access_token', access_token)
    localStorage.setItem('refresh_token', refresh_token)
    const { userId, role } = extractClaims(access_token)
    if (!userId || !role) throw new Error('Unable to parse token claims')
    setUser({ id: userId, role })
  }

  const register = async (username: string, email: string, password: string) => {
    return api.register(username, email, password)
  }

  const logout = () => {
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    setUser(null)
  }

  return (
    <AuthContext.Provider value={{
      user, loading, login, register, logout,
      isAdmin:    user?.role === 'administrator',
      isOperator: user?.role === 'operator' || user?.role === 'administrator',
    }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider')
  return ctx
}
