import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { PageHeader } from '../../shared/PageHeader'
import { useToast } from '../../shared/useToast'
import { createFacility } from './api'
import { FACILITY_TYPES, type FacilityType } from './types'

export function NewFacilityPage() {
  const navigate = useNavigate()
  const { show } = useToast()

  const [name, setName] = useState('')
  const [type, setType] = useState<FacilityType>('hospital')
  const [region, setRegion] = useState('')
  const [district, setDistrict] = useState('')
  const [phone, setPhone] = useState('')
  const [address, setAddress] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      const facility = await createFacility({
        name,
        type,
        region: region || undefined,
        district: district || undefined,
        phone: phone || undefined,
        address: address || undefined,
      })
      show(`${facility.name} created`)
      navigate(`/facilities/${facility.id}`)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not create facility.')
      setSubmitting(false)
    }
  }

  return (
    <div className="page page--narrow">
      <PageHeader
        breadcrumb={[{ label: 'Facilities', to: '/facilities' }, { label: 'New' }]}
        title="New facility"
      />
      <form className="form" onSubmit={handleSubmit}>
        <label htmlFor="name">Name</label>
        <input id="name" value={name} onChange={(e) => setName(e.target.value)} required />

        <label htmlFor="type">Type</label>
        <select id="type" value={type} onChange={(e) => setType(e.target.value as FacilityType)}>
          {FACILITY_TYPES.map((t) => (
            <option key={t} value={t}>
              {t.charAt(0).toUpperCase() + t.slice(1)}
            </option>
          ))}
        </select>

        <label htmlFor="region">Region</label>
        <input id="region" value={region} onChange={(e) => setRegion(e.target.value)} />

        <label htmlFor="district">District</label>
        <input id="district" value={district} onChange={(e) => setDistrict(e.target.value)} />

        <label htmlFor="phone">Phone</label>
        <input id="phone" value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="+252 6XX XXX XXX" />

        <label htmlFor="address">Address</label>
        <input id="address" value={address} onChange={(e) => setAddress(e.target.value)} />

        {error && (
          <p role="alert" className="form-error">
            {error}
          </p>
        )}

        <button type="submit" disabled={submitting}>
          {submitting ? 'Saving…' : 'Create facility'}
        </button>
      </form>
    </div>
  )
}
