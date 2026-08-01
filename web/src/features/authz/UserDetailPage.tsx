import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { useFetch } from '../../shared/useFetch'
import { LoadingState, ErrorState } from '../../shared/StatusMessage'
import { PageHeader } from '../../shared/PageHeader'
import { useToast } from '../../shared/useToast'
import { getUser, listRoles, updateUserRole, updateUserStatus } from './api'

export function UserDetailPage() {
  const { userID } = useParams<{ userID: string }>()
  const { show } = useToast()
  const userState = useFetch(() => getUser(userID!), [userID])
  const rolesState = useFetch(() => listRoles(), [])

  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleRoleChange(roleId: string) {
    setError(null)
    setSaving(true)
    try {
      await updateUserRole(userID!, roleId || null)
      userState.reload()
      show('Permission role updated')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not update permission role.')
    } finally {
      setSaving(false)
    }
  }

  async function handleStatusToggle() {
    if (!userState.data) return
    const next = userState.data.status === 'active' ? 'disabled' : 'active'
    setError(null)
    setSaving(true)
    try {
      await updateUserStatus(userID!, next)
      userState.reload()
      show(next === 'active' ? 'Account re-enabled' : 'Account disabled')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not update status.')
    } finally {
      setSaving(false)
    }
  }

  if (userState.loading || rolesState.loading) return <LoadingState />
  if (userState.error) return <ErrorState message={userState.error} />
  if (rolesState.error) return <ErrorState message={rolesState.error} />
  const user = userState.data!

  return (
    <div className="page page--narrow">
      <PageHeader
        breadcrumb={[{ label: 'Users', to: '/authentication/users' }, { label: user.email ?? user.phone ?? 'User' }]}
        title={user.email ?? user.phone ?? 'User'}
      />

      <dl className="detail-grid">
        <dt>Legacy role</dt>
        <dd>{user.legacy_role.replace('_', ' ')}</dd>
        <dt>Status</dt>
        <dd>
          <span className={`badge ${user.status === 'active' ? 'badge--success' : 'badge--danger'}`}>{user.status}</span>
        </dd>
        <dt>Created</dt>
        <dd>{new Date(user.created_at).toLocaleString()}</dd>
      </dl>

      <div className="form">
        <label htmlFor="permissionRole">Authentication permission role</label>
        <select
          id="permissionRole"
          value={user.role_id ?? ''}
          disabled={saving}
          onChange={(e) => handleRoleChange(e.target.value)}
        >
          <option value="">No admin permissions</option>
          {rolesState.data?.map((r) => (
            <option key={r.id} value={r.id}>
              {r.name}
            </option>
          ))}
        </select>

        {error && (
          <p role="alert" className="form-error">
            {error}
          </p>
        )}

        <div className="action-row">
          <button type="button" className="button button--outline-danger" disabled={saving} onClick={handleStatusToggle}>
            {user.status === 'active' ? 'Disable account' : 'Re-enable account'}
          </button>
        </div>
      </div>
    </div>
  )
}
