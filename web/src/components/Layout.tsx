import { Link, useLocation, useNavigate } from 'react-router-dom'
import { clearSession, getSession } from '../auth/session'

const links = [
  { to: '/', label: 'Overview' },
  { to: '/at-risk', label: 'At Risk' },
  { to: '/companies', label: 'Companies' },
  { to: '/sub-accounts', label: 'Sub-accounts' },
  { to: '/history', label: 'History' },
  { to: '/settings', label: 'Settings' },
]

export function Layout({ children }: { children: React.ReactNode }) {
  const location = useLocation()
  const navigate = useNavigate()
  const session = getSession()

  const logout = () => {
    clearSession()
    navigate('/login', { replace: true })
  }

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <span className="brand-icon">🛡</span>
          <div>
            <strong>Mailtarget</strong>
            <small>Sentinel</small>
          </div>
        </div>
        <nav>
          {links.map((l) => (
            <Link
              key={l.to}
              to={l.to}
              className={location.pathname === l.to ? 'nav-link active' : 'nav-link'}
            >
              {l.label}
            </Link>
          ))}
        </nav>
        <div className="sidebar-footer">
          {session && <small className="sidebar-user">Signed in as {session.username}</small>}
          <button type="button" className="btn-logout" onClick={logout}>
            Log out
          </button>
        </div>
      </aside>
      <main className="main">{children}</main>
    </div>
  )
}
