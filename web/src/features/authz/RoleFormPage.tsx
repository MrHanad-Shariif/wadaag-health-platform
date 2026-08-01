import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { useFetch } from '../../shared/useFetch'
import { LoadingState, ErrorState } from '../../shared/StatusMessage'
import { PageHeader } from '../../shared/PageHeader'
import { useToast } from '../../shared/useToast'
import { createRole, deleteRole, getRole, listPermissions, updateRole } from './api'

const ACTIONS = ['create', 'read', 'update', 'delete'] as const
const ACTION_LABELS: Record<string, string> = { create: 'Create', read: 'Read', update: 'Edit', delete: 'Delete' }

export function RoleFormPage() {
  const { roleID } = useParams<{ roleID: string }>()
  const isEditing = Boolean(roleID)
  const navigate = useNavigate()
  const { show } = useToast()

  const permissionsState = useFetch(() => listPermissions(), [])
  const roleState = useFetch(() => (roleID ? getRole(roleID) : Promise.resolve(null)), [roleID])

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [fullAccess, setFullAccess] = useState(false)
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (roleState.data) {
      setName(roleState.data.name)
      setDescription(roleState.data.description ?? '')
      setFullAccess(roleState.data.full_access)
      setSelected(new Set(roleState.data.permissions?.map((p) => p.id)))
    }
  }, [roleState.data])

  const byResource = useMemo(() => {
    const map = new Map<string, { id: string; action: string }[]>()
    for (const p of permissionsState.data ?? []) {
      const list = map.get(p.resource) ?? []
      list.push({ id: p.id, action: p.action })
      map.set(p.resource, list)
    }
    return map
  }, [permissionsState.data])

  function toggle(id: string) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      const input = { name, description: description || undefined, full_access: fullAccess, permission_ids: [...selected] }
      if (isEditing && roleID) {
        await updateRole(roleID, input)
        show('Role updated')
      } else {
        await createRole(input)
        show('Role created')
      }
      navigate('/authentication/roles')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not save role.')
      setSubmitting(false)
    }
  }

  async function handleDelete() {
    if (!roleID) return
    setError(null)
    try {
      await deleteRole(roleID)
      show('Role deleted')
      navigate('/authentication/roles')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not delete role — it may still be assigned to a user.')
    }
  }

  if (permissionsState.loading || roleState.loading) return <LoadingState />
  if (permissionsState.error) return <ErrorState message={permissionsState.error} />
  if (roleState.error) return <ErrorState message={roleState.error} />

  return (
    <div className="page">
      <PageHeader
        breadcrumb={[
          { label: 'Roles', to: '/authentication/roles' },
          { label: isEditing ? roleState.data?.name ?? 'Edit' : 'New' },
        ]}
        title={isEditing ? 'Edit role' : 'New role'}
      />

      <form className="form" onSubmit={handleSubmit}>
        <label htmlFor="name">Name</label>
        <input id="name" value={name} onChange={(e) => setName(e.target.value)} required />

        <label htmlFor="description">Description</label>
        <input id="description" value={description} onChange={(e) => setDescription(e.target.value)} />

        <label className="checkbox-row">
          <input type="checkbox" checked={fullAccess} onChange={(e) => setFullAccess(e.target.checked)} />
          Full access (bypasses every permission check — reserve for administrators)
        </label>

        {!fullAccess && (
          <>
            <h2 className="form-section-title">Permissions</h2>
            <div className="permission-matrix">
              <table>
                <thead>
                  <tr>
                    <th>Resource</th>
                    {ACTIONS.map((a) => (
                      <th key={a}>{ACTION_LABELS[a]}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {[...byResource.entries()].map(([resource, perms]) => (
                    <tr key={resource}>
                      <td className="permission-matrix__resource">{resource}</td>
                      {ACTIONS.map((action) => {
                        const perm = perms.find((p) => p.action === action)
                        if (!perm) return <td key={action} />
                        return (
                          <td key={action}>
                            <input type="checkbox" checked={selected.has(perm.id)} onChange={() => toggle(perm.id)} />
                          </td>
                        )
                      })}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </>
        )}

        {error && (
          <p role="alert" className="form-error">
            {error}
          </p>
        )}

        <div className="action-row">
          <button type="submit" className="button button--primary" disabled={submitting}>
            {submitting ? 'Saving…' : isEditing ? 'Save changes' : 'Create role'}
          </button>
          {isEditing && (
            <button type="button" className="button button--outline-danger" onClick={handleDelete}>
              Delete role
            </button>
          )}
        </div>
      </form>
    </div>
  )
}
