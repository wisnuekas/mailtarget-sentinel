import { useEffect, useState } from 'react'
import { api, type Settings } from '../api/sentinel'

const DEFAULT_COMPANY_ID = 287

export function SettingsPage() {
  const [settings, setSettings] = useState<Settings | null>(null)
  const [form, setForm] = useState<Partial<Settings>>({ company_id: DEFAULT_COMPANY_ID })
  const [loading, setLoading] = useState(true)
  const [msg, setMsg] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    api.getSettings()
      .then((s) => {
        setSettings(s)
        setForm(s)
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false))
  }, [])

  const save = async () => {
    setMsg('')
    setError('')
    try {
      const companyId = Number(form.company_id)
      if (Number.isNaN(companyId) || companyId < 0) {
        setError('Company ID must be a non-negative number')
        return
      }
      const payload = { ...form, company_id: companyId }
      const updated = await api.updateSettings(payload)
      setSettings(updated)
      setForm(updated)
      setMsg('Settings saved to Redis')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Save failed')
    }
  }

  if (loading) return <p className="loading">Loading settings…</p>

  return (
    <div className="page">
      <header className="page-header">
        <div>
          <h1>Settings</h1>
          <p>Dynamic thresholds and company scope (stored in Redis)</p>
        </div>
      </header>

      {msg && <p className="info-banner success">{msg}</p>}
      {error && <p className="error">{error}</p>}

      <section className="card form-card">
        <label>
          Company ID
          <input
            type="text"
            inputMode="numeric"
            className="input"
            value={form.company_id ?? DEFAULT_COMPANY_ID}
            onChange={(e) => setForm({ ...form, company_id: Number(e.target.value) || 0 })}
          />
          <small className="field-hint">
            Default 287. Set to 0 for all companies (super-user).
          </small>
        </label>
        <label>
          Minimum volume (emails / window)
          <input
            type="number"
            className="input"
            value={form.min_volume ?? ''}
            onChange={(e) => setForm({ ...form, min_volume: Number(e.target.value) })}
          />
        </label>
        <label>
          Bounce rate threshold (%)
          <input
            type="number"
            step="0.1"
            className="input"
            value={form.bounce_rate_threshold_pct ?? ''}
            onChange={(e) => setForm({ ...form, bounce_rate_threshold_pct: Number(e.target.value) })}
          />
        </label>
        <label>
          Spam rate threshold (%)
          <input
            type="number"
            step="0.1"
            className="input"
            value={form.spam_rate_threshold_pct ?? ''}
            onChange={(e) => setForm({ ...form, spam_rate_threshold_pct: Number(e.target.value) })}
          />
        </label>
        <label>
          Alert cooldown (minutes)
          <input
            type="number"
            className="input"
            value={form.alert_cooldown_minutes ?? ''}
            onChange={(e) => setForm({ ...form, alert_cooldown_minutes: Number(e.target.value) })}
          />
        </label>
        <button className="btn-primary" onClick={save}>Save Settings</button>
      </section>

      {settings && (
        <section className="card">
          <h2>Current Values</h2>
          <pre className="code-block">{JSON.stringify(settings, null, 2)}</pre>
        </section>
      )}
    </div>
  )
}
