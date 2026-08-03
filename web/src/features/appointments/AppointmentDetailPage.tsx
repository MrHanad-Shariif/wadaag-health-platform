import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { useFetch } from '../../shared/useFetch'
import { LoadingState, ErrorState } from '../../shared/StatusMessage'
import { PageHeader } from '../../shared/PageHeader'
import { useToast } from '../../shared/useToast'
import { useAuth } from '../auth/useAuth'
import { formatStatus } from './format'
import { cancelAppointment, completeAppointment, getAppointment, markNoShow, rescheduleAppointment } from './api'
import { SlotPicker } from './SlotPicker'
import type { Slot } from './types'

// Non-terminal statuses — the only ones any status-changing action applies
// from.
const OPEN_STATUSES = ['scheduled', 'confirmed']

function todayIso(): string {
  const now = new Date()
  const offset = now.getTimezoneOffset()
  return new Date(now.getTime() - offset * 60_000).toISOString().slice(0, 10)
}

export function AppointmentDetailPage() {
  const { appointmentId } = useParams<{ appointmentId: string }>()
  const { user } = useAuth()
  const { show } = useToast()
  const isPatient = user?.role === 'patient'
  // Backend enforces exactly who may act on a given appointment (the
  // patient it belongs to, the provider it's assigned to, or an admin) —
  // the frontend just shows the buttons for roles that could plausibly be
  // allowed, same convention as ReferralDetailPage's isProvider gate.
  const isProviderSide = user?.role === 'physician' || user?.role === 'hospital_admin' || user?.role === 'system_admin'

  const appointmentState = useFetch(() => getAppointment(appointmentId!), [appointmentId])

  const [actionError, setActionError] = useState<string | null>(null)
  const [running, setRunning] = useState(false)
  const [rescheduling, setRescheduling] = useState(false)
  const [rescheduleDate, setRescheduleDate] = useState('')
  const [rescheduleSlot, setRescheduleSlot] = useState<Slot | null>(null)

  async function runAction(action: (id: string) => Promise<unknown>, toastMessage: string) {
    setActionError(null)
    setRunning(true)
    try {
      await action(appointmentId!)
      appointmentState.reload()
      show(toastMessage)
    } catch (err) {
      setActionError(err instanceof ApiError ? err.message : 'Action failed.')
    } finally {
      setRunning(false)
    }
  }

  function startReschedule() {
    if (!appointmentState.data) return
    setActionError(null)
    setRescheduleDate(appointmentState.data.start_at.slice(0, 10))
    setRescheduleSlot(null)
    setRescheduling(true)
  }

  async function confirmReschedule() {
    if (!rescheduleSlot) return
    setActionError(null)
    setRunning(true)
    try {
      await rescheduleAppointment(appointmentId!, rescheduleSlot.start_at, rescheduleSlot.end_at)
      appointmentState.reload()
      setRescheduling(false)
      show('Appointment rescheduled')
    } catch (err) {
      setActionError(err instanceof ApiError ? err.message : 'Could not reschedule appointment.')
    } finally {
      setRunning(false)
    }
  }

  if (appointmentState.loading) return <LoadingState />
  if (appointmentState.error) return <ErrorState message={appointmentState.error} />
  const appointment = appointmentState.data!

  const isOpen = OPEN_STATUSES.includes(appointment.status)
  const canCancel = isOpen && (isPatient || isProviderSide)
  const canReschedule = isOpen && (isPatient || isProviderSide)
  const canManageAsProvider = isOpen && isProviderSide

  return (
    <div className="page">
      <PageHeader
        eyebrow="Appointment"
        title={new Date(appointment.start_at).toLocaleString()}
        description={appointment.reason}
      />

      <p>
        <span className={`status-pill status-pill--${appointment.status}`}>{formatStatus(appointment.status)}</span>
      </p>

      <dl className="detail-grid">
        <dt>Start</dt>
        <dd>{new Date(appointment.start_at).toLocaleString()}</dd>
        <dt>End</dt>
        <dd>{new Date(appointment.end_at).toLocaleString()}</dd>
        <dt>Patient</dt>
        <dd>{appointment.patient_id}</dd>
        <dt>Provider</dt>
        <dd>{appointment.provider_id}</dd>
        <dt>Facility</dt>
        <dd>{appointment.facility_id}</dd>
      </dl>

      {(canCancel || canReschedule || canManageAsProvider) && (
        <div className="action-row">
          {canReschedule && !rescheduling && (
            <button type="button" className="button" disabled={running} onClick={startReschedule}>
              Reschedule
            </button>
          )}
          {canManageAsProvider && (
            <button
              type="button"
              className="button button--primary"
              disabled={running}
              onClick={() => runAction(completeAppointment, 'Appointment marked completed')}
            >
              Complete
            </button>
          )}
          {canManageAsProvider && (
            <button
              type="button"
              className="button"
              disabled={running}
              onClick={() => runAction(markNoShow, 'Appointment marked as no-show')}
            >
              Mark no-show
            </button>
          )}
          {canCancel && (
            <button
              type="button"
              className="button button--outline-danger"
              disabled={running}
              onClick={() => runAction(cancelAppointment, 'Appointment cancelled')}
            >
              Cancel appointment
            </button>
          )}
        </div>
      )}

      {rescheduling && (
        <section>
          <h2>Reschedule</h2>
          <SlotPicker
            providerId={appointment.provider_id}
            date={rescheduleDate}
            onDateChange={setRescheduleDate}
            selectedSlot={rescheduleSlot}
            onSelectSlot={setRescheduleSlot}
            minDate={todayIso()}
          />
          <div className="action-row">
            <button type="button" className="button button--primary" disabled={running || !rescheduleSlot} onClick={confirmReschedule}>
              {running ? 'Saving…' : 'Confirm new time'}
            </button>
            <button type="button" className="button" disabled={running} onClick={() => setRescheduling(false)}>
              Cancel
            </button>
          </div>
        </section>
      )}

      {actionError && <ErrorState message={actionError} />}
    </div>
  )
}
