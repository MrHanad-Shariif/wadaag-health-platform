import { useState } from 'react'
import { ApiError } from '../../api/client'
import { useFetch } from '../../shared/useFetch'
import { DataTable, type DataTableColumn } from '../../shared/DataTable'
import { PageHeader } from '../../shared/PageHeader'
import { useToast } from '../../shared/useToast'
import { listSessions, revokeSession } from './api'
import type { Session } from './types'

// Phase 0 stub — lives under features/auth for now; move under a
// Settings area once Phase 9 lands.
export function SessionsPage() {
  const state = useFetch(() => listSessions(), [])
  const { show } = useToast()
  const [revokingId, setRevokingId] = useState<string | null>(null)

  async function handleRevoke(session: Session) {
    setRevokingId(session.id)
    try {
      await revokeSession(session.id)
      state.reload()
      show('Session revoked')
    } catch (err) {
      show(err instanceof ApiError ? err.message : 'Could not revoke session.')
    } finally {
      setRevokingId(null)
    }
  }

  const columns: DataTableColumn<Session>[] = [
    {
      key: 'device',
      header: 'Device',
      sortable: true,
      value: (s) => s.device_label ?? 'Unknown device',
      render: (s) => (
        <div>
          <div>{s.device_label ?? 'Unknown device'}</div>
          {s.user_agent && <div className="page-subtitle">{s.user_agent}</div>}
        </div>
      ),
    },
    {
      key: 'ip',
      header: 'IP address',
      sortable: true,
      value: (s) => s.ip ?? '',
      render: (s) => s.ip ?? '—',
    },
    {
      key: 'created_at',
      header: 'Created',
      sortable: true,
      value: (s) => s.created_at,
      render: (s) => new Date(s.created_at).toLocaleString(),
    },
    {
      key: 'expires_at',
      header: 'Expires',
      sortable: true,
      value: (s) => s.expires_at,
      render: (s) => new Date(s.expires_at).toLocaleString(),
    },
    {
      key: 'actions',
      header: '',
      align: 'right',
      render: (s) => (
        <button
          type="button"
          className="button button--outline-danger"
          disabled={revokingId === s.id}
          onClick={() => handleRevoke(s)}
        >
          {revokingId === s.id ? 'Revoking…' : 'Revoke'}
        </button>
      ),
    },
  ]

  return (
    <div className="page page--wide">
      <PageHeader
        eyebrow="Account"
        title="Active sessions"
        description="Devices and browsers currently signed in to your account. Revoke any session you don't recognize."
      />

      <DataTable
        columns={columns}
        data={state.data}
        loading={state.loading}
        error={state.error}
        getRowKey={(s) => s.id}
        searchPlaceholder="Search sessions…"
        emptyMessage="No active sessions."
      />
    </div>
  )
}
