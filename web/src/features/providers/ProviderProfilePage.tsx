import { useEffect, useState, type FormEvent } from 'react'
import { useParams } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { useFetch } from '../../shared/useFetch'
import { LoadingState, ErrorState } from '../../shared/StatusMessage'
import { PageHeader } from '../../shared/PageHeader'
import { useToast } from '../../shared/useToast'
import { useAuth } from '../auth/useAuth'
import { getOwnProviderProfile, getProvider, updateOwnProviderProfile, updateProviderVerificationStatus } from './api'
import { VERIFICATION_STATUSES, type Provider, type UpdateProviderProfileInput, type VerificationStatus } from './types'

type ArrayField = 'qualifications' | 'languages' | 'areas_of_expertise'

const ARRAY_FIELDS: { key: ArrayField; label: string; placeholder: string }[] = [
  { key: 'qualifications', label: 'Qualifications', placeholder: 'MBBS, MD Internal Medicine' },
  { key: 'languages', label: 'Languages', placeholder: 'en, so, ar' },
  { key: 'areas_of_expertise', label: 'Areas of expertise', placeholder: 'Cardiology, Pediatrics' },
]

function statusPillClass(status: string): string {
  if (status === 'verified') return 'status-pill status-pill--completed'
  if (status === 'rejected') return 'status-pill status-pill--declined'
  return 'status-pill status-pill--created'
}

// Serves two routes: "/providers/me" (own, self-editable profile) and
// "/providers/:providerId" (read-only view of any provider, plus an
// admin-only verification-status control). Which mode we're in is purely a
// function of whether providerId is present in the URL.
export function ProviderProfilePage() {
  const { providerId } = useParams<{ providerId: string }>()
  const { user } = useAuth()
  const { show } = useToast()
  const selfMode = !providerId
  const canModerate = user?.role === 'system_admin' || user?.role === 'hospital_admin'

  const providerState = useFetch(() => (selfMode ? getOwnProviderProfile() : getProvider(providerId!)), [selfMode, providerId])

  if (providerState.loading) return <LoadingState />
  if (providerState.error) return <ErrorState message={providerState.error} />
  const provider = providerState.data
  if (!provider) return null

  return (
    <div className="page page--narrow">
      <PageHeader
        eyebrow="Provider"
        title={selfMode ? 'My provider profile' : 'Provider profile'}
        description={provider.specialty ?? undefined}
      />

      <dl className="detail-grid">
        <dt>Specialty</dt>
        <dd>{provider.specialty ?? <span className="empty-state">Not set</span>}</dd>
        <dt>License number</dt>
        <dd>{provider.license_number ?? <span className="empty-state">Not set</span>}</dd>
        <dt>Verification status</dt>
        <dd>
          <span className={statusPillClass(provider.verification_status)}>{provider.verification_status}</span>
        </dd>
      </dl>

      {canModerate && !selfMode && (
        <VerificationStatusControl
          provider={provider}
          onSaved={() => {
            providerState.reload()
            show('Verification status updated')
          }}
        />
      )}

      {selfMode ? (
        <ProfileEditSection provider={provider} onSaved={providerState.reload} />
      ) : (
        <ProfileReadOnlySection provider={provider} />
      )}
    </div>
  )
}

function VerificationStatusControl({ provider, onSaved }: { provider: Provider; onSaved: () => void }) {
  const [status, setStatus] = useState<VerificationStatus>((provider.verification_status as VerificationStatus) ?? 'pending')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    setStatus((provider.verification_status as VerificationStatus) ?? 'pending')
  }, [provider.verification_status])

  async function handleSave(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSaving(true)
    try {
      await updateProviderVerificationStatus(provider.id, status)
      onSaved()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not update verification status.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <section>
      <h2>Verification (admin)</h2>
      <form className="form form--inline" onSubmit={handleSave}>
        <label htmlFor="verificationStatus">Status</label>
        <select
          id="verificationStatus"
          value={status}
          onChange={(e) => setStatus(e.target.value as VerificationStatus)}
        >
          {VERIFICATION_STATUSES.map((s) => (
            <option key={s} value={s}>
              {s.charAt(0).toUpperCase() + s.slice(1)}
            </option>
          ))}
        </select>
        <button type="submit" disabled={saving || status === provider.verification_status}>
          {saving ? 'Saving…' : 'Update status'}
        </button>
        {error && (
          <p role="alert" className="form-error">
            {error}
          </p>
        )}
      </form>
    </section>
  )
}

function ProfileReadOnlySection({ provider }: { provider: Provider }) {
  return (
    <>
      <section>
        <h2>Profile</h2>
        <dl className="detail-grid">
          <dt>Years of experience</dt>
          <dd>{provider.years_experience ?? <span className="empty-state">Not set</span>}</dd>
          <dt>Consultation fee</dt>
          <dd>{provider.consultation_fee ?? <span className="empty-state">Not set</span>}</dd>
        </dl>
        {ARRAY_FIELDS.map(({ key, label }) => {
          const values = provider[key] ?? []
          return (
            <dl key={key} className="detail-grid">
              <dt>{label}</dt>
              <dd>{values.length ? values.join(', ') : <span className="empty-state">None recorded</span>}</dd>
            </dl>
          )
        })}
      </section>
      <CertificatesSection provider={provider} />
    </>
  )
}

function ProfileEditSection({ provider, onSaved }: { provider: Provider; onSaved: () => void }) {
  const { show } = useToast()
  const [editing, setEditing] = useState(false)
  const [fieldText, setFieldText] = useState<Record<ArrayField, string>>({
    qualifications: '',
    languages: '',
    areas_of_expertise: '',
  })
  const [yearsExperience, setYearsExperience] = useState('')
  const [consultationFee, setConsultationFee] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  function fillFromProvider() {
    const next = {} as Record<ArrayField, string>
    for (const { key } of ARRAY_FIELDS) {
      next[key] = (provider[key] ?? []).join(', ')
    }
    setFieldText(next)
    setYearsExperience(provider.years_experience != null ? String(provider.years_experience) : '')
    setConsultationFee(provider.consultation_fee ?? '')
  }

  useEffect(() => {
    fillFromProvider()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [provider])

  function startEditing() {
    setError(null)
    fillFromProvider()
    setEditing(true)
  }

  function cancelEditing() {
    setEditing(false)
    setError(null)
    fillFromProvider()
  }

  async function handleSave(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSaving(true)
    try {
      const input: UpdateProviderProfileInput = {}
      for (const { key } of ARRAY_FIELDS) {
        input[key] = (fieldText[key] ?? '')
          .split(',')
          .map((v) => v.trim())
          .filter(Boolean)
      }
      if (yearsExperience.trim() !== '') {
        const n = Number(yearsExperience)
        if (!Number.isNaN(n)) input.years_experience = n
      }
      if (consultationFee.trim() !== '') {
        input.consultation_fee = consultationFee.trim()
      }
      await updateOwnProviderProfile(input)
      onSaved()
      setEditing(false)
      show('Provider profile updated')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not update provider profile.')
    } finally {
      setSaving(false)
    }
  }

  if (!editing) {
    return (
      <>
        <section>
          <h2>Profile</h2>
          <dl className="detail-grid">
            <dt>Years of experience</dt>
            <dd>{provider.years_experience ?? <span className="empty-state">Not set</span>}</dd>
            <dt>Consultation fee</dt>
            <dd>{provider.consultation_fee ?? <span className="empty-state">Not set</span>}</dd>
          </dl>
          {ARRAY_FIELDS.map(({ key, label }) => {
            const values = provider[key] ?? []
            return (
              <dl key={key} className="detail-grid">
                <dt>{label}</dt>
                <dd>{values.length ? values.join(', ') : <span className="empty-state">None recorded</span>}</dd>
              </dl>
            )
          })}
          <button type="button" className="button-link" onClick={startEditing}>
            Edit profile
          </button>
        </section>
        <CertificatesSection provider={provider} />
      </>
    )
  }

  return (
    <>
      <section>
        <h2>Profile</h2>
        <form className="form" onSubmit={handleSave}>
          <label htmlFor="yearsExperience">Years of experience</label>
          <input
            id="yearsExperience"
            type="number"
            min={0}
            value={yearsExperience}
            onChange={(e) => setYearsExperience(e.target.value)}
            placeholder="e.g. 8"
          />

          <label htmlFor="consultationFee">Consultation fee</label>
          <input
            id="consultationFee"
            inputMode="decimal"
            value={consultationFee}
            onChange={(e) => setConsultationFee(e.target.value)}
            placeholder="e.g. 15.00"
          />

          {ARRAY_FIELDS.map(({ key, label, placeholder }) => (
            <div key={key}>
              <label htmlFor={`provider-${key}`}>{label}</label>
              <input
                id={`provider-${key}`}
                value={fieldText[key]}
                onChange={(e) => setFieldText((prev) => ({ ...prev, [key]: e.target.value }))}
                placeholder={`Comma-separated, e.g. ${placeholder}`}
              />
            </div>
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
            <button type="button" onClick={cancelEditing} disabled={saving}>
              Cancel
            </button>
          </div>
        </form>
      </section>
      <CertificatesSection provider={provider} />
    </>
  )
}

// Read-only for now — full certificate upload/edit UI is out of scope until
// Phase 3's attachments module lands (see the certificates column comment in
// backend/db/migrations/0014_extend_providers.up.sql).
function CertificatesSection({ provider }: { provider: Provider }) {
  const certificates = provider.certificates ?? []

  return (
    <section>
      <h2>Certificates</h2>
      {certificates.length ? (
        <ul className="record-list">
          {certificates.map((cert, i) => (
            <li key={i}>
              {cert.name ?? 'Certificate'}
              {cert.issuer && ` — ${cert.issuer}`}
              {cert.year && ` (${cert.year})`}
              {cert.url && (
                <>
                  {' '}
                  <a href={cert.url} target="_blank" rel="noreferrer">
                    View
                  </a>
                </>
              )}
            </li>
          ))}
        </ul>
      ) : (
        <p className="empty-state">No certificates recorded.</p>
      )}
    </section>
  )
}
