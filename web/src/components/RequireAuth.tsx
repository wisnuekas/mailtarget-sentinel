import { Navigate, useLocation } from 'react-router-dom'
import { isAuthenticated } from '../auth/session'

export function RequireAuth({ children }: { children: React.ReactNode }) {
  const location = useLocation()

  if (!isAuthenticated()) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }

  return <>{children}</>
}
