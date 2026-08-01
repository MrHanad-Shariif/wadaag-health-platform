import { useState, type FormEvent } from 'react'
import { Send } from 'lucide-react'
import { Link, useParams } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { useFetch } from '../../shared/useFetch'
import { LoadingState, ErrorState } from '../../shared/StatusMessage'
import { PageHeader } from '../../shared/PageHeader'
import { useToast } from '../../shared/useToast'
import { useAuth } from '../auth/useAuth'
import { createEncounter, getPatient, listEncounters } from './api'
import type { Encounter, EncounterType } from './types'
import { listForPatient } from '../referrals/api'
import { formatStatus } from '../referrals/format'
import type { Referral } from '../referrals/types'

export function PatientDetailPage() {
  const { patientId } = useParams<{ patientId: string }>()
  const { user } = useAuth()
  const { show } = useToast()
  const isProvider = user?.role === 'physician' || user?.role === 'hospital_admin'

  const patientState = useFetch(() => getPatient(patientId!), [patientId])
  const encountersState = useFetch(() => listEncounters(patientId!), [patientId])
  const referralsState = useFetch(() => listForPatient(patientId!), [patientId])

  const [encounterType, setEncounterType] = useState<EncounterType>('general_visit')
  const [notes, setNotes] = useState('')
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  async function handleCreateEncounter(e: FormEvent) {
    e.preventDefault()
    setCreateError(null)
    setCreating(true)
    try {
      await createEncounter(patientId!, { type: encounterType, notes: notes || undefined })
      setNotes('')
      encountersState.reload()
      show('Encounter added')
    } catch (err) {
      setCreateError(err instanceof ApiError ? err.message : 'Could not create encounter.')
    } finally {
      setCreating(false)
    }
  }

  if (patientState.loading) return <LoadingState />
  if (patientState.error) return <ErrorState message={patientState.error} />
  const patient = patientState.data!

  return (
    <div className="page">
      <PageHeader
        eyebrow="Patient"
        title={patient.full_name}
        actions={
          isProvider ? (
            <Link className="button-link" to={`/referrals/new?patient=${patient.id}`}>
              <Send size={16} aria-hidden="true" />
              Refer this patient
            </Link>
          ) : undefined
        }
      />
      {(patient.date_of_birth || patient.sex || patient.phone || patient.national_id) && (
        <dl className="detail-grid">
          {patient.date_of_birth && (
            <>
              <dt>Date of birth</dt>
              <dd>{patient.date_of_birth}</dd>
            </>
          )}
          {patient.sex && (
            <>
              <dt>Sex</dt>
              <dd>{patient.sex}</dd>
            </>
          )}
          {patient.phone && (
            <>
              <dt>Phone</dt>
              <dd>{patient.phone}</dd>
            </>
          )}
          {patient.national_id && (
            <>
              <dt>National ID</dt>
              <dd>{patient.national_id}</dd>
            </>
          )}
        </dl>
      )}

      <section>
        <h2>Referrals</h2>
        <ReferralList state={referralsState} />
      </section>

      <section>
        <h2>Encounters</h2>
        <EncounterList state={encountersState} />

        {isProvider && (
          <form className="form form--inline" onSubmit={handleCreateEncounter}>
            <label htmlFor="encType">New encounter</label>
            <select id="encType" value={encounterType} onChange={(e) => setEncounterType(e.target.value as EncounterType)}>
              <option value="general_visit">General visit</option>
              <option value="consult">Consult</option>
              <option value="referral">Referral</option>
            </select>
            <input
              placeholder="Notes (optional)"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
            />
            <button type="submit" disabled={creating}>
              {creating ? 'Adding…' : 'Add encounter'}
            </button>
            {createError && <p role="alert" className="form-error">{createError}</p>}
          </form>
        )}
      </section>
    </div>
  )
}

function EncounterList({ state }: { state: { data: Encounter[] | null; loading: boolean; error: string | null } }) {
  if (state.loading) return <LoadingState />
  if (state.error) return <ErrorState message={state.error} />
  if (!state.data?.length) return <p className="empty-state">No encounters yet.</p>

  return (
    <ul className="record-list">
      {state.data.map((e) => (
        <li key={e.id}>
          <Link to={`/encounters/${e.id}`}>
            {e.type.replace('_', ' ')} — {new Date(e.occurred_at).toLocaleString()}
          </Link>
          {e.notes && <span className="record-list__note">{e.notes}</span>}
        </li>
      ))}
    </ul>
  )
}

function ReferralList({ state }: { state: { data: Referral[] | null; loading: boolean; error: string | null } }) {
  if (state.loading) return <LoadingState />
  if (state.error) return <ErrorState message={state.error} />
  if (!state.data?.length) return <p className="empty-state">No referrals yet.</p>

  return (
    <ul className="record-list">
      {state.data.map((r) => (
        <li key={r.id} data-status={r.status}>
          <Link to={`/referrals/${r.id}`}>
            {r.specialty_requested} — <span className={`status-pill status-pill--${r.status}`}>{formatStatus(r.status)}</span>
          </Link>
        </li>
      ))}
    </ul>
  )
}
