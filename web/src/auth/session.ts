const SESSION_KEY = 'sentinel_session'

export interface Session {
  username: string
  token: string
}

export function getSession(): Session | null {
  try {
    const raw = sessionStorage.getItem(SESSION_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as Session
    if (!parsed.username) return null
    return parsed
  } catch {
    return null
  }
}

export function setSession(session: Session): void {
  sessionStorage.setItem(SESSION_KEY, JSON.stringify(session))
}

export function clearSession(): void {
  sessionStorage.removeItem(SESSION_KEY)
}

export function isAuthenticated(): boolean {
  return getSession() !== null
}

export function getAdminToken(): string | undefined {
  const session = getSession()
  if (session?.token) return session.token
  return import.meta.env.VITE_SENTINEL_ADMIN_TOKEN as string | undefined
}
