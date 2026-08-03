-- name: CreateAvailability :one
INSERT INTO provider_availability (provider_id, weekday, start_time, end_time)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListAvailabilityForProvider :many
SELECT * FROM provider_availability
WHERE provider_id = $1
ORDER BY weekday, start_time;

-- name: DeleteAvailability :exec
-- Scoped to the owning provider so one provider can't delete another's
-- availability row — see appointments.Service.DeleteAvailability.
DELETE FROM provider_availability WHERE id = $1 AND provider_id = $2;

-- name: CreateAppointment :one
INSERT INTO appointments (patient_id, provider_id, facility_id, start_at, end_at, reason)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: FindAppointmentByID :one
SELECT * FROM appointments WHERE id = $1;

-- name: ListAppointmentsForProviderInRange :many
-- Overlap check: any appointment that isn't cancelled whose interval
-- intersects [range_start, range_end). Used both for slot-generation's
-- overlap filter and for a provider's own calendar view.
SELECT * FROM appointments
WHERE provider_id = $1
  AND start_at < $3
  AND end_at > $2
  AND status != 'cancelled'
ORDER BY start_at;

-- name: ListAppointmentsForPatient :many
SELECT * FROM appointments WHERE patient_id = $1 ORDER BY start_at DESC;

-- name: UpdateAppointmentTime :one
-- Reschedule: sets status back to 'scheduled' unless it was already
-- cancelled/completed, in which case the WHERE guard means zero rows
-- match and the repository surfaces ErrAppointmentNotFound.
UPDATE appointments
SET start_at = $2,
    end_at = $3,
    status = 'scheduled',
    updated_at = now()
WHERE id = $1
  AND status NOT IN ('cancelled', 'completed')
RETURNING *;

-- name: UpdateAppointmentStatus :one
UPDATE appointments
SET status = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ListUpcomingUnreminded :many
SELECT * FROM appointments
WHERE status IN ('scheduled', 'confirmed')
  AND reminder_sent_at IS NULL
  AND start_at BETWEEN now() AND now() + interval '24 hours';

-- name: MarkReminderSent :exec
UPDATE appointments SET reminder_sent_at = now() WHERE id = $1;
