import { useState, type FormEvent } from 'react'
import { useParams } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { useFetch } from '../../shared/useFetch'
import { LoadingState, ErrorState } from '../../shared/StatusMessage'
import { DataTable, type DataTableColumn } from '../../shared/DataTable'
import { PageHeader } from '../../shared/PageHeader'
import { useToast } from '../../shared/useToast'
import { useAuth } from '../auth/useAuth'
import {
  createBranch,
  createDepartment,
  deleteBranch,
  deleteDepartment,
  getFacility,
  listBranches,
  listDepartments,
  updateBranch,
  updateDepartment,
  updateFacility,
} from './api'
import { WEEKDAYS, type Branch, type BranchInput, type Department, type DepartmentInput, type WeekdayKey, type WorkingHours } from './types'

export function FacilityDetailPage() {
  const { facilityId } = useParams<{ facilityId: string }>()
  const { user } = useAuth()
  const { show } = useToast()
  const canManage = user?.role === 'system_admin' || user?.role === 'hospital_admin'

  const facilityState = useFetch(() => getFacility(facilityId!), [facilityId])

  if (facilityState.loading) return <LoadingState />
  if (facilityState.error) return <ErrorState message={facilityState.error} />
  const facility = facilityState.data!

  return (
    <div className="page">
      <PageHeader
        breadcrumb={[{ label: 'Facilities', to: '/facilities' }, { label: facility.name }]}
        title={facility.name}
        description={facility.type.charAt(0).toUpperCase() + facility.type.slice(1)}
      />

      <dl className="detail-grid">
        <dt>Type</dt>
        <dd>{facility.type}</dd>
        {facility.region && (
          <>
            <dt>Region</dt>
            <dd>{facility.region}</dd>
          </>
        )}
        {facility.district && (
          <>
            <dt>District</dt>
            <dd>{facility.district}</dd>
          </>
        )}
        {facility.phone && (
          <>
            <dt>Phone</dt>
            <dd>{facility.phone}</dd>
          </>
        )}
        {facility.address && (
          <>
            <dt>Address</dt>
            <dd>{facility.address}</dd>
          </>
        )}
      </dl>

      <LogoAndHoursSection
        facilityId={facilityId!}
        logoUrl={facility.logo_url}
        workingHours={facility.working_hours}
        canManage={canManage}
        onSaved={() => {
          facilityState.reload()
          show('Facility updated')
        }}
      />

      <BranchesSection facilityId={facilityId!} canManage={canManage} />
      <DepartmentsSection facilityId={facilityId!} canManage={canManage} />
    </div>
  )
}

function LogoAndHoursSection({
  facilityId,
  logoUrl,
  workingHours,
  canManage,
  onSaved,
}: {
  facilityId: string
  logoUrl?: string
  workingHours?: WorkingHours
  canManage: boolean
  onSaved: () => void
}) {
  const [editing, setEditing] = useState(false)
  const [logoInput, setLogoInput] = useState(logoUrl ?? '')
  const [hours, setHours] = useState<Record<WeekdayKey, { enabled: boolean; open: string; close: string }>>(() =>
    buildHoursState(workingHours),
  )
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  function buildHoursState(wh?: WorkingHours) {
    const result = {} as Record<WeekdayKey, { enabled: boolean; open: string; close: string }>
    for (const { key } of WEEKDAYS) {
      const entry = wh?.[key]
      result[key] = { enabled: Boolean(entry), open: entry?.open ?? '08:00', close: entry?.close ?? '17:00' }
    }
    return result
  }

  function startEditing() {
    setLogoInput(logoUrl ?? '')
    setHours(buildHoursState(workingHours))
    setError(null)
    setEditing(true)
  }

  function toggleDay(day: WeekdayKey) {
    setHours((prev) => ({ ...prev, [day]: { ...prev[day], enabled: !prev[day].enabled } }))
  }

  function setDayTime(day: WeekdayKey, field: 'open' | 'close', value: string) {
    setHours((prev) => ({ ...prev, [day]: { ...prev[day], [field]: value } }))
  }

  async function handleSave(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSaving(true)
    try {
      const working_hours: WorkingHours = {}
      for (const { key } of WEEKDAYS) {
        const day = hours[key]
        if (day.enabled) working_hours[key] = { open: day.open, close: day.close }
      }
      await updateFacility(facilityId, { logo_url: logoInput || undefined, working_hours })
      setEditing(false)
      onSaved()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not update facility.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <section>
      <h2>Logo &amp; working hours</h2>

      {!editing && (
        <>
          <dl className="detail-grid">
            <dt>Logo</dt>
            <dd>
              {logoUrl ? (
                <img src={logoUrl} alt="" className="facility-logo" />
              ) : (
                <span className="empty-state">No logo set</span>
              )}
            </dd>
          </dl>

          {workingHours && Object.keys(workingHours).length > 0 ? (
            <ul className="record-list">
              {WEEKDAYS.filter(({ key }) => workingHours[key]).map(({ key, label }) => (
                <li key={key}>
                  {label} — {workingHours[key]!.open} to {workingHours[key]!.close}
                </li>
              ))}
            </ul>
          ) : (
            <p className="empty-state">No working hours set.</p>
          )}

          {canManage && (
            <button type="button" className="button-link" onClick={startEditing}>
              Edit logo &amp; hours
            </button>
          )}
        </>
      )}

      {editing && (
        <form className="form" onSubmit={handleSave}>
          <label htmlFor="logoUrl">Logo URL</label>
          <input
            id="logoUrl"
            value={logoInput}
            onChange={(e) => setLogoInput(e.target.value)}
            placeholder="https://…"
          />

          <h2 className="form-section-title">Working hours</h2>
          {WEEKDAYS.map(({ key, label }) => (
            <label key={key} className="checkbox-row">
              <input type="checkbox" checked={hours[key].enabled} onChange={() => toggleDay(key)} />
              {label}
              <input
                type="time"
                value={hours[key].open}
                disabled={!hours[key].enabled}
                onChange={(e) => setDayTime(key, 'open', e.target.value)}
              />
              <span>to</span>
              <input
                type="time"
                value={hours[key].close}
                disabled={!hours[key].enabled}
                onChange={(e) => setDayTime(key, 'close', e.target.value)}
              />
            </label>
          ))}

          {error && (
            <p role="alert" className="form-error">
              {error}
            </p>
          )}

          <div className="action-row">
            <button type="submit" disabled={saving}>
              {saving ? 'Saving…' : 'Save'}
            </button>
            <button type="button" onClick={() => setEditing(false)} disabled={saving}>
              Cancel
            </button>
          </div>
        </form>
      )}
    </section>
  )
}

function BranchesSection({ facilityId, canManage }: { facilityId: string; canManage: boolean }) {
  const { show } = useToast()
  const branchesState = useFetch(() => listBranches(facilityId), [facilityId])

  const [editingBranch, setEditingBranch] = useState<Branch | null>(null)
  const [creating, setCreating] = useState(false)

  async function handleDelete(branch: Branch) {
    if (!window.confirm(`Delete branch "${branch.name}"?`)) return
    try {
      await deleteBranch(branch.id)
      branchesState.reload()
      show('Branch deleted')
    } catch (err) {
      show(err instanceof ApiError ? err.message : 'Could not delete branch.')
    }
  }

  const columns: DataTableColumn<Branch>[] = [
    { key: 'name', header: 'Name', sortable: true, value: (b) => b.name, render: (b) => b.name },
    { key: 'address', header: 'Address', render: (b) => b.address ?? <span className="empty-state">—</span> },
    { key: 'phone', header: 'Phone', render: (b) => b.phone ?? <span className="empty-state">—</span> },
  ]
  if (canManage) {
    columns.push({
      key: 'actions',
      header: '',
      align: 'right',
      render: (b) => (
        <div className="action-row action-row--tight">
          <button type="button" className="button" onClick={() => setEditingBranch(b)}>
            Edit
          </button>
          <button type="button" className="button button--outline-danger" onClick={() => handleDelete(b)}>
            Delete
          </button>
        </div>
      ),
    })
  }

  return (
    <section>
      <h2>Branches</h2>

      <DataTable
        columns={columns}
        data={branchesState.data}
        loading={branchesState.loading}
        error={branchesState.error}
        getRowKey={(b) => b.id}
        searchPlaceholder="Search branches…"
        emptyMessage="No branches yet."
      />

      {canManage && !editingBranch && !creating && (
        <button type="button" className="button-link" onClick={() => setCreating(true)}>
          Add branch
        </button>
      )}

      {canManage && creating && (
        <BranchForm
          submitLabel="Add branch"
          onCancel={() => setCreating(false)}
          onSubmit={async (input) => {
            await createBranch(facilityId, input)
            setCreating(false)
            branchesState.reload()
            show('Branch added')
          }}
        />
      )}

      {canManage && editingBranch && (
        <BranchForm
          initial={editingBranch}
          submitLabel="Save changes"
          onCancel={() => setEditingBranch(null)}
          onSubmit={async (input) => {
            await updateBranch(editingBranch.id, input)
            setEditingBranch(null)
            branchesState.reload()
            show('Branch updated')
          }}
        />
      )}
    </section>
  )
}

function BranchForm({
  initial,
  submitLabel,
  onCancel,
  onSubmit,
}: {
  initial?: Branch
  submitLabel: string
  onCancel: () => void
  onSubmit: (input: BranchInput) => Promise<void>
}) {
  const [name, setName] = useState(initial?.name ?? '')
  const [address, setAddress] = useState(initial?.address ?? '')
  const [phone, setPhone] = useState(initial?.phone ?? '')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      await onSubmit({ name, address: address || undefined, phone: phone || undefined })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not save branch.')
      setSubmitting(false)
    }
  }

  return (
    <form className="form" onSubmit={handleSubmit}>
      <label htmlFor="branchName">Name</label>
      <input id="branchName" value={name} onChange={(e) => setName(e.target.value)} required />

      <label htmlFor="branchAddress">Address</label>
      <input id="branchAddress" value={address} onChange={(e) => setAddress(e.target.value)} />

      <label htmlFor="branchPhone">Phone</label>
      <input id="branchPhone" value={phone} onChange={(e) => setPhone(e.target.value)} />

      {error && (
        <p role="alert" className="form-error">
          {error}
        </p>
      )}

      <div className="action-row">
        <button type="submit" disabled={submitting}>
          {submitting ? 'Saving…' : submitLabel}
        </button>
        <button type="button" onClick={onCancel} disabled={submitting}>
          Cancel
        </button>
      </div>
    </form>
  )
}

function DepartmentsSection({ facilityId, canManage }: { facilityId: string; canManage: boolean }) {
  const { show } = useToast()
  const departmentsState = useFetch(() => listDepartments(facilityId), [facilityId])

  const [editingDepartment, setEditingDepartment] = useState<Department | null>(null)
  const [creating, setCreating] = useState(false)

  async function handleDelete(department: Department) {
    if (!window.confirm(`Delete department "${department.name}"?`)) return
    try {
      await deleteDepartment(department.id)
      departmentsState.reload()
      show('Department deleted')
    } catch (err) {
      show(err instanceof ApiError ? err.message : 'Could not delete department.')
    }
  }

  const columns: DataTableColumn<Department>[] = [
    { key: 'name', header: 'Name', sortable: true, value: (d) => d.name, render: (d) => d.name },
  ]
  if (canManage) {
    columns.push({
      key: 'actions',
      header: '',
      align: 'right',
      render: (d) => (
        <div className="action-row action-row--tight">
          <button type="button" className="button" onClick={() => setEditingDepartment(d)}>
            Edit
          </button>
          <button type="button" className="button button--outline-danger" onClick={() => handleDelete(d)}>
            Delete
          </button>
        </div>
      ),
    })
  }

  return (
    <section>
      <h2>Departments</h2>

      <DataTable
        columns={columns}
        data={departmentsState.data}
        loading={departmentsState.loading}
        error={departmentsState.error}
        getRowKey={(d) => d.id}
        searchPlaceholder="Search departments…"
        emptyMessage="No departments yet."
      />

      {canManage && !editingDepartment && !creating && (
        <button type="button" className="button-link" onClick={() => setCreating(true)}>
          Add department
        </button>
      )}

      {canManage && creating && (
        <DepartmentForm
          submitLabel="Add department"
          onCancel={() => setCreating(false)}
          onSubmit={async (input) => {
            await createDepartment(facilityId, input)
            setCreating(false)
            departmentsState.reload()
            show('Department added')
          }}
        />
      )}

      {canManage && editingDepartment && (
        <DepartmentForm
          initial={editingDepartment}
          submitLabel="Save changes"
          onCancel={() => setEditingDepartment(null)}
          onSubmit={async (input) => {
            await updateDepartment(editingDepartment.id, input)
            setEditingDepartment(null)
            departmentsState.reload()
            show('Department updated')
          }}
        />
      )}
    </section>
  )
}

function DepartmentForm({
  initial,
  submitLabel,
  onCancel,
  onSubmit,
}: {
  initial?: Department
  submitLabel: string
  onCancel: () => void
  onSubmit: (input: DepartmentInput) => Promise<void>
}) {
  const [name, setName] = useState(initial?.name ?? '')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      await onSubmit({ name })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not save department.')
      setSubmitting(false)
    }
  }

  return (
    <form className="form" onSubmit={handleSubmit}>
      <label htmlFor="departmentName">Name</label>
      <input id="departmentName" value={name} onChange={(e) => setName(e.target.value)} required />

      {error && (
        <p role="alert" className="form-error">
          {error}
        </p>
      )}

      <div className="action-row">
        <button type="submit" disabled={submitting}>
          {submitting ? 'Saving…' : submitLabel}
        </button>
        <button type="button" onClick={onCancel} disabled={submitting}>
          Cancel
        </button>
      </div>
    </form>
  )
}
