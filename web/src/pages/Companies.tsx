import { useCallback, useEffect, useState } from 'react'
import { api, type CompanyRow } from '../api/sentinel'
import { CompanyTable, WindowSelect } from '../components/Tables'

const PAGE_SIZE = 20
const REFRESH_MS = 30_000

export function CompaniesPage() {
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [atRiskOnly, setAtRiskOnly] = useState(false)
  const [window, setWindow] = useState('5m')
  const [rows, setRows] = useState<CompanyRow[]>([])
  const [count, setCount] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = useCallback(() => {
    setLoading(true)
    setError('')
    const params: Record<string, string> = {
      page: String(page),
      size: String(PAGE_SIZE),
      window,
      include_risk: 'true',
    }
    if (search) params.search = search
    if (atRiskOnly) params.at_risk = 'true'

    api.companies(params)
      .then((data) => {
        setRows(data.companies ?? [])
        setCount(data.count)
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false))
  }, [page, search, atRiskOnly, window])

  useEffect(() => { load() }, [load])

  useEffect(() => {
    const id = setInterval(load, REFRESH_MS)
    return () => clearInterval(id)
  }, [load])

  const totalPages = Math.max(1, Math.ceil(count / PAGE_SIZE))

  return (
    <div className="page">
      <header className="page-header">
        <div>
          <h1>Companies</h1>
          <p>Browse all companies from PostgreSQL with at-risk status from ClickHouse</p>
        </div>
        <WindowSelect value={window} onChange={setWindow} />
      </header>

      {error && <p className="error">{error}</p>}

      <div className="filters">
        <input
          className="input"
          placeholder="Search company name…"
          value={search}
          onChange={(e) => { setSearch(e.target.value); setPage(1) }}
        />
        <label className="checkbox-label">
          <input
            type="checkbox"
            checked={atRiskOnly}
            onChange={(e) => { setAtRiskOnly(e.target.checked); setPage(1) }}
          />
          At risk only
        </label>
      </div>

      <section className="card">
        <h2>
          Companies ({count})
          {loading && <span className="loading-inline"> · loading…</span>}
        </h2>
        <CompanyTable companies={rows} />

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
    </div>
  )
}
