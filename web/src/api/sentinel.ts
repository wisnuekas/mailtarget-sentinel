import { getAdminToken } from '../auth/session'

const BASE = '/api/v1/sentinel'

function authHeaders(): Record<string, string> {
  const token = getAdminToken()
  if (!token) return {}
  return { Authorization: `Bearer ${token}` }
}

function adminHeaders(): Record<string, string> {
  return { 'Content-Type': 'application/json', ...authHeaders() }
}

export interface SubAccountMetrics {
  company_id: number
  sub_account_id: number
  sent: number
  delivered: number
  bounced: number
  spam_bounced: number
  bounce_rate_pct: number
  delivery_rate_pct: number
  spam_rate_pct: number
  sub_account_name?: string
  company_name?: string
  status?: string
}

export interface CompanyRiskSummary {
  company_id: number
  affected_accounts: number
  total_sent: number
  max_bounce_rate_pct: number
  max_spam_rate_pct: number
  worst_sub_account_id: number
  company_name?: string
}

export interface Alert {
  id: number
  alert_id: string
  company_id: number
  sub_account_id: number
  sent: number
  bounced: number
  spam_bounced: number
  bounce_rate_pct: number
  spam_rate_pct: number
  status: string
  transmission_id?: string
  detected_at: string
  resolved_at?: string
}

export interface DomainMetrics {
  company_id: number
  sending_domain: string
  sent: number
  delivered: number
  bounced: number
  spam_bounced: number
  bounce_rate_pct: number
  delivery_rate_pct: number
  spam_rate_pct: number
  domain_id?: number
  is_sending?: boolean
  is_blocked?: boolean
}

export interface SendingIPMetrics {
  company_id: number
  sending_ip: string
  sent: number
  delivered: number
  bounced: number
  spam_bounced: number
  bounce_rate_pct: number
  delivery_rate_pct: number
  spam_rate_pct: number
  company_name?: string
}

export interface CompanyRow {
  id: number
  name: string
  active: boolean
  at_risk?: boolean
  max_bounce_rate_pct?: number
}

export interface SubAccountRow {
  id: number
  name: string
  status: string
  created_at: number
  company_id?: number
  company_name?: string
  ip_pool_name?: string
  metrics?: SubAccountMetrics
}

export interface Settings {
  min_volume: number
  bounce_rate_threshold_pct: number
  spam_rate_threshold_pct: number
  alert_cooldown_minutes: number
  company_id: number
}

interface Envelope<T> {
  success: boolean
  data?: T
  message?: string
  error?: string
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    ...init,
  })
  const body: Envelope<T> = await res.json()
  if (!res.ok || !body.success) {
    throw new Error(body.error || body.message || `HTTP ${res.status}`)
  }
  return body.data as T
}

export const api = {
  login: (username: string, password: string) =>
    request<{ username: string; token: string }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),

  overview: () =>
    request<{ recent_alerts: Alert[]; stats_by_status: Record<string, number> }>(
      '/alerts/overview',
    ),

  atRisk: (window = '5m') =>
    request<{
      window: string
      items: SubAccountMetrics[]
      summary: CompanyRiskSummary[]
      domains: DomainMetrics[]
      sending_ips: SendingIPMetrics[]
    }>(`/companies/at-risk?window=${window}`),

  atRiskSummary: (window = '5m') =>
    request<{ window: string; summary: CompanyRiskSummary[]; count: number }>(
      `/companies/at-risk/summary?window=${window}`,
    ),

  companies: (params: Record<string, string> = {}) => {
    const qs = new URLSearchParams(params).toString()
    return request<{
      page: number
      size: number
      count: number
      companies: CompanyRow[]
    }>(`/companies?${qs}`)
  },

  alerts: (params: Record<string, string> = {}) => {
    const qs = new URLSearchParams(params).toString()
    return request<{ alerts: Alert[]; total: number; page: number; limit: number }>(
      `/alerts?${qs}`,
    )
  },

  metrics: (window = '5m', companyId?: number) => {
    const qs = new URLSearchParams({ window })
    if (companyId) qs.set('company_id', String(companyId))
    return request<{ window: string; metrics: SubAccountMetrics[] }>(`/metrics?${qs}`)
  },

  getSettings: () => request<Settings>('/settings'),

  updateSettings: (partial: Partial<Settings>) =>
    request<Settings>('/settings', { method: 'POST', body: JSON.stringify(partial) }),

  manualOverride: (subAccountId: number, action: 'suspend' | 'resume') =>
    request<{ sub_account_id: number; status: string; action: string }>(
      '/manual-override',
      {
        method: 'POST',
        headers: adminHeaders(),
        body: JSON.stringify({ sub_account_id: subAccountId, action }),
      },
    ),

  subAccounts: (params: Record<string, string> = {}) => {
    const qs = new URLSearchParams(params).toString()
    return request<{
      page: number
      size: number
      count: number
      window: string
      sub_accounts: SubAccountRow[]
    }>(`/sub-accounts?${qs}`)
  },

  subAccount: (id: number, window = '5m') =>
    request<{ window: string; sub_account: SubAccountRow }>(
      `/sub-accounts/${id}?window=${window}`,
    ),

  sendWarningEmail: (subAccountId: number, window = '5m') =>
    request<{ sub_account_id: number; transmission_id: string; subject: string }>(
      '/sub-accounts/warning-email',
      {
        method: 'POST',
        headers: adminHeaders(),
        body: JSON.stringify({ sub_account_id: subAccountId, window }),
      },
    ),

  sendKillSwitchEmail: (opts: { sub_account_id?: number; alert_id?: string; window?: string }) =>
    request<{
      alert_id: string
      sub_account_id: number
      transmission_id: string
      subject: string
    }>('/sub-accounts/kill-switch-email', {
      method: 'POST',
      headers: adminHeaders(),
      body: JSON.stringify(opts),
    }),
}

export function statusColor(status: string): string {
  switch (status) {
    case 'suspended':
      return '#dc2626'
    case 'alert_sent':
      return '#ea580c'
    case 'detected':
      return '#ca8a04'
    case 'resolved':
      return '#16a34a'
    default:
      return '#64748b'
  }
}

export function formatPct(v: number): string {
  return `${v.toFixed(2)}%`
}

export function formatDate(v: string): string {
  return new Date(v).toLocaleString()
}

export function subAccountStatusColor(status: string): string {
  switch (status) {
    case 'Active':
      return '#16a34a'
    case 'Suspended':
      return '#dc2626'
    case 'Terminated':
      return '#64748b'
    default:
      return '#64748b'
  }
}
