import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'

import { api, configureClient } from '../api/client'
import type { User } from '../api/types'

const TOKEN_KEY = 'secunda_token'
const USER_KEY = 'secunda_user'
const EXPIRES_KEY = 'secunda_expires'

interface AuthState {
  token: string | null
  user: User | null
  expiresAt: string | null
}

interface AuthContextValue extends AuthState {
  ready: boolean
  isAuthenticated: boolean
  login: (email: string, password: string) => Promise<void>
  register: (email: string, password: string, name: string) => Promise<void>
  logout: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

function loadState(): AuthState {
  try {
    const token = localStorage.getItem(TOKEN_KEY)
    const expiresAt = localStorage.getItem(EXPIRES_KEY)
    const raw = localStorage.getItem(USER_KEY)
    const user = raw ? (JSON.parse(raw) as User) : null
    if (!token || !user) {
      return { token: null, user: null, expiresAt: null }
    }
    if (expiresAt) {
      const exp = new Date(expiresAt).getTime()
      if (exp <= Date.now()) {
        localStorage.removeItem(TOKEN_KEY)
        localStorage.removeItem(USER_KEY)
        localStorage.removeItem(EXPIRES_KEY)
        return { token: null, user: null, expiresAt: null }
      }
    }
    return { token, user, expiresAt }
  } catch {
    return { token: null, user: null, expiresAt: null }
  }
}

function persist(state: AuthState) {
  if (!state.token || !state.user) {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
    localStorage.removeItem(EXPIRES_KEY)
    return
  }
  localStorage.setItem(TOKEN_KEY, state.token)
  localStorage.setItem(USER_KEY, JSON.stringify(state.user))
  if (state.expiresAt) localStorage.setItem(EXPIRES_KEY, state.expiresAt)
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>({
    token: null,
    user: null,
    expiresAt: null,
  })
  const [ready, setReady] = useState(false)

  const logout = useCallback(() => {
    setState({ token: null, user: null, expiresAt: null })
    persist({ token: null, user: null, expiresAt: null })
  }, [])

  useEffect(() => {
    setState(loadState())
    setReady(true)
  }, [])

  useEffect(() => {
    configureClient({
      getToken: () => localStorage.getItem(TOKEN_KEY),
      onUnauthorized: logout,
    })
  }, [logout])

  const applyAuth = useCallback(
    (token: string, expiresAt: string, user: User) => {
      const next = { token, expiresAt, user }
      setState(next)
      persist(next)
    },
    [],
  )

  const login = useCallback(
    async (email: string, password: string) => {
      const res = await api.login({ email, password })
      applyAuth(res.token, res.expires_at, res.user)
    },
    [applyAuth],
  )

  const register = useCallback(
    async (email: string, password: string, name: string) => {
      const res = await api.register({ email, password, name })
      applyAuth(res.token, res.expires_at, res.user)
    },
    [applyAuth],
  )

  const value = useMemo<AuthContextValue>(
    () => ({
      ...state,
      ready,
      isAuthenticated: Boolean(state.token && state.user),
      login,
      register,
      logout,
    }),
    [state, ready, login, register, logout],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
