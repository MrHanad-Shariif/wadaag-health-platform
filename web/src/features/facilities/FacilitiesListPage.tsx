import { Plus } from 'lucide-react'
import { Link, useNavigate } from 'react-router-dom'
import { useFetch } from '../../shared/useFetch'
import { DataTable, type DataTableColumn } from '../../shared/DataTable'
import { PageHeader } from '../../shared/PageHeader'
import { useAuth } from '../auth/useAuth'
import { listFacilities } from './api'
import type { Facility } from './types'

export function FacilitiesListPage() {
  const navigate = useNavigate()
  const { user } = useAuth()
  // Facility creation is a platform-level action (onboarding a new
  // hospital), restricted to system_admin server-side — a hospital_admin
  // has at most one facility already and creating another isn't "managing
  // their own facility." See requireFacilityAccess in the Go handler.
  const canCreate = user?.role === 'system_admin'
  const state = useFetch(() => listFacilities(), [])

  const columns: DataTableColumn<Facility>[] = [
    { key: 'name', header: 'Name', sortable: true, value: (f) => f.name, render: (f) => f.name },
    {
      key: 'type',
      header: 'Type',
      sortable: true,
      value: (f) => f.type,
      render: (f) => <span className="badge badge--accent">{f.type}</span>,
    },
    {
      key: 'region',
      header: 'Region',
      sortable: true,
      value: (f) => f.region ?? '',
      render: (f) => f.region ?? <span className="empty-state">—</span>,
    },
    {
      key: 'district',
      header: 'District',
      sortable: true,
      value: (f) => f.district ?? '',
      render: (f) => f.district ?? <span className="empty-state">—</span>,
    },
    {
      key: 'phone',
      header: 'Phone',
      render: (f) => f.phone ?? <span className="empty-state">—</span>,
    },
  ]

  return (
    <div className="page page--wide">
      <PageHeader
        eyebrow="Facilities"
        title="Facilities"
        description="Hospitals, clinics, labs, pharmacies, and insurers registered on the platform."
        actions={
          canCreate ? (
            <Link className="button-link" to="/facilities/new">
              <Plus size={16} aria-hidden="true" />
              New facility
            </Link>
          ) : undefined
        }
      />

      <DataTable
        columns={columns}
        data={state.data}
        loading={state.loading}
        error={state.error}
        getRowKey={(f) => f.id}
        onRowClick={(f) => navigate(`/facilities/${f.id}`)}
        searchPlaceholder="Search facilities…"
        emptyMessage="No facilities registered yet."
      />
    </div>
  )
}
