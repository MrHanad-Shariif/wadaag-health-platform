import { useNavigate } from 'react-router-dom'
import { useFetch } from '../../shared/useFetch'
import { DataTable, type DataTableColumn } from '../../shared/DataTable'
import { PageHeader } from '../../shared/PageHeader'
import { listMine } from './api'
import { formatStatus } from './format'
import type { Appointment } from './types'

export function AppointmentsListPage() {
  const navigate = useNavigate()
  const state = useFetch(() => listMine(), [])

  const columns: DataTableColumn<Appointment>[] = [
    {
      key: 'status',
      header: 'Status',
      sortable: true,
      value: (a) => a.status,
      render: (a) => <span className={`status-pill status-pill--${a.status}`}>{formatStatus(a.status)}</span>,
    },
    {
      key: 'start_at',
      header: 'Date',
      sortable: true,
      value: (a) => a.start_at,
      render: (a) => new Date(a.start_at).toLocaleDateString(),
    },
    {
      key: 'time',
      header: 'Time',
      value: (a) => a.start_at,
      render: (a) =>
        `${new Date(a.start_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })} – ${new Date(
          a.end_at,
        ).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`,
    },
    { key: 'reason', header: 'Reason', render: (a) => a.reason ?? <span className="empty-state">—</span> },
  ]

  return (
    <div className="page page--wide">
      <PageHeader eyebrow="Appointments" title="My appointments" description="Upcoming and past appointments." />

      <DataTable
        columns={columns}
        data={state.data}
        loading={state.loading}
        error={state.error}
        getRowKey={(a) => a.id}
        onRowClick={(a) => navigate(`/appointments/${a.id}`)}
        searchPlaceholder="Search appointments…"
        emptyMessage="No appointments yet."
      />
    </div>
  )
}
