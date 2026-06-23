import { useCallback, useEffect, useState } from 'react'
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from 'recharts'
import { api, type Alert, type CompanyRiskSummary } from '../api/sentinel'
import { StatCard, AlertTable, CompanySummaryTable } from '../components/Tables'

const REFRESH_MS = 30_000

export function OverviewPage() {
  const [recent, setRecent] = useState<Alert[]>([])
  const [stats, setStats] = useState<Record<string, number>>({})
  const [summary, setSummary] = useState<CompanyRiskSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = useCallback(() => {
    Promise.all([api.overview(), api.atRiskSummary('5m')])
      .then(([ov, atRisk]) => {
        setRecent(ov.recent_alerts ?? [])
        setStats(ov.stats_by_status ?? {})
        setSummary(atRisk.summary ?? [])
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => { load() }, [load])

  useEffect(() => {
    const id = setInterval(load, REFRESH_MS)
    return () => clearInterval(id)
  }, [load])

  if (loading && recent.length === 0 && summary.length === 0) {
    return <p className="loading">Loading dashboard…</p>
  }
  if (error) return <p className="error">{error}</p>

  const chartData = summary.map((s) => ({
    name: s.company_name || `Co ${s.company_id}`,
    bounce: s.max_bounce_rate_pct,
    spam: s.max_spam_rate_pct,
  }))

  const activeRisk = summary.length
  const totalAlerts = Object.values(stats).reduce((a, b) => a + b, 0)

  return (
    <div className="page">
      <header className="page-header">
        <div>
          <h1>Overview</h1>
          <p>Real-time domain reputation monitoring</p>
        </div>
      </header>

      <div className="stat-grid">
        <StatCard label="Companies at Risk" value={activeRisk} danger={activeRisk > 0} hint="Last 5 minutes" />
        <StatCard label="Total Alerts" value={totalAlerts} />
        <StatCard label="Suspended" value={stats.suspended ?? 0} />
        <StatCard label="Pending" value={(stats.detected ?? 0) + (stats.alert_sent ?? 0)} />
      </div>

      {activeRisk === 0 && (
        <p className="info-banner">No companies at risk in the last 5 minutes.</p>
      )}

      {chartData.length > 0 && (
        <section className="card">
          <h2>Bounce Rate by Company</h2>
          <ResponsiveContainer width="100%" height={260}>
            <BarChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#334155" />
              <XAxis dataKey="name" stroke="#94a3b8" />
              <YAxis stroke="#94a3b8" unit="%" />
              <Tooltip contentStyle={{ background: '#1e293b', border: 'none' }} />
              <Bar dataKey="bounce" fill="#ef4444" name="Bounce %" radius={[4, 4, 0, 0]} />
              <Bar dataKey="spam" fill="#f97316" name="Spam %" radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </section>
      )}

      <section className="card">
        <h2>Companies at Risk</h2>
        <CompanySummaryTable summary={summary} />
      </section>

      <section className="card">
        <h2>Recent Alerts</h2>
        <AlertTable alerts={recent} />
      </section>
    </div>
  )
}
