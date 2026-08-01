import { ArrowRight, Building2, ClipboardList, Stethoscope, UserPlus, Users } from 'lucide-react'
import { Link } from 'react-router-dom'
import { useAuth } from '../auth/useAuth'
import { useFetch } from '../../shared/useFetch'
import { LoadingState, ErrorState } from '../../shared/StatusMessage'
import { PageHeader } from '../../shared/PageHeader'
import { getSummary } from './api'
import { StatCard } from './StatCard'
import { ReferralsStatusChart } from './ReferralsStatusChart'

export function DashboardPage() {
  const { user } = useAuth()
  const isProvider = user?.role === 'physician' || user?.role === 'hospital_admin'

  return (
    <div className="page page--wide">
      <PageHeader
        eyebrow="Dashboard"
        title="Welcome back"
        description={`Signed in as ${user?.email ?? user?.phone} · ${user?.role?.replace('_', ' ')}`}
      />

      {user?.full_access && <AdminSummary />}

      {isProvider && (
        <div className="quick-links">
          <Link className="quick-link" to="/patients/new">
            <span className="quick-link__icon quick-link__icon--accent">
              <UserPlus size={20} aria-hidden="true" />
            </span>
            <span className="quick-link__body">
              <strong>Register a patient</strong>
              <span>Onboard a new patient at your facility</span>
            </span>
            <ArrowRight size={18} className="quick-link__arrow" aria-hidden="true" />
          </Link>
          <Link className="quick-link" to="/referrals">
            <span className="quick-link__icon quick-link__icon--secondary">
              <ClipboardList size={20} aria-hidden="true" />
            </span>
            <span className="quick-link__body">
              <strong>Referrals inbox</strong>
              <span>Referrals routed to your facility</span>
            </span>
            <ArrowRight size={18} className="quick-link__arrow" aria-hidden="true" />
          </Link>
        </div>
      )}

      {!isProvider && !user?.full_access && <p className="empty-state">Tools for your role are on the way.</p>}
    </div>
  )
}

function AdminSummary() {
  const state = useFetch(() => getSummary(), [])

  if (state.loading) return <LoadingState />
  if (state.error) return <ErrorState message={state.error} />
  const summary = state.data!

  return (
    <>
      <div className="stat-grid">
        <StatCard label="Patients" value={summary.total_patients} icon={Users} />
        <StatCard label="Encounters" value={summary.total_encounters} icon={Stethoscope} />
        <StatCard label="Facilities" value={summary.total_facilities} icon={Building2} />
        <StatCard label="Users" value={summary.total_users} icon={UserPlus} />
      </div>

      <ReferralsStatusChart counts={summary.referrals_by_status} />
    </>
  )
}
