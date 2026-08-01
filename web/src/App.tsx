import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { AuthProvider } from './features/auth/AuthContext'
import { useAuth } from './features/auth/useAuth'
import { LoginPage } from './features/auth/LoginPage'
import { ThemeProvider } from './shared/ThemeContext'
import { ToastProvider } from './shared/ToastContext'
import { Layout } from './shared/Layout'
import { DashboardPage } from './features/dashboard/DashboardPage'
import { NewPatientPage } from './features/records/NewPatientPage'
import { PatientDetailPage } from './features/records/PatientDetailPage'
import { EncounterDetailPage } from './features/records/EncounterDetailPage'
import { ReferralsInboxPage } from './features/referrals/ReferralsInboxPage'
import { NewReferralPage } from './features/referrals/NewReferralPage'
import { ReferralDetailPage } from './features/referrals/ReferralDetailPage'
import { UsersListPage } from './features/authz/UsersListPage'
import { CreateUserPage } from './features/authz/CreateUserPage'
import { UserDetailPage } from './features/authz/UserDetailPage'
import { RolesListPage } from './features/authz/RolesListPage'
import { RoleFormPage } from './features/authz/RoleFormPage'
import { PatientsListPage } from './features/records/PatientsListPage'
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
          <Route path="patients" element={<PatientsListPage />} />
          <Route path="authentication/users" element={<UsersListPage />} />
          <Route path="authentication/users/new" element={<CreateUserPage />} />
          <Route path="authentication/users/:userID" element={<UserDetailPage />} />
          <Route path="authentication/roles" element={<RolesListPage />} />
          <Route path="authentication/roles/new" element={<RoleFormPage />} />
          <Route path="authentication/roles/:roleID" element={<RoleFormPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

function App() {
  return (
    <ThemeProvider>
      <ToastProvider>
        <AuthProvider>
          <AppShell />
        </AuthProvider>
      </ToastProvider>
    </ThemeProvider>
  )
}

export default App
