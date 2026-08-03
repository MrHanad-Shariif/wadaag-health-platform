import { useState, type FormEvent } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { ErrorState } from '../../shared/StatusMessage'
import { PageHeader } from '../../shared/PageHeader'
import { useToast } from '../../shared/useToast'
import { useAuth } from '../auth/useAuth'
import { bookAppointment } from './api'
import { SlotPicker } from './SlotPicker'
import type { Slot } from './types'

function todayIso(): string {
  const now = new Date()
  const offset = now.getTimezoneOffset()
  return new Date(now.getTime() - offset * 60_000).toISOString().slice(0, 10)
}

// Reached from a provider's profile page with the provider (and their
// facility) already chosen — see the "Book appointment" button on
// ProviderProfilePage — so this page only asks for a date/slot, who the
// appointment is for (patients book only for themselves; clinicians/admins
// booking on someone's behalf enter a patient ID — there's no patient
// search picker yet, same rough edge as the consults/messaging flows), and
// an optional reason.
export function BookAppointmentPage() {
  const navigate = useNavigate()
  const { show } = useToast()
  const { user } = useAuth()
  const [searchParams] = useSearchParams()
  const providerId = searchParams.get('provider')
  const facilityId = searchParams.get('facility')
  const isPatient = user?.role === 'patient'

  const [date, setDate] = useState(todayIso())
  const [selectedSlot, setSelectedSlot] = useState<Slot | null>(null)
  const [patientId, setPatientId] = useState('')
  const [reason, setReason] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  if (!providerId || !facilityId) {
    return <ErrorState message="Missing provider — start booking from a provider's profile page." />
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!selectedSlot) return
    setError(null)
    setSubmitting(true)
    try {
      const appointment = await bookAppointment({
        patient_id: isPatient ? undefined : patientId,
        provider_id: providerId!,
        facility_id: facilityId!,
        start_at: selectedSlot.start_at,
        end_at: selectedSlot.end_at,
        reason: reason || undefined,
      })
      show('Appointment booked')
      navigate(`/appointments/${appointment.id}`)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not book appointment.')
      setSubmitting(false)
    }
  }

  return (
    <div className="page page--narrow">
      <PageHeader eyebrow="Appointments" title="Book an appointment" />

      <form className="form" onSubmit={handleSubmit}>
        <SlotPicker
          providerId={providerId}
          date={date}
          onDateChange={setDate}
          selectedSlot={selectedSlot}
          onSelectSlot={setSelectedSlot}
          minDate={todayIso()}
        />

        {!isPatient && (
          <>
            <label htmlFor="patientId">Patient ID</label>
            <input
              id="patientId"
              value={patientId}
              onChange={(e) => setPatientId(e.target.value)}
              placeholder="Patient's ID"
              required
            />
            <p className="form-hint">Booking on a patient&apos;s behalf — enter their patient ID.</p>
          </>
        )}

        <label htmlFor="reason">Reason (optional)</label>
        <textarea id="reason" value={reason} onChange={(e) => setReason(e.target.value)} rows={3} />

        {error && (
          <p role="alert" className="form-error">
            {error}
          </p>
        )}

        <button type="submit" disabled={submitting || !selectedSlot}>
          {submitting ? 'Booking…' : 'Confirm appointment'}
        </button>
      </form>
    </div>
  )
}
