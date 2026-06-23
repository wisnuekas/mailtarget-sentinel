import { useCallback, useEffect, useState } from 'react'
import { api, type SubAccountMetrics, type CompanyRiskSummary, type DomainMetrics, type SendingIPMetrics } from '../api/sentinel'
import { AtRiskTable, CompanySummaryTable, DomainRiskTable, SendingIPRiskTable, WindowSelect } from '../components/Tables'

const REFRESH_MS = 30_000

export function AtRiskPage() {
  const [window, setWindow] = useState('5m')
  const [items, setItems] = useState<SubAccountMetrics[]>([])
  const [summary, setSummary] = useState<CompanyRiskSummary[]>([])
  const [domains, setDomains] = useState<DomainMetrics[]>([])
  const [sendingIPs, setSendingIPs] = useState<SendingIPMetrics[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [actionMsg, setActionMsg] = useState('')
  const [busy, setBusy] = useState<number | null>(null)

  const load = useCallback(() => {
    setLoading(true)
    setError('')
    api.atRisk(window)
      .then((data) => {
        setItems(data.items ?? [])
        setSummary(data.summary ?? [])
        setDomains(data.domains ?? [])
        setSendingIPs(data.sending_ips ?? [])
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false))
  }, [window])

  useEffect(() => { load() }, [load])

  useEffect(() => {
    const id = setInterval(load, REFRESH_MS)
    return () => clearInterval(id)
  }, [load])

  const runAction = async (
    subAccountId: number,
    label: string,
    fn: () => Promise<{ status?: string; transmission_id?: string }>,
  ) => {
    if (!confirm(`${label} sub-account ${subAccountId}?`)) return
    setBusy(subAccountId)
    try {
      const res = await fn()
      const detail = res.status ?? res.transmission_id ?? 'done'
      setActionMsg(`${label} OK — ${subAccountId} (${detail})`)
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

  const isEmpty = !loading && !error && items.length === 0 && domains.length === 0 && sendingIPs.length === 0

  return (
    <div className="page">
      <header className="page-header">
        <div>
          <h1>At Risk</h1>
          <p>Live sub-accounts, sending domains, and sending IPs exceeding bounce/spam thresholds</p>
        </div>
        <WindowSelect value={window} onChange={setWindow} />
      </header>

      {actionMsg && <p className="info-banner">{actionMsg}</p>}
      {loading && items.length === 0 && domains.length === 0 && sendingIPs.length === 0 && (
        <p className="loading">Scanning ClickHouse…</p>
      )}
      {error && <p className="error">{error}</p>}

      {isEmpty && (
        <p className="info-banner">
          No at-risk data in the last {window}. Seed dev events:{' '}
          <code>make clickhouse-seed-287</code> then refresh (data expires after ~5 minutes).
        </p>
      )}

      {!error && (
        <>
          <section className="card">
            <h2>By Company ({summary.length})</h2>
            <CompanySummaryTable summary={summary} />
          </section>

          <section className="card">
            <h2>At-Risk Sending IPs ({sendingIPs.length})</h2>
            <SendingIPRiskTable sendingIPs={sendingIPs} />
          </section>

          <section className="card">
            <h2>At-Risk Sending Domains ({domains.length})</h2>
            <DomainRiskTable domains={domains} />
          </section>

          <section className="card">
            <h2>All Affected Sub-accounts ({items.length})</h2>
            <AtRiskTable
              items={items}
              onSuspend={handleSuspend}
              onResume={handleResume}
              onWarning={handleWarning}
              busy={busy}
            />
          </section>
        </>
      )}
    </div>
  )
}
