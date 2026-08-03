import { useState, type FormEvent } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { ErrorState } from '../../shared/StatusMessage'
import { PageHeader } from '../../shared/PageHeader'
import { useToast } from '../../shared/useToast'
import { createConsult } from './api'

export function NewConsultPage() {
  const navigate = useNavigate()
  const { show } = useToast()
  const [searchParams] = useSearchParams()
  const patientId = searchParams.get('patient')

  const [targetProviderId, setTargetProviderId] = useState('')
  const [reason, setReason] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  if (!patientId) {
    return <ErrorState message="Missing patient — start a consult from a patient's record." />
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      const consult = await createConsult({
        patient_id: patientId!,
        target_provider_id: targetProviderId,
        reason,
      })
      show('Consult request sent')
      navigate(`/consults/${consult.id}`)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not request consult.')
      setSubmitting(false)
    }
  }

  return (
    <div className="page page--narrow">
      <PageHeader eyebrow="Consultations" title="Request a consult" />

      <form className="form" onSubmit={handleSubmit}>
        <label htmlFor="targetProvider">Target provider ID</label>
        <input
          id="targetProvider"
          value={targetProviderId}
          onChange={(e) => setTargetProviderId(e.target.value)}
          placeholder="Provider's user ID"
          required
        />
        <p className="form-hint">Ask the target doctor for their provider ID — there's no provider directory yet.</p>

        <label htmlFor="reason">Reason for consult</label>
        <textarea id="reason" value={reason} onChange={(e) => setReason(e.target.value)} rows={4} required />

        {error && <p role="alert" className="form-error">{error}</p>}

        <button type="submit" disabled={submitting}>
          {submitting ? 'Sending…' : 'Send consult request'}
        </button>
      </form>
    </div>
  )
}
