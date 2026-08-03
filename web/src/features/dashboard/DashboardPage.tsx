import { ArrowRight, Award, Building2, CalendarCheck, ClipboardList, Stethoscope, UserPlus, Users } from 'lucide-react'
import { Link } from 'react-router-dom'
import { useAuth } from '../auth/useAuth'
import { useFetch } from '../../shared/useFetch'
import { LoadingState, ErrorState } from '../../shared/StatusMessage'
import { PageHeader } from '../../shared/PageHeader'
import { getSummary } from './api'
import { StatCard } from './StatCard'
import { ReferralsStatusChart } from './ReferralsStatusChart'
import type { DoctorActivity } from './types'

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

      {user?.role === 'physician' && <DoctorSummary />}

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

// Computes "completed / total" across every status bucket, guarding
// against a divide-by-zero when there are no referrals at all yet.
function referralSuccessRate(referralsByStatus: Record<string, number>): number {
  const total = Object.values(referralsByStatus).reduce((sum, n) => sum + n, 0)
  if (total === 0) return 0
  const completed = referralsByStatus.completed ?? 0
  return Math.round((completed / total) * 100)
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
        <StatCard
          label="Referral success rate"
          value={referralSuccessRate(summary.referrals_by_status)}
          suffix="%"
          icon={Award}
        />
      </div>

      <ReferralsStatusChart counts={summary.referrals_by_status} />

      <MostActiveDoctorsPanel doctors={summary.most_active_doctors} />
    </>
  )
}

function MostActiveDoctorsPanel({ doctors }: { doctors: DoctorActivity[] }) {
  return (
    <div className="chart-card">
      <div className="chart-card__header">
        <div>
          <h2 className="chart-card__title">Most active doctors</h2>
          <p className="chart-card__subtitle">Ranked by referrals handled</p>
        </div>
      </div>

      {doctors.length === 0 ? (
        <p className="empty-state">No referral activity yet.</p>
      ) : (
        <table className="chart-table">
          <thead>
            <tr>
              <th>Doctor</th>
              <th className="chart-table__num">Referrals</th>
            </tr>
          </thead>
          <tbody>
            {doctors.map((d) => (
              <tr key={d.provider_id}>
                <td>{d.full_name ?? d.provider_id}</td>
                <td className="chart-table__num">{d.referral_count}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}

// Physician-only panel — the three counts below are only ever non-zero
// when the summary was requested by a physician (see Summary's comment),
// so this is gated on role rather than full_access.
function DoctorSummary() {
  const state = useFetch(() => getSummary(), [])

  if (state.loading) return <LoadingState />
  if (state.error) return <ErrorState message={state.error} />
  const summary = state.data!

  return (
    <div className="stat-grid">
      <StatCard label="Pending consults" value={summary.pending_consult_count} icon={Stethoscope} />
      <StatCard label="My referrals pending" value={summary.my_referrals_pending_count} icon={ClipboardList} />
      <StatCard label="Patients seen today" value={summary.patients_today_count} icon={CalendarCheck} />
    </div>
  )
}
