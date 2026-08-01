import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { AuthProvider } from './features/auth/AuthContext'
import { useAuth } from './features/auth/useAuth'
import { LoginPage } from './features/auth/LoginPage'
import { ThemeProvider } from './shared/ThemeContext'
import { Layout } from './shared/Layout'
import { DashboardPage } from './features/dashboard/DashboardPage'
import { NewPatientPage } from './features/records/NewPatientPage'
import { PatientDetailPage } from './features/records/PatientDetailPage'
import { EncounterDetailPage } from './features/records/EncounterDetailPage'
import { ReferralsInboxPage } from './features/referrals/ReferralsInboxPage'
import { NewReferralPage } from './features/referrals/NewReferralPage'
import { ReferralDetailPage } from './features/referrals/ReferralDetailPage'
import './App.css'

function AppShell() {
  const { user, loading } = useAuth()

  if (loading) {
    return null
  }

  if (!user) {
    return <LoginPage />
  }

  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route index element={<DashboardPage />} />
          <Route path="patients/new" element={<NewPatientPage />} />
          <Route path="patients/:patientId" element={<PatientDetailPage />} />
          <Route path="encounters/:encounterId" element={<EncounterDetailPage />} />
          <Route path="referrals" element={<ReferralsInboxPage />} />
          <Route path="referrals/new" element={<NewReferralPage />} />
          <Route path="referrals/:referralId" element={<ReferralDetailPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <AppShell />
      </AuthProvider>
    </ThemeProvider>
  )
}

export default App
