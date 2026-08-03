import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { useFetch } from '../../shared/useFetch'
import { LoadingState, ErrorState } from '../../shared/StatusMessage'
import { PageHeader } from '../../shared/PageHeader'
import { useToast } from '../../shared/useToast'
import { listFacilities } from '../facilities/api'
import type { Role as LegacyRole } from '../auth/types'
import { createUser, listRoles } from './api'

const LEGACY_ROLES: { value: LegacyRole; label: string }[] = [
  { value: 'patient', label: 'Patient' },
  { value: 'physician', label: 'Physician' },
  { value: 'hospital_admin', label: 'Hospital admin' },
  { value: 'lab_tech', label: 'Lab tech' },
  { value: 'pharmacist', label: 'Pharmacist' },
  { value: 'insurer', label: 'Insurer' },
  { value: 'system_admin', label: 'System admin' },
]

export function CreateUserPage() {
  const navigate = useNavigate()
  const { show } = useToast()
  const rolesState = useFetch(() => listRoles(), [])
  const facilitiesState = useFetch(() => listFacilities(), [])

  const [fullName, setFullName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [legacyRole, setLegacyRole] = useState<LegacyRole>('physician')
  const [roleId, setRoleId] = useState('')
  const [facilityId, setFacilityId] = useState('')
  const [specialty, setSpecialty] = useState('')
  const [licenseNumber, setLicenseNumber] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      await createUser({
        email,
        password,
        full_name: fullName,
        legacy_role: legacyRole,
        role_id: roleId || undefined,
        facility_id: facilityId || undefined,
        specialty: specialty || undefined,
        license_number: licenseNumber || undefined,
      })
      show('User created')
      navigate('/authentication/users')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not create user.')
      setSubmitting(false)
    }
  }

  if (rolesState.loading || facilitiesState.loading) return <LoadingState />
  if (rolesState.error) return <ErrorState message={rolesState.error} />
  if (facilitiesState.error) return <ErrorState message={facilitiesState.error} />

  return (
    <div className="page page--narrow">
      <PageHeader
        breadcrumb={[{ label: 'Users', to: '/authentication/users' }, { label: 'New' }]}
        title="New user"
      />

      <form className="form" onSubmit={handleSubmit}>
        <label htmlFor="fullName">Full name</label>
        <input id="fullName" type="text" value={fullName} onChange={(e) => setFullName(e.target.value)} required />

        <label htmlFor="email">Email</label>
        <input id="email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />

        <label htmlFor="password">Temporary password</label>
        <input
          id="password"
          type="text"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          minLength={8}
          required
        />

        <label htmlFor="legacyRole">Role</label>
        <select id="legacyRole" value={legacyRole} onChange={(e) => setLegacyRole(e.target.value as LegacyRole)}>
          {LEGACY_ROLES.map((r) => (
            <option key={r.value} value={r.value}>
              {r.label}
            </option>
          ))}
        </select>

        <label htmlFor="facility">Facility (optional)</label>
        <select id="facility" value={facilityId} onChange={(e) => setFacilityId(e.target.value)}>
          <option value="">No facility</option>
          {facilitiesState.data?.map((f) => (
            <option key={f.id} value={f.id}>
              {f.name}
            </option>
          ))}
        </select>

        {facilityId && (
          <>
            <label htmlFor="specialty">Specialty (optional)</label>
            <input id="specialty" value={specialty} onChange={(e) => setSpecialty(e.target.value)} />

            <label htmlFor="license">License number (optional)</label>
            <input id="license" value={licenseNumber} onChange={(e) => setLicenseNumber(e.target.value)} />
          </>
        )}

        <label htmlFor="permissionRole">Authentication permission role (optional)</label>
        <select id="permissionRole" value={roleId} onChange={(e) => setRoleId(e.target.value)}>
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

        <button type="submit" disabled={submitting}>
          {submitting ? 'Creating…' : 'Create user'}
        </button>
      </form>
    </div>
  )
}
