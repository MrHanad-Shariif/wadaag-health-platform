import { Plus } from 'lucide-react'
import { Link, useNavigate } from 'react-router-dom'
import { useFetch } from '../../shared/useFetch'
import { DataTable, type DataTableColumn } from '../../shared/DataTable'
import { PageHeader } from '../../shared/PageHeader'
import { listUsers } from './api'
import type { AuthzUser } from './types'

export function UsersListPage() {
  const navigate = useNavigate()
  const state = useFetch(() => listUsers(), [])

  const columns: DataTableColumn<AuthzUser>[] = [
    {
      key: 'identifier',
      header: 'User',
      sortable: true,
      value: (u) => u.email ?? u.phone ?? '',
      render: (u) => u.email ?? u.phone ?? '—',
    },
    {
      key: 'legacy_role',
      header: 'Legacy role',
      sortable: true,
      value: (u) => u.legacy_role,
      render: (u) => <span className="badge">{u.legacy_role.replace('_', ' ')}</span>,
    },
    {
      key: 'role_name',
      header: 'Permission role',
      sortable: true,
      value: (u) => u.role_name ?? '',
      render: (u) => (u.role_name ? <span className="badge badge--accent">{u.role_name}</span> : <span className="empty-state">None</span>),
    },
    {
      key: 'status',
      header: 'Status',
      sortable: true,
      value: (u) => u.status,
      render: (u) => (
        <span className={`badge ${u.status === 'active' ? 'badge--success' : 'badge--danger'}`}>{u.status}</span>
      ),
    },
    {
      key: 'created_at',
      header: 'Created',
      sortable: true,
      value: (u) => u.created_at,
      render: (u) => new Date(u.created_at).toLocaleDateString(),
    },
  ]

  return (
    <div className="page page--wide">
      <PageHeader
        eyebrow="Authentication"
        title="Users"
        description="Every account on the platform, and the permission role assigned to each."
        actions={
          <Link className="button-link" to="/authentication/users/new">
            <Plus size={16} aria-hidden="true" />
            New user
          </Link>
        }
      />

      <DataTable
        columns={columns}
        data={state.data}
        loading={state.loading}
        error={state.error}
        getRowKey={(u) => u.id}
        onRowClick={(u) => navigate(`/authentication/users/${u.id}`)}
        searchPlaceholder="Search users…"
        emptyMessage="No users yet."
      />
    </div>
  )
}
