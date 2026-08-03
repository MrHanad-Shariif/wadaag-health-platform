import { useEffect, useState, type FormEvent } from 'react'
import { Send, Stethoscope } from 'lucide-react'
import { Link, useParams } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { useFetch } from '../../shared/useFetch'
import { LoadingState, ErrorState } from '../../shared/StatusMessage'
import { PageHeader } from '../../shared/PageHeader'
import { useToast } from '../../shared/useToast'
import { useAuth } from '../auth/useAuth'
import {
  createEncounter,
  getMedicalHistory,
  getPatient,
  listEncounters,
  updateMedicalHistory,
  updatePatientDemographics,
} from './api'
import type { Encounter, EncounterType, UpdateMedicalHistoryInput } from './types'
import { listForPatient } from '../referrals/api'
import { formatStatus } from '../referrals/format'
import type { Referral } from '../referrals/types'
import { AttachmentsSection } from '../attachments/AttachmentsSection'

type MedicalHistoryArrayField = keyof UpdateMedicalHistoryInput

const MEDICAL_HISTORY_FIELDS: { key: MedicalHistoryArrayField; label: string }[] = [
  { key: 'allergies', label: 'Allergies' },
  { key: 'chronic_conditions', label: 'Chronic conditions' },
  { key: 'current_medications', label: 'Current medications' },
  { key: 'past_surgeries', label: 'Past surgeries' },
  { key: 'family_history', label: 'Family history' },
  { key: 'vaccination_history', label: 'Vaccination history' },
]

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

  const [editingDemographics, setEditingDemographics] = useState(false)
  const [genderInput, setGenderInput] = useState('')
  const [bloodGroupInput, setBloodGroupInput] = useState('')
  const [savingDemographics, setSavingDemographics] = useState(false)
  const [demographicsError, setDemographicsError] = useState<string | null>(null)

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

  function startEditingDemographics() {
    setGenderInput(patientState.data?.gender ?? '')
    setBloodGroupInput(patientState.data?.blood_group ?? '')
    setDemographicsError(null)
    setEditingDemographics(true)
  }

  async function handleSaveDemographics(e: FormEvent) {
    e.preventDefault()
    setDemographicsError(null)
    setSavingDemographics(true)
    try {
      await updatePatientDemographics(patientId!, {
        gender: genderInput || undefined,
        blood_group: bloodGroupInput || undefined,
      })
      patientState.reload()
      setEditingDemographics(false)
      show('Demographics updated')
    } catch (err) {
      setDemographicsError(err instanceof ApiError ? err.message : 'Could not update demographics.')
    } finally {
      setSavingDemographics(false)
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
            <>
              <Link className="button-link" to={`/referrals/new?patient=${patient.id}`}>
                <Send size={16} aria-hidden="true" />
                Refer this patient
              </Link>
              <Link className="button-link" to={`/consults/new?patient=${patient.id}`}>
                <Stethoscope size={16} aria-hidden="true" />
                Request consult
              </Link>
            </>
          ) : undefined
        }
      />
      {(patient.date_of_birth || patient.sex || patient.gender || patient.blood_group || patient.phone || patient.national_id) && (
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
          {patient.gender && (
            <>
              <dt>Gender</dt>
              <dd>{patient.gender}</dd>
            </>
          )}
          {patient.blood_group && (
            <>
              <dt>Blood group</dt>
              <dd>{patient.blood_group}</dd>
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

      {isProvider && (
        <section>
          <h2>Demographics</h2>
          {editingDemographics ? (
            <form className="form form--inline" onSubmit={handleSaveDemographics}>
              <label htmlFor="demoGender">Gender</label>
              <select id="demoGender" value={genderInput} onChange={(e) => setGenderInput(e.target.value)}>
                <option value="">Not specified</option>
                <option value="female">Female</option>
                <option value="male">Male</option>
              </select>
              <label htmlFor="demoBloodGroup">Blood group</label>
              <input
                id="demoBloodGroup"
                value={bloodGroupInput}
                onChange={(e) => setBloodGroupInput(e.target.value)}
                placeholder="O+, A-, ..."
              />
              <button type="submit" disabled={savingDemographics}>
                {savingDemographics ? 'Saving…' : 'Save'}
              </button>
              <button type="button" onClick={() => setEditingDemographics(false)} disabled={savingDemographics}>
                Cancel
              </button>
              {demographicsError && <p role="alert" className="form-error">{demographicsError}</p>}
            </form>
          ) : (
            <button type="button" className="button-link" onClick={startEditingDemographics}>
              Edit demographics
            </button>
          )}
        </section>
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

      <MedicalHistorySection patientId={patientId!} isProvider={isProvider} />

      <AttachmentsSection patientId={patientId!} isProvider={isProvider} />
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

function MedicalHistorySection({ patientId, isProvider }: { patientId: string; isProvider: boolean }) {
  const { show } = useToast()
  const historyState = useFetch(() => getMedicalHistory(patientId), [patientId])

  const [editing, setEditing] = useState(false)
  const [fieldText, setFieldText] = useState<Record<MedicalHistoryArrayField, string>>({
    allergies: '',
    chronic_conditions: '',
    current_medications: '',
    past_surgeries: '',
    family_history: '',
    vaccination_history: '',
  })
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!historyState.data) return
    const next = {} as Record<MedicalHistoryArrayField, string>
    for (const { key } of MEDICAL_HISTORY_FIELDS) {
      next[key] = historyState.data[key].join(', ')
    }
    setFieldText(next)
  }, [historyState.data])

  function startEditing() {
    setError(null)
    setEditing(true)
  }

  function cancelEditing() {
    setEditing(false)
    setError(null)
    if (!historyState.data) return
    const next = {} as Record<MedicalHistoryArrayField, string>
    for (const { key } of MEDICAL_HISTORY_FIELDS) {
      next[key] = historyState.data[key].join(', ')
    }
    setFieldText(next)
  }

  async function handleSave(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSaving(true)
    try {
      const input = {} as UpdateMedicalHistoryInput
      for (const { key } of MEDICAL_HISTORY_FIELDS) {
        input[key] = (fieldText[key] ?? '')
          .split(',')
          .map((v) => v.trim())
          .filter(Boolean)
      }
      await updateMedicalHistory(patientId, input)
      historyState.reload()
      setEditing(false)
      show('Medical history updated')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not update medical history.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <section>
      <h2>Medical history</h2>
      {historyState.loading && <LoadingState />}
      {historyState.error && <ErrorState message={historyState.error} />}

      {historyState.data && !editing && (
        <>
          <dl className="medical-history-grid">
            {MEDICAL_HISTORY_FIELDS.map(({ key, label }) => {
              const values = historyState.data![key]
              return (
                <div key={key} className="medical-history-grid__item">
                  <dt>{label}</dt>
                  <dd>{values.length ? values.join(', ') : <span className="empty-state">None recorded</span>}</dd>
                </div>
              )
            })}
          </dl>
          {isProvider && (
            <button type="button" className="button-link" onClick={startEditing}>
              Edit medical history
            </button>
          )}
        </>
      )}

      {historyState.data && editing && (
        <form className="form" onSubmit={handleSave}>
          {MEDICAL_HISTORY_FIELDS.map(({ key, label }) => (
            <div key={key}>
              <label htmlFor={`mh-${key}`}>{label}</label>
              <input
                id={`mh-${key}`}
                value={fieldText[key]}
                onChange={(e) => setFieldText((prev) => ({ ...prev, [key]: e.target.value }))}
                placeholder="Comma-separated, e.g. Penicillin, Peanuts"
              />
            </div>
          ))}
          {error && <p role="alert" className="form-error">{error}</p>}
          <div className="form--inline">
            <button type="submit" disabled={saving}>
              {saving ? 'Saving…' : 'Save'}
            </button>
            <button type="button" onClick={cancelEditing} disabled={saving}>
              Cancel
            </button>
          </div>
        </form>
      )}
    </section>
  )
}
