import { Link } from 'react-router-dom'
import { useFetch } from '../../shared/useFetch'
import { LoadingState, ErrorState } from '../../shared/StatusMessage'
import { listInbox } from './api'
import { formatStatus } from './format'

export function ReferralsInboxPage() {
  const state = useFetch(() => listInbox(), [])

  return (
    <div className="page">
      <p className="page-eyebrow">Referrals</p>
      <h1>Inbox</h1>
      <p className="page-subtitle">Referrals routed to your facility.</p>

      {state.loading && <LoadingState />}
      {state.error && <ErrorState message={state.error} />}
      {state.data && !state.data.length && <p className="empty-state">No referrals yet.</p>}

      {state.data && state.data.length > 0 && (
        <ul className="record-list">
          {state.data.map((r) => (
            <li key={r.id} data-status={r.status}>
              <Link to={`/referrals/${r.id}`}>
                <span className={`status-pill status-pill--${r.status}`}>{formatStatus(r.status)}</span>{' '}
                {r.specialty_requested} · <span className={`urgency-tag urgency-tag--${r.urgency}`}>{r.urgency}</span>
              </Link>
              <span className="record-list__note">{r.reason}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
