import { useState, type FormEvent } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { ErrorAlert } from '../components/ErrorAlert'

export function RegisterPage() {
  const { register, isAuthenticated } = useAuth()
  const navigate = useNavigate()

  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<unknown>(null)
  const [pending, setPending] = useState(false)

  if (isAuthenticated) return <Navigate to="/" replace />

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setPending(true)
    try {
      await register(email.trim(), password, name.trim())
      navigate('/', { replace: true })
    } catch (err) {
      setError(err)
    } finally {
      setPending(false)
    }
  }

  return (
    <div className="auth-page">
      <div className="auth-panel">
        <p className="auth-brand">Secunda</p>
        <h1>Регистрация</h1>
        <p className="muted">Создайте аккаунт и начните работу в команде.</p>
        <ErrorAlert error={error} />
        <form className="form" onSubmit={onSubmit}>
          <label>
            Имя
            <input
              type="text"
              autoComplete="name"
              required
              maxLength={100}
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </label>
          <label>
            Email
            <input
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </label>
          <label>
            Пароль
            <input
              type="password"
              autoComplete="new-password"
              required
              minLength={8}
              maxLength={72}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </label>
          <button type="submit" className="btn btn-primary" disabled={pending}>
            {pending ? 'Создаём…' : 'Создать аккаунт'}
          </button>
        </form>
        <p className="auth-switch">
          Уже есть аккаунт? <Link to="/login">Войти</Link>
        </p>
      </div>
    </div>
  )
}
