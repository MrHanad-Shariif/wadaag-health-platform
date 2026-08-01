import { Plus } from 'lucide-react'
import { Link, useNavigate } from 'react-router-dom'
import { useFetch } from '../../shared/useFetch'
import { DataTable, type DataTableColumn } from '../../shared/DataTable'
import { PageHeader } from '../../shared/PageHeader'
import { listRoles } from './api'
import type { AuthzRole } from './types'

export function RolesListPage() {
  const navigate = useNavigate()
  const state = useFetch(() => listRoles(), [])

  const columns: DataTableColumn<AuthzRole>[] = [
    { key: 'name', header: 'Name', sortable: true, value: (r) => r.name, render: (r) => r.name },
    {
      key: 'description',
      header: 'Description',
      render: (r) => r.description ?? <span className="empty-state">—</span>,
    },
    {
      key: 'access',
      header: 'Access',
      sortable: true,
      value: (r) => (r.full_access ? 1 : 0),
      render: (r) => (r.full_access ? <span className="badge badge--accent">Full access</span> : 'Scoped'),
    },
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
      <PageHeader
        eyebrow="Authentication"
        title="Roles"
        description="Roles bundle permissions — assign one to a user from the Users screen."
        actions={
          <Link className="button-link" to="/authentication/roles/new">
            <Plus size={16} aria-hidden="true" />
            New role
          </Link>
        }
      />

      <DataTable
        columns={columns}
        data={state.data}
        loading={state.loading}
        error={state.error}
        getRowKey={(r) => r.id}
        onRowClick={(r) => navigate(`/authentication/roles/${r.id}`)}
        searchPlaceholder="Search roles…"
        emptyMessage="No roles yet — create one to start assigning permissions."
      />
    </div>
  )
}
