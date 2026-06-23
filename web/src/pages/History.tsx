import { useCallback, useEffect, useState } from 'react'
import { api, type Alert } from '../api/sentinel'
import { AlertTable } from '../components/Tables'

export function HistoryPage() {
  const [alerts, setAlerts] = useState<Alert[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [companyId, setCompanyId] = useState('')
  const [status, setStatus] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = useCallback(() => {
    setLoading(true)
    const params: Record<string, string> = { page: String(page), limit: '20' }
    if (companyId) params.company_id = companyId
    if (status) params.status = status

    api.alerts(params)
      .then((data) => {
        setAlerts(data.alerts ?? [])
        setTotal(data.total)
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false))
  }, [page, companyId, status])

  useEffect(() => { load() }, [load])

  const totalPages = Math.max(1, Math.ceil(total / 20))

  return (
    <div className="page">
      <header className="page-header">
        <div>
          <h1>Alert History</h1>
          <p>Persistent log from SQLite — {total} total alerts</p>
        </div>
      </header>

      <div className="filters">
        <input
          placeholder="Filter company_id"
          value={companyId}
          onChange={(e) => { setCompanyId(e.target.value); setPage(1) }}
          className="input"
        />
        <select value={status} onChange={(e) => { setStatus(e.target.value); setPage(1) }} className="select">
          <option value="">All statuses</option>
          <option value="detected">detected</option>
          <option value="alert_sent">alert_sent</option>
          <option value="suspended">suspended</option>
          <option value="resolved">resolved</option>
        </select>
      </div>

      {loading && <p className="loading">Loading history…</p>}
      {error && <p className="error">{error}</p>}

      {!loading && !error && (
        <>
          <section className="card">
            <AlertTable alerts={alerts} />
          </section>
          <div className="pagination">
            <button disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>Previous</button>
            <span>Page {page} / {totalPages}</span>
            <button disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)}>Next</button>
          </div>
        </>
      )}
    </div>
  )
}
