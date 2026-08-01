import { ArrowRight, ClipboardList, UserPlus } from 'lucide-react'
import { Link } from 'react-router-dom'
import { useAuth } from '../auth/useAuth'

export function DashboardPage() {
  const { user } = useAuth()
  const isProvider = user?.role === 'physician' || user?.role === 'hospital_admin'

  return (
    <div className="page">
      <p className="page-eyebrow">Dashboard</p>
      <h1>Welcome back</h1>
      <p className="page-subtitle">
        Signed in as <strong>{user?.email ?? user?.phone}</strong> · {user?.role?.replace('_', ' ')}
      </p>

      {isProvider ? (
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
      ) : (
        <p className="empty-state">Tools for your role are on the way.</p>
      )}
    </div>
  )
}
