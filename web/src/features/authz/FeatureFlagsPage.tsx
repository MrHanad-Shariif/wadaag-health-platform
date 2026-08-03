import { useState, type FormEvent } from 'react'
import { ApiError } from '../../api/client'
import { useFetch } from '../../shared/useFetch'
import { LoadingState, ErrorState } from '../../shared/StatusMessage'
import { PageHeader } from '../../shared/PageHeader'
import { useToast } from '../../shared/useToast'
import { listFeatureFlags, setFeatureFlag } from './api'
import type { FeatureFlag } from './types'

// system_admin only — the underlying endpoint 403s for every other role
// (see backend/internal/authz/handler.go's feature-flags route group).
export function FeatureFlagsPage() {
  const state = useFetch(() => listFeatureFlags(), [])
  const { show } = useToast()
  const [togglingKey, setTogglingKey] = useState<string | null>(null)

  async function handleToggle(flag: FeatureFlag) {
    setTogglingKey(flag.key)
    try {
      await setFeatureFlag(flag.key, { enabled: !flag.enabled, description: flag.description })
      state.reload()
      show(`"${flag.key}" ${!flag.enabled ? 'enabled' : 'disabled'}`)
    } catch (err) {
      show(err instanceof ApiError ? err.message : 'Could not update feature flag.')
    } finally {
      setTogglingKey(null)
    }
  }

  return (
    <div className="page page--narrow">
      <PageHeader
        eyebrow="Admin"
        title="Feature flags"
        description="Platform-wide toggles. Changes take effect immediately for every user."
      />

      {state.loading ? (
        <LoadingState />
      ) : state.error ? (
        <ErrorState message={state.error} />
      ) : state.data && state.data.length > 0 ? (
        <ul className="record-list">
          {state.data.map((flag) => (
            <li key={flag.key} className="feature-flag-row">
              <div>
                <strong>{flag.key}</strong>
                {flag.description && <div className="record-list__note">{flag.description}</div>}
                <div className="record-list__timestamp">Updated {new Date(flag.updated_at).toLocaleString()}</div>
              </div>
              <label className="toggle-switch">
                <input
                  type="checkbox"
                  checked={flag.enabled}
                  disabled={togglingKey === flag.key}
                  onChange={() => handleToggle(flag)}
                  aria-label={`Toggle ${flag.key}`}
                />
                <span className="toggle-switch__track" aria-hidden="true" />
              </label>
            </li>
          ))}
        </ul>
      ) : (
        <p className="empty-state">No feature flags defined yet.</p>
      )}

      <NewFlagForm onCreated={() => state.reload()} />
    </div>
  )
}

function NewFlagForm({ onCreated }: { onCreated: () => void }) {
  const { show } = useToast()
  const [key, setKey] = useState('')
  const [description, setDescription] = useState('')
  const [enabled, setEnabled] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    const trimmedKey = key.trim()
    if (!trimmedKey) {
      setError('Key is required.')
      return
    }
    setSubmitting(true)
    try {
      await setFeatureFlag(trimmedKey, { enabled, description: description.trim() || undefined })
      setKey('')
      setDescription('')
      setEnabled(false)
      onCreated()
      show('Feature flag added')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not create feature flag.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form className="form" onSubmit={handleSubmit}>
      <h2 className="form-section-title">Add a new flag</h2>

      <label htmlFor="flagKey">Key</label>
      <input
        id="flagKey"
        value={key}
        onChange={(e) => setKey(e.target.value)}
        placeholder="e.g. new_referral_flow"
        required
      />

      <label htmlFor="flagDescription">Description</label>
      <input
        id="flagDescription"
        value={description}
        onChange={(e) => setDescription(e.target.value)}
        placeholder="What does this flag control?"
      />

      <label className="checkbox-row">
        <input type="checkbox" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />
        Enabled from creation
      </label>

      {error && (
        <p role="alert" className="form-error">
          {error}
        </p>
      )}

      <button type="submit" disabled={submitting}>
        {submitting ? 'Adding…' : 'Add flag'}
      </button>
    </form>
  )
}
