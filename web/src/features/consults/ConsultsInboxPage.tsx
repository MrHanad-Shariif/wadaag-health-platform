import { useNavigate } from 'react-router-dom'
import { ArrowDownLeft, ArrowUpRight } from 'lucide-react'
import { useFetch } from '../../shared/useFetch'
import { DataTable, type DataTableColumn } from '../../shared/DataTable'
import { PageHeader } from '../../shared/PageHeader'
import { ProviderDisplayName } from '../providers/ProviderDisplayName'
import { listInbox } from './api'
import { formatStatus } from './format'
import type { Consultation } from './types'

// The "other party" provider id for a consult row — the requester for an
// incoming request, the target for one we sent.
function otherProviderId(c: Consultation): string {
  return c.direction === 'incoming' ? c.requesting_provider_id : c.target_provider_id
}

export function ConsultsInboxPage() {
  const navigate = useNavigate()
  const state = useFetch(() => listInbox(), [])

  const columns: DataTableColumn<Consultation>[] = [
    {
      key: 'direction',
      header: 'Direction',
      sortable: true,
      value: (c) => c.direction ?? '',
      render: (c) =>
        c.direction ? (
          <span className={`direction-tag direction-tag--${c.direction}`}>
            {c.direction === 'incoming' ? (
              <ArrowDownLeft size={14} aria-hidden="true" />
            ) : (
              <ArrowUpRight size={14} aria-hidden="true" />
            )}
            {c.direction === 'incoming' ? 'Incoming' : 'Outgoing'}
          </span>
        ) : null,
    },
    {
      key: 'status',
      header: 'Status',
      sortable: true,
      value: (c) => c.status,
      render: (c) => <span className={`status-pill status-pill--${c.status}`}>{formatStatus(c.status)}</span>,
    },
    {
      key: 'other_provider',
      header: 'With',
      value: (c) => otherProviderId(c),
      render: (c) => <ProviderDisplayName providerId={otherProviderId(c)} />,
    },
    { key: 'reason', header: 'Reason', render: (c) => c.reason },
    {
      key: 'created_at',
      header: 'Created',
      sortable: true,
      value: (c) => c.created_at,
      render: (c) => new Date(c.created_at).toLocaleDateString(),
    },
  ]

  return (
    <div className="page page--wide">
      <PageHeader eyebrow="Consultations" title="Inbox" description="Second-opinion requests you sent or received." />

      <DataTable
        columns={columns}
        data={state.data}
        loading={state.loading}
        error={state.error}
        getRowKey={(c) => c.id}
        onRowClick={(c) => navigate(`/consults/${c.id}`)}
        searchPlaceholder="Search consultations…"
        emptyMessage="No consultations yet."
      />
    </div>
  )
}
