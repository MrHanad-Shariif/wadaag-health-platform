import { useState } from 'react'
import { useFetch } from '../../shared/useFetch'
import { DataTable, type DataTableColumn } from '../../shared/DataTable'
import { PageHeader } from '../../shared/PageHeader'
import { listAuditEntries } from './api'
import type { AuditEntry, AuditResult } from './types'

const RESULT_OPTIONS: { value: AuditResult | ''; label: string }[] = [
  { value: '', label: 'Any result' },
  { value: 'allowed', label: 'Allowed' },
  { value: 'denied', label: 'Denied' },
]

// system_admin only — the underlying endpoint 403s for every other role
// (see backend/internal/audit/handler.go), and this route is only linked
// from the sidebar's Admin group for that role (see shared/Layout.tsx).
export function AuditLogPage() {
  const [actorUserId, setActorUserId] = useState('')
  const [resourceType, setResourceType] = useState('')
  const [result, setResult] = useState<AuditResult | ''>('')

  const state = useFetch(
    () =>
      listAuditEntries({
        actor_user_id: actorUserId.trim() || undefined,
        resource_type: resourceType.trim() || undefined,
        result: result || undefined,
      }),
    [actorUserId, resourceType, result],
  )

  const columns: DataTableColumn<AuditEntry>[] = [
    {
      key: 'occurred_at',
      header: 'When',
      sortable: true,
      value: (e) => e.occurred_at,
      render: (e) => new Date(e.occurred_at).toLocaleString(),
    },
    {
      key: 'actor',
      header: 'Actor',
      sortable: true,
      value: (e) => e.actor_user_id,
      render: (e) => (
        <div>
          <div>{e.actor_user_id}</div>
          <div className="page-subtitle">{e.actor_role.replace('_', ' ')}</div>
        </div>
      ),
    },
    {
      key: 'action',
      header: 'Action',
      sortable: true,
      value: (e) => e.action,
      render: (e) => e.action.replace(/_/g, ' '),
    },
    {
      key: 'resource_type',
      header: 'Resource',
      sortable: true,
      value: (e) => e.resource_type,
      render: (e) => (
        <div>
          <div>{e.resource_type}</div>
          {e.resource_id && <div className="page-subtitle">{e.resource_id}</div>}
        </div>
      ),
    },
    {
      key: 'patient_id',
      header: 'Patient',
      value: (e) => e.patient_id ?? '',
      render: (e) => e.patient_id ?? <span className="empty-state">—</span>,
    },
    {
      key: 'result',
      header: 'Result',
      sortable: true,
      value: (e) => e.result,
      render: (e) => (
        <span className={`badge ${e.result === 'allowed' ? 'badge--success' : 'badge--danger'}`}>{e.result}</span>
      ),
    },
  ]

  return (
    <div className="page page--wide">
      <PageHeader
        eyebrow="Admin"
        title="Audit log"
        description="Every recorded access attempt across the platform — filter by actor, resource type, or outcome."
      />

      <div className="form form--inline">
        <label htmlFor="auditActorFilter">Actor user id</label>
        <input
          id="auditActorFilter"
          value={actorUserId}
          onChange={(e) => setActorUserId(e.target.value)}
          placeholder="uuid…"
        />

        <label htmlFor="auditResourceFilter">Resource type</label>
        <input
          id="auditResourceFilter"
          value={resourceType}
          onChange={(e) => setResourceType(e.target.value)}
          placeholder="e.g. patient"
        />

        <label htmlFor="auditResultFilter">Result</label>
        <select id="auditResultFilter" value={result} onChange={(e) => setResult(e.target.value as AuditResult | '')}>
          {RESULT_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
      </div>

      <DataTable
        columns={columns}
        data={state.data}
        loading={state.loading}
        error={state.error}
        getRowKey={(e) => e.id}
        searchPlaceholder="Search this page…"
        emptyMessage="No matching audit entries."
      />
    </div>
  )
}
