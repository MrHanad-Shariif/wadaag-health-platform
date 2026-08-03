import { useState, type FormEvent } from 'react'
import { ApiError } from '../../api/client'
import { LoadingState, ErrorState } from '../../shared/StatusMessage'
import { PageHeader } from '../../shared/PageHeader'
import { useFetch } from '../../shared/useFetch'
import { useToast } from '../../shared/useToast'
import { getOwnProviderProfile } from '../providers/api'
import { deleteAvailabilityRule, getAvailability, setAvailability } from './api'
import { WEEKDAY_LABELS } from './format'

// A physician's own weekly schedule: a flat list of "this weekday, this
// time range" rules rather than a drag-to-select calendar grid — matches
// what the backend's one-rule-per-call POST endpoint expects. Reached from
// "/providers/me" (see the link added there) since it operates purely on
// the signed-in physician's own provider profile.
export function AvailabilityEditorPage() {
  const { show } = useToast()

  const providerState = useFetch(() => getOwnProviderProfile(), [])
  const providerId = providerState.data?.id

  const rulesState = useFetch(
    () => (providerId ? getAvailability(providerId) : Promise.resolve([])),
    [providerId],
  )

  const [weekday, setWeekday] = useState(1)
  const [startTime, setStartTime] = useState('09:00')
  const [endTime, setEndTime] = useState('17:00')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [deletingId, setDeletingId] = useState<string | null>(null)

  async function handleAddRule(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      await setAvailability({ weekday, start_time: startTime, end_time: endTime })
      rulesState.reload()
      show('Availability rule added')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not add availability rule.')
    } finally {
      setSubmitting(false)
    }
  }

  async function handleDelete(ruleId: string) {
    setDeletingId(ruleId)
    try {
      await deleteAvailabilityRule(ruleId)
      rulesState.reload()
      show('Availability rule removed')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not remove availability rule.')
    } finally {
      setDeletingId(null)
    }
  }

  if (providerState.loading) return <LoadingState />
  if (providerState.error) return <ErrorState message={providerState.error} />

  const rules = [...(rulesState.data ?? [])].sort((a, b) => a.weekday - b.weekday || a.start_time.localeCompare(b.start_time))

  return (
    <div className="page page--narrow">
      <PageHeader
        breadcrumb={[{ label: 'My provider profile', to: '/providers/me' }, { label: 'Weekly availability' }]}
        title="Weekly availability"
        description="Set the recurring weekly hours patients can book you for. Add one rule per weekday you're available."
      />

      <form className="form form--inline" onSubmit={handleAddRule}>
        <label htmlFor="weekday">Weekday</label>
        <select id="weekday" value={weekday} onChange={(e) => setWeekday(Number(e.target.value))}>
          {WEEKDAY_LABELS.map((label, i) => (
            <option key={i} value={i}>
              {label}
            </option>
          ))}
        </select>

        <label htmlFor="startTime">Start</label>
        <input id="startTime" type="time" value={startTime} onChange={(e) => setStartTime(e.target.value)} required />

        <label htmlFor="endTime">End</label>
        <input id="endTime" type="time" value={endTime} onChange={(e) => setEndTime(e.target.value)} required />

        <button type="submit" disabled={submitting}>
          {submitting ? 'Adding…' : 'Add rule'}
        </button>
      </form>

      {error && <ErrorState message={error} />}

      <section>
        <h2>Current rules</h2>
        {rulesState.loading && <LoadingState />}
        {rulesState.error && <ErrorState message={rulesState.error} />}
        {rulesState.data && (
          rules.length > 0 ? (
            <ul className="availability-rule-list">
              {rules.map((rule) => (
                <li key={rule.id}>
                  <span>
                    {WEEKDAY_LABELS[rule.weekday] ?? `Day ${rule.weekday}`}: {rule.start_time}–{rule.end_time}
                  </span>
                  <button
                    type="button"
                    className="button button--outline-danger"
                    disabled={deletingId === rule.id}
                    onClick={() => handleDelete(rule.id)}
                  >
                    {deletingId === rule.id ? 'Removing…' : 'Remove'}
                  </button>
                </li>
              ))}
            </ul>
          ) : (
            <p className="empty-state">No availability rules set yet.</p>
          )
        )}
      </section>
    </div>
  )
}
