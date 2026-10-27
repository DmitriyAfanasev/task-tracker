import { Link, Outlet, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'

export function Layout() {
  const { user, logout, isAuthenticated } = useAuth()
  const navigate = useNavigate()

  function handleLogout() {
    logout()
    navigate('/login')
  }

  return (
    <div className="app-shell">
      <header className="topbar">
        <Link to="/" className="brand">
          <span className="brand-mark" aria-hidden />
          Secunda
        </Link>
        {isAuthenticated && user && (
          <div className="topbar-right">
            <span className="user-chip">
              <span className="user-name">{user.name}</span>
              <span className="user-email">{user.email}</span>
            </span>
            <button type="button" className="btn btn-ghost" onClick={handleLogout}>
              Выйти
            </button>
          </div>
        )}
      </header>
      <main className="main">
        <Outlet />
      </main>
    </div>
  )
}
