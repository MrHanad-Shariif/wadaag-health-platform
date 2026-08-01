import { useState, type FormEvent } from 'react'
import { ArrowLeft } from 'lucide-react'
import { Link, useParams } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { useFetch } from '../../shared/useFetch'
import { LoadingState, ErrorState } from '../../shared/StatusMessage'
import { useAuth } from '../auth/useAuth'
import { createObservation, getEncounter, listObservations } from './api'
import type { ObservationType } from './types'

export function EncounterDetailPage() {
  const { encounterId } = useParams<{ encounterId: string }>()
  const { user } = useAuth()
  const isProvider = user?.role === 'physician' || user?.role === 'hospital_admin'

  const encounterState = useFetch(() => getEncounter(encounterId!), [encounterId])
  const observationsState = useFetch(() => listObservations(encounterId!), [encounterId])

  const [obsType, setObsType] = useState<ObservationType>('note')
  const [icd10, setIcd10] = useState('')
  const [note, setNote] = useState('')
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  async function handleAddObservation(e: FormEvent) {
    e.preventDefault()
    setCreateError(null)
    setCreating(true)
    try {
      const payload: Record<string, unknown> = { note }
      if (obsType === 'diagnosis' && icd10) payload.icd10 = icd10
      await createObservation(encounterId!, { type: obsType, payload })
      setNote('')
      setIcd10('')
      observationsState.reload()
    } catch (err) {
      setCreateError(err instanceof ApiError ? err.message : 'Could not add observation.')
    } finally {
      setCreating(false)
    }
  }

  if (encounterState.loading) return <LoadingState />
  if (encounterState.error) return <ErrorState message={encounterState.error} />
  const encounter = encounterState.data!

  return (
    <div className="page">
      <Link to={`/patients/${encounter.patient_id}`} className="back-link">
        <ArrowLeft size={16} aria-hidden="true" /> Back to patient
      </Link>
      <h1>{encounter.type.replace('_', ' ')} encounter</h1>
      <p className="page-subtitle">{new Date(encounter.occurred_at).toLocaleString()}</p>
      {encounter.notes && <p>{encounter.notes}</p>}

      <section>
        <h2>Observations</h2>
        {observationsState.loading && <LoadingState />}
        {observationsState.error && <ErrorState message={observationsState.error} />}
        {observationsState.data && !observationsState.data.length && (
          <p className="empty-state">No observations recorded yet.</p>
        )}
        {observationsState.data && observationsState.data.length > 0 && (
          <ul className="record-list">
            {observationsState.data.map((o) => (
              <li key={o.id}>
                <strong>{o.type}</strong>
                {typeof o.payload.icd10 === 'string' && <span> · {o.payload.icd10}</span>}
                {typeof o.payload.note === 'string' && <span className="record-list__note">{o.payload.note}</span>}
                <span className="record-list__timestamp">{new Date(o.recorded_at).toLocaleString()}</span>
              </li>
            ))}
          </ul>
        )}

        {isProvider && (
          <form className="form form--inline" onSubmit={handleAddObservation}>
            <label htmlFor="obsType">Add observation</label>
            <select id="obsType" value={obsType} onChange={(e) => setObsType(e.target.value as ObservationType)}>
              <option value="note">Note</option>
              <option value="vitals">Vitals</option>
              <option value="diagnosis">Diagnosis</option>
            </select>
            {obsType === 'diagnosis' && (
              <input placeholder="ICD-10 code (optional)" value={icd10} onChange={(e) => setIcd10(e.target.value)} />
            )}
            <input placeholder="Details" value={note} onChange={(e) => setNote(e.target.value)} required />
            <button type="submit" disabled={creating}>
              {creating ? 'Adding…' : 'Add'}
            </button>
            {createError && <p role="alert" className="form-error">{createError}</p>}
          </form>
        )}
      </section>
    </div>
  )
}
