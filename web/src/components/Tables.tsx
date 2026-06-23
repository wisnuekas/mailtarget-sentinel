import { statusColor, formatPct, formatDate, subAccountStatusColor } from '../api/sentinel'
import type { Alert, SubAccountMetrics, CompanyRiskSummary, SubAccountRow, DomainMetrics, CompanyRow, SendingIPMetrics } from '../api/sentinel'

export function SubAccountActions({
  subAccountId,
  status,
  onSuspend,
  onResume,
  onWarning,
  onKillSwitch,
  busy,
}: {
  subAccountId: number
  status?: string
  onSuspend?: (id: number) => void
  onResume?: (id: number) => void
  onWarning?: (id: number) => void
  onKillSwitch?: (id: number) => void
  busy?: number | null
}) {
  const isBusy = busy === subAccountId
  const suspended = status === 'Suspended'

  return (
    <div className="action-group">
      {onKillSwitch && (
        <button
          className="btn-killswitch"
          disabled={isBusy}
          onClick={() => onKillSwitch(subAccountId)}
          title="Resend anomaly alert with kill-switch link"
        >
          Kill Switch
        </button>
      )}
      {onWarning && (
        <button
          className="btn-warning"
          disabled={isBusy}
          onClick={() => onWarning(subAccountId)}
          title="Send warning email"
        >
          Warning
        </button>
      )}
      {!suspended && onSuspend && (
        <button
          className="btn-danger"
          disabled={isBusy}
          onClick={() => onSuspend(subAccountId)}
          title="Suspend sub-account"
        >
          Suspend
        </button>
      )}
      {(suspended || !status) && onResume && (
        <button
          className="btn-success"
          disabled={isBusy}
          onClick={() => onResume(subAccountId)}
          title="Resume sub-account"
        >
          Resume
        </button>
      )}
    </div>
  )
}

export function SubAccountTable({
  rows,
  onSuspend,
  onResume,
  onWarning,
  onKillSwitch,
  busy,
}: {
  rows: SubAccountRow[]
  onSuspend?: (id: number) => void
  onResume?: (id: number) => void
  onWarning?: (id: number) => void
  onKillSwitch?: (id: number) => void
  busy?: number | null
}) {
  const list = rows ?? []
  if (list.length === 0) {
    return <p className="empty">No sub-accounts found.</p>
  }

  const hasActions = onSuspend || onResume || onWarning || onKillSwitch

  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>ID</th>
            <th>Name</th>
            <th>Company</th>
            <th>Status</th>
            <th>Sent (5m)</th>
            <th>Bounce</th>
            <th>Spam</th>
            {hasActions && <th>Actions</th>}
          </tr>
        </thead>
        <tbody>
          {list.map((sa) => (
            <tr key={sa.id}>
              <td className="num">{sa.id}</td>
              <td>{sa.name || '—'}</td>
              <td>{sa.company_name || (sa.company_id ? `Co ${sa.company_id}` : '—')}</td>
              <td>
                <span className="badge" style={{ background: subAccountStatusColor(sa.status) }}>
                  {sa.status}
                </span>
              </td>
              <td className="num">{sa.metrics?.sent?.toLocaleString() ?? '—'}</td>
              <td className={`num ${sa.metrics && sa.metrics.bounce_rate_pct > 5 ? 'danger-text' : ''}`}>
                {sa.metrics ? formatPct(sa.metrics.bounce_rate_pct) : '—'}
              </td>
              <td className="num">{sa.metrics ? formatPct(sa.metrics.spam_rate_pct) : '—'}</td>
              {hasActions && (
                <td>
                  <SubAccountActions
                    subAccountId={sa.id}
                    status={sa.status}
                    onSuspend={onSuspend}
                    onResume={onResume}
                    onWarning={onWarning}
                    onKillSwitch={onKillSwitch}
                    busy={busy}
                  />
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export function StatCard({ label, value, hint, danger }: { label: string; value: string | number; hint?: string; danger?: boolean }) {
  return (
    <div className={`stat-card ${danger ? 'danger' : ''}`}>
      <span className="stat-label">{label}</span>
      <strong className="stat-value">{value}</strong>
      {hint && <small>{hint}</small>}
    </div>
  )
}

export function AlertTable({ alerts }: { alerts: Alert[] }) {
  const rows = alerts ?? []
  if (rows.length === 0) {
    return <p className="empty">No alerts yet.</p>
  }

  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Detected</th>
            <th>Company</th>
            <th>Sub-account</th>
            <th>Bounce</th>
            <th>Spam</th>
            <th>Sent</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((a) => (
            <tr key={a.alert_id}>
              <td>{formatDate(a.detected_at)}</td>
              <td>{a.company_id}</td>
              <td>{a.sub_account_id}</td>
              <td className="num danger-text">{formatPct(a.bounce_rate_pct)}</td>
              <td className="num">{formatPct(a.spam_rate_pct)}</td>
              <td className="num">{a.sent.toLocaleString()}</td>
              <td>
                <span className="badge" style={{ background: statusColor(a.status) }}>
                  {a.status}
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export function DomainRiskTable({ domains }: { domains: DomainMetrics[] }) {
  const rows = domains ?? []
  if (rows.length === 0) {
    return <p className="empty">No sending domains currently at risk.</p>
  }

  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Company</th>
            <th>Sending Domain</th>
            <th>Sending</th>
            <th>Blocked</th>
            <th>Sent</th>
            <th>Bounce Rate</th>
            <th>Spam Rate</th>
            <th>Delivery</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((d) => (
            <tr key={`${d.company_id}-${d.sending_domain}`}>
              <td>{d.company_id}</td>
              <td><strong>{d.sending_domain}</strong></td>
              <td>{d.is_sending ? 'Yes' : '—'}</td>
              <td>
                {d.is_blocked ? (
                  <span className="badge" style={{ background: '#dc2626' }}>Blocked</span>
                ) : '—'}
              </td>
              <td className="num">{d.sent.toLocaleString()}</td>
              <td className="num danger-text">{formatPct(d.bounce_rate_pct)}</td>
              <td className="num">{formatPct(d.spam_rate_pct)}</td>
              <td className="num">{formatPct(d.delivery_rate_pct)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export function SendingIPRiskTable({ sendingIPs }: { sendingIPs: SendingIPMetrics[] }) {
  const rows = sendingIPs ?? []
  if (rows.length === 0) {
    return <p className="empty">No sending IPs currently at risk.</p>
  }

  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Sending IP</th>
            <th>Companies</th>
            <th>Sent</th>
            <th>Bounce Rate</th>
            <th>Spam Rate</th>
            <th>Delivery</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((ip) => (
            <tr key={ip.sending_ip}>
              <td><strong>{ip.sending_ip}</strong></td>
              <td className="num">{ip.affected_companies}</td>
              <td className="num">{ip.sent.toLocaleString()}</td>
              <td className="num danger-text">{formatPct(ip.bounce_rate_pct)}</td>
              <td className="num">{formatPct(ip.spam_rate_pct)}</td>
              <td className="num">{formatPct(ip.delivery_rate_pct)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export function AtRiskTable({
  items,
  onSuspend,
  onResume,
  onWarning,
  busy,
}: {
  items: SubAccountMetrics[]
  onSuspend?: (subAccountId: number) => void
  onResume?: (subAccountId: number) => void
  onWarning?: (subAccountId: number) => void
  busy?: number | null
}) {
  const rows = items ?? []
  if (rows.length === 0) {
    return <p className="empty">No sub-accounts currently at risk.</p>
  }

  const hasActions = onSuspend || onResume || onWarning

  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Company</th>
            <th>Sub-account</th>
            <th>Sent</th>
            <th>Bounce Rate</th>
            <th>Spam Rate</th>
            <th>Delivery</th>
            {hasActions && <th>Actions</th>}
          </tr>
        </thead>
        <tbody>
          {rows.map((m) => (
            <tr key={`${m.company_id}-${m.sub_account_id}`}>
              <td>{m.company_name || m.company_id}</td>
              <td>
                {m.sub_account_name ? (
                  <>{m.sub_account_name} <small className="muted">({m.sub_account_id})</small></>
                ) : m.sub_account_id}
              </td>
              <td className="num">{m.sent.toLocaleString()}</td>
              <td className="num danger-text">{formatPct(m.bounce_rate_pct)}</td>
              <td className="num">{formatPct(m.spam_rate_pct)}</td>
              <td className="num">{formatPct(m.delivery_rate_pct)}</td>
              {hasActions && (
                <td>
                  <SubAccountActions
                    subAccountId={m.sub_account_id}
                    status={m.status}
                    onSuspend={onSuspend}
                    onResume={onResume}
                    onWarning={onWarning}
                    busy={busy}
                  />
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export function CompanySummaryTable({ summary }: { summary: CompanyRiskSummary[] }) {
  const rows = summary ?? []
  if (rows.length === 0) {
    return <p className="empty">No companies at risk.</p>
  }

  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Company</th>
            <th>Affected Accounts</th>
            <th>Total Sent</th>
            <th>Max Bounce</th>
            <th>Max Spam</th>
            <th>Worst Sub-account</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((s) => (
            <tr key={s.company_id}>
              <td>
                <strong>{s.company_name || `Co ${s.company_id}`}</strong>
                {s.company_name && <small className="muted"> ({s.company_id})</small>}
              </td>
              <td className="num">{s.affected_accounts}</td>
              <td className="num">{s.total_sent.toLocaleString()}</td>
              <td className="num danger-text">{formatPct(s.max_bounce_rate_pct)}</td>
              <td className="num">{formatPct(s.max_spam_rate_pct)}</td>
              <td>{s.worst_sub_account_id}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export function CompanyTable({ companies }: { companies: CompanyRow[] }) {
  const rows = companies ?? []
  if (rows.length === 0) {
    return <p className="empty">No companies found.</p>
  }

  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>ID</th>
            <th>Name</th>
            <th>Active</th>
            <th>At Risk</th>
            <th>Max Bounce</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((c) => (
            <tr key={c.id}>
              <td className="num">{c.id}</td>
              <td><strong>{c.name}</strong></td>
              <td>
                <span className="badge" style={{ background: c.active ? '#16a34a' : '#64748b' }}>
                  {c.active ? 'Active' : 'Inactive'}
                </span>
              </td>
              <td>
                {c.at_risk ? (
                  <span className="badge" style={{ background: '#dc2626' }}>At Risk</span>
                ) : '—'}
              </td>
              <td className={`num ${c.max_bounce_rate_pct && c.max_bounce_rate_pct > 5 ? 'danger-text' : ''}`}>
                {c.max_bounce_rate_pct != null ? formatPct(c.max_bounce_rate_pct) : '—'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export function WindowSelect({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  return (
    <select value={value} onChange={(e) => onChange(e.target.value)} className="select">
      <option value="5m">Last 5 minutes</option>
      <option value="15m">Last 15 minutes</option>
      <option value="1h">Last 1 hour</option>
    </select>
  )
}
