import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { Layout } from './components/Layout'
import { RequireAuth } from './components/RequireAuth'
import { LoginPage } from './pages/Login'
import { OverviewPage } from './pages/Overview'
import { AtRiskPage } from './pages/AtRisk'
import { HistoryPage } from './pages/History'
import { SubAccountsPage } from './pages/SubAccounts'
import { CompaniesPage } from './pages/Companies'
import { SettingsPage } from './pages/Settings'
import { isAuthenticated } from './auth/session'
import './App.css'

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route
          path="/login"
          element={isAuthenticated() ? <Navigate to="/" replace /> : <LoginPage />}
        />
        <Route
          path="/*"
          element={
            <RequireAuth>
              <Layout>
                <Routes>
                  <Route path="/" element={<OverviewPage />} />
                  <Route path="/at-risk" element={<AtRiskPage />} />
                  <Route path="/sub-accounts" element={<SubAccountsPage />} />
                  <Route path="/history" element={<HistoryPage />} />
                  <Route path="/companies" element={<CompaniesPage />} />
                  <Route path="/settings" element={<SettingsPage />} />
                </Routes>
              </Layout>
            </RequireAuth>
          }
        />
      </Routes>
    </BrowserRouter>
  )
}

export default App
