import { useNavigate } from 'react-router-dom'
import { useFetch } from '../../shared/useFetch'
import { DataTable, type DataTableColumn } from '../../shared/DataTable'
import { PageHeader } from '../../shared/PageHeader'
import { listInbox } from './api'
import { formatStatus } from './format'
import type { Referral } from './types'

export function ReferralsInboxPage() {
  const navigate = useNavigate()
  const state = useFetch(() => listInbox(), [])

  const columns: DataTableColumn<Referral>[] = [
    {
      key: 'status',
      header: 'Status',
      sortable: true,
      value: (r) => r.status,
      render: (r) => <span className={`status-pill status-pill--${r.status}`}>{formatStatus(r.status)}</span>,
    },
    {
      key: 'specialty_requested',
      header: 'Specialty',
      sortable: true,
      value: (r) => r.specialty_requested,
      render: (r) => r.specialty_requested,
    },
    {
      key: 'urgency',
      header: 'Urgency',
      sortable: true,
      value: (r) => r.urgency,
      render: (r) => <span className={`urgency-tag urgency-tag--${r.urgency}`}>{r.urgency}</span>,
    },
    { key: 'reason', header: 'Reason', render: (r) => r.reason },
    {
      key: 'created_at',
      header: 'Created',
      sortable: true,
      value: (r) => r.created_at,
      render: (r) => new Date(r.created_at).toLocaleDateString(),
    },
  ]

  return (
    <div className="page page--wide">
      <PageHeader eyebrow="Referrals" title="Inbox" description="Referrals routed to your facility." />

      <DataTable
        columns={columns}
        data={state.data}
        loading={state.loading}
        error={state.error}
        getRowKey={(r) => r.id}
        onRowClick={(r) => navigate(`/referrals/${r.id}`)}
        searchPlaceholder="Search referrals…"
        emptyMessage="No referrals yet."
      />
    </div>
  )
}
