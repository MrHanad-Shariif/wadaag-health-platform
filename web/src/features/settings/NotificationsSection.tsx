import { useEffect, useState } from 'react'
import { ApiError } from '../../api/client'
import { LoadingState, ErrorState } from '../../shared/StatusMessage'
import { PageHeader } from '../../shared/PageHeader'
import { useFetch } from '../../shared/useFetch'
import { useToast } from '../../shared/useToast'
import { getPreferences, setPreference } from '../notifications/api'

// The backend has no "list every known channel" endpoint — a channel only
// has a notification_preferences row once it's been set at least once (see
// features/notifications/types.ts's NotificationPreference comment). This
// fixed catalog is the frontend's own list of channels worth exposing a
// toggle for; a channel not yet in the fetched list is assumed enabled,
// matching the table's "enabled boolean not null default true" column.
const CHANNEL_CATALOG: { key: string; label: string; description: string }[] = [
  { key: 'in_app', label: 'In-app notifications', description: 'The notification feed shown in the top bar.' },
  { key: 'email', label: 'Email notifications', description: 'Sent to your account email address.' },
  { key: 'push', label: 'Push notifications', description: 'Sent to devices where you have push enabled.' },
]

export function NotificationsSection() {
  const state = useFetch(() => getPreferences(), [])
  const { show } = useToast()
  const [enabledByChannel, setEnabledByChannel] = useState<Record<string, boolean>>({})
  const [savingChannel, setSavingChannel] = useState<string | null>(null)

  useEffect(() => {
    if (!state.data) return
    const next: Record<string, boolean> = {}
    for (const pref of state.data) next[pref.channel] = pref.enabled
    setEnabledByChannel(next)
  }, [state.data])

  async function handleToggle(channel: string) {
    const nextEnabled = !(enabledByChannel[channel] ?? true)
    setSavingChannel(channel)
    // Optimistic update — reverted on failure below.
    setEnabledByChannel((prev) => ({ ...prev, [channel]: nextEnabled }))
    try {
      await setPreference(channel, nextEnabled)
      show(`${nextEnabled ? 'Enabled' : 'Disabled'} ${channel.replace('_', ' ')} notifications`)
    } catch (err) {
      setEnabledByChannel((prev) => ({ ...prev, [channel]: !nextEnabled }))
      show(err instanceof ApiError ? err.message : 'Could not update notification preference.')
    } finally {
      setSavingChannel(null)
    }
  }

  return (
    <div className="page page--narrow">
      <PageHeader
        eyebrow="Settings"
        title="Notifications"
        description="Choose which channels should notify you of new activity."
      />

      {state.loading ? (
        <LoadingState />
      ) : state.error ? (
        <ErrorState message={state.error} />
      ) : (
        <ul className="record-list">
          {CHANNEL_CATALOG.map((channel) => {
            const enabled = enabledByChannel[channel.key] ?? true
            return (
              <li key={channel.key} className="feature-flag-row">
                <div>
                  <strong>{channel.label}</strong>
                  <div className="record-list__note">{channel.description}</div>
                </div>
                <label className="toggle-switch">
                  <input
                    type="checkbox"
                    checked={enabled}
                    disabled={savingChannel === channel.key}
                    onChange={() => handleToggle(channel.key)}
                    aria-label={`Toggle ${channel.label}`}
                  />
                  <span className="toggle-switch__track" aria-hidden="true" />
                </label>
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}
