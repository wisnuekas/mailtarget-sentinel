import { useCallback, useEffect, useState } from 'react'
import { api, type SubAccountRow } from '../api/sentinel'
import { SubAccountTable } from '../components/Tables'

const PAGE_SIZE = 20
const REFRESH_MS = 30_000

export function SubAccountsPage() {
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [rows, setRows] = useState<SubAccountRow[]>([])
  const [count, setCount] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [actionMsg, setActionMsg] = useState('')
  const [busy, setBusy] = useState<number | null>(null)

  const load = useCallback(() => {
    setLoading(true)
    setError('')
    const params: Record<string, string> = {
      page: String(page),
      size: String(PAGE_SIZE),
      window: '5m',
    }
    if (search) params.search = search
    if (statusFilter) params.status = statusFilter

    api.subAccounts(params)
      .then((data) => {
        setRows(data.sub_accounts ?? [])
        setCount(data.count)
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false))
  }, [page, search, statusFilter])

  useEffect(() => { load() }, [load])

  useEffect(() => {
    const id = setInterval(load, REFRESH_MS)
    return () => clearInterval(id)
  }, [load])

  const runAction = async (
    subAccountId: number,
    label: string,
    fn: () => Promise<{ status?: string; subject?: string; transmission_id?: string }>,
  ) => {
    if (!confirm(`${label} sub-account ${subAccountId}?`)) return
    setBusy(subAccountId)
    setActionMsg('')
    try {
      const res = await fn()
      const detail = res.status ?? res.transmission_id ?? 'done'
      setActionMsg(`${label} OK — sub-account ${subAccountId} (${detail})`)
      load()
    } catch (e) {
      setActionMsg(e instanceof Error ? e.message : `${label} failed`)
    } finally {
      setBusy(null)
    }
  }

  const handleSuspend = (id: number) =>
    runAction(id, 'Suspend', () => api.manualOverride(id, 'suspend'))

  const handleResume = (id: number) =>
    runAction(id, 'Resume', () => api.manualOverride(id, 'resume'))

  const handleWarning = (id: number) =>
    runAction(id, 'Warning email', () => api.sendWarningEmail(id))

  const handleKillSwitch = (id: number) =>
    runAction(id, 'Kill switch email', () => api.sendKillSwitchEmail({ sub_account_id: id }))

  const totalPages = Math.max(1, Math.ceil(count / PAGE_SIZE))

  return (
    <div className="page">
      <header className="page-header">
        <div>
          <h1>Sub-accounts</h1>
          <p>
            Live list from PostgreSQL with ClickHouse metrics (super-user dashboard)
          </p>
        </div>
      </header>

      {actionMsg && (
        <p className={`info-banner ${actionMsg.includes('OK') ? 'success' : ''}`}>{actionMsg}</p>
      )}
      {error && <p className="error">{error}</p>}

      <div className="filters">
        <input
          className="input"
          placeholder="Search name…"
          value={search}
          onChange={(e) => { setSearch(e.target.value); setPage(1) }}
        />
        <select
          className="select"
          value={statusFilter}
          onChange={(e) => { setStatusFilter(e.target.value); setPage(1) }}
        >
          <option value="">All statuses</option>
          <option value="Active">Active</option>
          <option value="Suspended">Suspended</option>
          <option value="Terminated">Terminated</option>
        </select>
      </div>

      <section className="card">
        <h2>
          Accounts ({count})
          {loading && <span className="loading-inline"> · loading…</span>}
        </h2>
        <SubAccountTable
          rows={rows}
          onSuspend={handleSuspend}
          onResume={handleResume}
          onWarning={handleWarning}
          onKillSwitch={handleKillSwitch}
          busy={busy}
        />

        {count > PAGE_SIZE && (
          <div className="pagination">
            <button disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
              Previous
            </button>
            <span>Page {page} / {totalPages}</span>
            <button disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)}>
              Next
            </button>
          </div>
        )}
      </section>

      <section className="card">
        <h2>API actions</h2>
        <ul className="api-hints">
          <li><strong>Suspend / Resume</strong> — PostgreSQL <code>UPDATE sub_account.status</code> via <code>POST /manual-override</code></li>
          <li><strong>Warning email</strong> — <code>POST /layang/transmissions</code> via Sentinel <code>POST /sub-accounts/warning-email</code></li>
          <li><strong>Kill switch email</strong> — resend anomaly alert via <code>POST /sub-accounts/kill-switch-email</code></li>
        </ul>
      </section>
    </div>
  )
}
