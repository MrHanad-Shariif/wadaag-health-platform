import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { useFetch } from '../../shared/useFetch'
import { LoadingState, ErrorState } from '../../shared/StatusMessage'
import { PageHeader } from '../../shared/PageHeader'
import { useToast } from '../../shared/useToast'
import { useAuth } from '../auth/useAuth'
import { formatStatus } from './format'
import {
  acceptReferral,
  cancelReferral,
  completeReferral,
  declineReferral,
  getReferral,
  getReferralEvents,
  startReferral,
} from './api'
import type { Referral } from './types'

const ACTIONS_BY_STATUS: Record<
  string,
  { label: string; run: typeof acceptReferral; className?: string; toast: string }[]
> = {
  routed: [
    { label: 'Accept', run: acceptReferral, className: 'button--primary', toast: 'Referral accepted' },
    { label: 'Decline', run: declineReferral, className: 'button--outline-danger', toast: 'Referral declined' },
  ],
  accepted: [{ label: 'Start', run: startReferral, className: 'button--primary', toast: 'Referral started' }],
  in_progress: [{ label: 'Complete', run: completeReferral, className: 'button--primary', toast: 'Referral completed' }],
}

export function ReferralDetailPage() {
  const { referralId } = useParams<{ referralId: string }>()
  const { user } = useAuth()
  const { show } = useToast()
  const isProvider = user?.role === 'physician' || user?.role === 'hospital_admin'

  const referralState = useFetch(() => getReferral(referralId!), [referralId])
  const eventsState = useFetch(() => getReferralEvents(referralId!), [referralId])

  const [actionError, setActionError] = useState<string | null>(null)
  const [running, setRunning] = useState(false)

  async function runAction(action: (id: string, version: number) => Promise<Referral>, toastMessage: string) {
    if (!referralState.data) return
    setActionError(null)
    setRunning(true)
    try {
      await action(referralId!, referralState.data.version)
      referralState.reload()
      eventsState.reload()
      show(toastMessage)
    } catch (err) {
      setActionError(err instanceof ApiError ? err.message : 'Action failed.')
    } finally {
      setRunning(false)
    }
  }

  if (referralState.loading) return <LoadingState />
  if (referralState.error) return <ErrorState message={referralState.error} />
  const referral = referralState.data!

  const canCancel = ['routed', 'accepted', 'in_progress'].includes(referral.status)
  const statusActions = ACTIONS_BY_STATUS[referral.status] ?? []

  return (
    <div className="page">
      <PageHeader
        breadcrumb={[{ label: 'Patient', to: `/patients/${referral.patient_id}` }, { label: 'Referral' }]}
        title={`${referral.specialty_requested} referral`}
        description={referral.reason}
      />
      <p>
        <span className={`status-pill status-pill--${referral.status}`}>{formatStatus(referral.status)}</span>{' '}
        <span className={`urgency-tag urgency-tag--${referral.urgency}`}>{referral.urgency}</span>
      </p>

      {isProvider && (statusActions.length > 0 || canCancel) && (
        <div className="action-row">
          {statusActions.map((a) => (
            <button
              key={a.label}
              type="button"
              className={`button ${a.className ?? 'button--primary'}`}
              disabled={running}
              onClick={() => runAction(a.run, a.toast)}
            >
              {a.label}
            </button>
          ))}
          {canCancel && (
            <button
              type="button"
              className="button button--danger"
              disabled={running}
              onClick={() => runAction(cancelReferral, 'Referral cancelled')}
            >
              Cancel referral
            </button>
          )}
        </div>
      )}
      {actionError && <ErrorState message={actionError} />}

      <section>
        <h2>Status history</h2>
        {eventsState.loading && <LoadingState />}
        {eventsState.error && <ErrorState message={eventsState.error} />}
        {eventsState.data && (
          <ul className="timeline">
            {eventsState.data.map((e, i) => (
              <li key={i} data-status={e.to_status}>
                <span className={`status-pill status-pill--${e.to_status}`}>{formatStatus(e.to_status)}</span>
                <span className="timeline__time">{new Date(e.occurred_at).toLocaleString()}</span>
                {e.note && <span className="record-list__note">{e.note}</span>}
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  )
}
