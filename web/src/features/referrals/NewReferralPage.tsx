import { useState, type FormEvent } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { useFetch } from '../../shared/useFetch'
import { LoadingState, ErrorState } from '../../shared/StatusMessage'
import { PageHeader } from '../../shared/PageHeader'
import { useToast } from '../../shared/useToast'
import { listFacilities } from '../facilities/api'
import { createReferral } from './api'
import type { Urgency } from './types'

export function NewReferralPage() {
  const navigate = useNavigate()
  const { show } = useToast()
  const [searchParams] = useSearchParams()
  const patientId = searchParams.get('patient')

  const facilitiesState = useFetch(() => listFacilities(), [])

  const [receivingFacilityId, setReceivingFacilityId] = useState('')
  const [specialty, setSpecialty] = useState('')
  const [urgency, setUrgency] = useState<Urgency>('routine')
  const [reason, setReason] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  if (!patientId) {
    return <ErrorState message="Missing patient — start a referral from a patient's record." />
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      const referral = await createReferral({
        patient_id: patientId!,
        receiving_facility_id: receivingFacilityId,
        specialty_requested: specialty,
        urgency,
        reason,
      })
      show('Referral sent')
      navigate(`/referrals/${referral.id}`)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not create referral.')
      setSubmitting(false)
    }
  }

  return (
    <div className="page page--narrow">
      <PageHeader eyebrow="Referrals" title="Refer patient" />

      {facilitiesState.loading && <LoadingState />}
      {facilitiesState.error && <ErrorState message={facilitiesState.error} />}

      {facilitiesState.data && (
        <form className="form" onSubmit={handleSubmit}>
          <label htmlFor="facility">Receiving facility</label>
          <select
            id="facility"
            value={receivingFacilityId}
            onChange={(e) => setReceivingFacilityId(e.target.value)}
            required
          >
            <option value="" disabled>
              Select a facility
            </option>
            {facilitiesState.data.map((f) => (
              <option key={f.id} value={f.id}>
                {f.name} ({f.type})
              </option>
            ))}
          </select>

          <label htmlFor="specialty">Specialty requested</label>
          <input id="specialty" value={specialty} onChange={(e) => setSpecialty(e.target.value)} required />

          <label htmlFor="urgency">Urgency</label>
          <select id="urgency" value={urgency} onChange={(e) => setUrgency(e.target.value as Urgency)}>
            <option value="routine">Routine</option>
            <option value="urgent">Urgent</option>
            <option value="emergency">Emergency</option>
          </select>

          <label htmlFor="reason">Reason for referral</label>
          <textarea id="reason" value={reason} onChange={(e) => setReason(e.target.value)} rows={4} required />

          {error && <p role="alert" className="form-error">{error}</p>}

          <button type="submit" disabled={submitting}>
            {submitting ? 'Sending…' : 'Send referral'}
          </button>
        </form>
      )}
    </div>
  )
}
