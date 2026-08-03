import { useState } from 'react'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { AuthProvider } from './features/auth/AuthContext'
import { useAuth } from './features/auth/useAuth'
import { LoginPage } from './features/auth/LoginPage'
import { SignupPage } from './features/auth/SignupPage'
import { EmailVerifyPage } from './features/auth/EmailVerifyPage'
import { ForgotPasswordPage } from './features/auth/ForgotPasswordPage'
import { ResetPasswordPage } from './features/auth/ResetPasswordPage'
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
import { ConsultsInboxPage } from './features/consults/ConsultsInboxPage'
import { NewConsultPage } from './features/consults/NewConsultPage'
import { ConsultDetailPage } from './features/consults/ConsultDetailPage'
import { ConversationsListPage } from './features/messaging/ConversationsListPage'
import { NewConversationPage } from './features/messaging/NewConversationPage'
import { ConversationThreadPage } from './features/messaging/ConversationThreadPage'
import { UsersListPage } from './features/authz/UsersListPage'
import { CreateUserPage } from './features/authz/CreateUserPage'
import { UserDetailPage } from './features/authz/UserDetailPage'
import { RolesListPage } from './features/authz/RolesListPage'
import { RoleFormPage } from './features/authz/RoleFormPage'
import { PatientsListPage } from './features/records/PatientsListPage'
import { ProfileEditPage } from './features/profile/ProfileEditPage'
import { FacilitiesListPage } from './features/facilities/FacilitiesListPage'
import { NewFacilityPage } from './features/facilities/NewFacilityPage'
import { FacilityDetailPage } from './features/facilities/FacilityDetailPage'
import { ProviderProfilePage } from './features/providers/ProviderProfilePage'
import { SearchResultsPage } from './features/search/SearchResultsPage'
import { AppointmentsListPage } from './features/appointments/AppointmentsListPage'
import { BookAppointmentPage } from './features/appointments/BookAppointmentPage'
import { AppointmentDetailPage } from './features/appointments/AppointmentDetailPage'
import { AvailabilityEditorPage } from './features/appointments/AvailabilityEditorPage'
import { SettingsPage } from './features/settings/SettingsPage'
import { AuditLogPage } from './features/audit/AuditLogPage'
import { FeatureFlagsPage } from './features/authz/FeatureFlagsPage'
import './App.css'

// AuthGate renders the login/signup state-switch flow at "/" for a signed
// out visitor. Kept as its own component so it can sit alongside other
// public routes (verify-email, and eventually forgot/reset password)
// without those routes inheriting the login/signup toggle state.
function AuthGate() {
  const [authMode, setAuthMode] = useState<'login' | 'signup'>('login')

  return authMode === 'login' ? (
    <LoginPage onSwitchToSignup={() => setAuthMode('signup')} />
  ) : (
    <SignupPage onSwitchToLogin={() => setAuthMode('login')} />
  )
}

function AppShell() {
  const { user, loading } = useAuth()

  if (loading) {
    return null
  }

  return (
    <BrowserRouter>
      {!user ? (
        <Routes>
          {/* Public routes, reachable whether or not the visitor is signed in. */}
          <Route path="verify-email" element={<EmailVerifyPage />} />
          <Route path="forgot-password" element={<ForgotPasswordPage />} />
          <Route path="reset-password" element={<ResetPasswordPage />} />
          <Route path="*" element={<AuthGate />} />
        </Routes>
      ) : (
        <Routes>
          {/* Public routes must also be reachable for an already-signed-in
              user (e.g. testing a verification link in the same tab) — kept
              outside the Layout-wrapped group so they render standalone. */}
          <Route path="verify-email" element={<EmailVerifyPage />} />
          <Route path="forgot-password" element={<ForgotPasswordPage />} />
          <Route path="reset-password" element={<ResetPasswordPage />} />
          <Route element={<Layout />}>
            <Route index element={<DashboardPage />} />
            <Route path="patients/new" element={<NewPatientPage />} />
            <Route path="patients/:patientId" element={<PatientDetailPage />} />
            <Route path="encounters/:encounterId" element={<EncounterDetailPage />} />
            <Route path="referrals" element={<ReferralsInboxPage />} />
            <Route path="referrals/new" element={<NewReferralPage />} />
            <Route path="referrals/:referralId" element={<ReferralDetailPage />} />
            <Route path="consults" element={<ConsultsInboxPage />} />
            <Route path="consults/new" element={<NewConsultPage />} />
            <Route path="consults/:consultationId" element={<ConsultDetailPage />} />
            <Route path="messages" element={<ConversationsListPage />} />
            <Route path="messages/new" element={<NewConversationPage />} />
            <Route path="messages/:conversationId" element={<ConversationThreadPage />} />
            <Route path="patients" element={<PatientsListPage />} />
            <Route path="facilities" element={<FacilitiesListPage />} />
            <Route path="facilities/new" element={<NewFacilityPage />} />
            <Route path="facilities/:facilityId" element={<FacilityDetailPage />} />
            <Route path="providers/me" element={<ProviderProfilePage />} />
            <Route path="providers/:providerId" element={<ProviderProfilePage />} />
            <Route path="appointments" element={<AppointmentsListPage />} />
            <Route path="appointments/book" element={<BookAppointmentPage />} />
            <Route path="appointments/availability" element={<AvailabilityEditorPage />} />
            <Route path="appointments/:appointmentId" element={<AppointmentDetailPage />} />
            <Route path="search" element={<SearchResultsPage />} />
            <Route path="authentication/users" element={<UsersListPage />} />
            <Route path="authentication/users/new" element={<CreateUserPage />} />
            <Route path="authentication/users/:userID" element={<UserDetailPage />} />
            <Route path="authentication/roles" element={<RolesListPage />} />
            <Route path="authentication/roles/new" element={<RoleFormPage />} />
            <Route path="authentication/roles/:roleID" element={<RoleFormPage />} />
            <Route path="profile" element={<ProfileEditPage />} />
            <Route path="settings" element={<SettingsPage />} />
            <Route path="settings/:tab" element={<SettingsPage />} />
            <Route path="admin/audit" element={<AuditLogPage />} />
            <Route path="admin/feature-flags" element={<FeatureFlagsPage />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Route>
        </Routes>
      )}
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
