package appointments

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"github.com/wadaag/health-platform/backend/internal/platform/sqlcgen"
)

var ErrAppointmentNotFound = errors.New("appointment not found")

// ErrAppointmentNotModifiable signals UpdateAppointmentTime's WHERE guard
// rejected the write because the appointment is already cancelled or
// completed. Service.Reschedule always confirms the appointment exists
// (via FindAppointmentByID) before reaching this call, so a no-rows result
// here can only mean the status guard tripped, not a missing row — kept as
// a distinct sentinel from ErrAppointmentNotFound for a clearer message
// (and to guard the rare race where the status changes between that check
// and this update).
var ErrAppointmentNotModifiable = errors.New("appointment cannot be rescheduled in its current status")

type Repository struct {
	q *sqlcgen.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{q: sqlcgen.New(db)}
}

func (r *Repository) CreateAvailability(ctx context.Context, providerID uuid.UUID, weekday int16, startTime, endTime string) (AvailabilityRule, error) {
	start, err := platform.PgTimeOfDay(startTime)
	if err != nil {
		return AvailabilityRule{}, fmt.Errorf("invalid start_time: %w", err)
	}
	end, err := platform.PgTimeOfDay(endTime)
	if err != nil {
		return AvailabilityRule{}, fmt.Errorf("invalid end_time: %w", err)
	}
	row, err := r.q.CreateAvailability(ctx, sqlcgen.CreateAvailabilityParams{
		ProviderID: platform.PgUUID(providerID), Weekday: weekday, StartTime: start, EndTime: end,
	})
	if err != nil {
		return AvailabilityRule{}, fmt.Errorf("insert provider availability: %w", err)
	}
	return availabilityFromRow(row), nil
}

func (r *Repository) ListAvailabilityForProvider(ctx context.Context, providerID uuid.UUID) ([]AvailabilityRule, error) {
	rows, err := r.q.ListAvailabilityForProvider(ctx, platform.PgUUID(providerID))
	if err != nil {
		return nil, fmt.Errorf("list provider availability: %w", err)
	}
	out := make([]AvailabilityRule, len(rows))
	for i, row := range rows {
		out[i] = availabilityFromRow(row)
	}
	return out, nil
}

// DeleteAvailability scopes the delete to (id, providerID) so one provider
// can't delete another's row — see the sqlc query's comment. sqlc's
// generated :exec discards the command tag, so a mismatched id/providerID
// pair silently deletes zero rows rather than erroring; Service.
// DeleteAvailability only calls this once the caller's own provider
// identity has already been resolved, so that's an acceptable no-op rather
// than something callers need to distinguish from "deleted."
func (r *Repository) DeleteAvailability(ctx context.Context, id, providerID uuid.UUID) error {
	if err := r.q.DeleteAvailability(ctx, sqlcgen.DeleteAvailabilityParams{
		ID: platform.PgUUID(id), ProviderID: platform.PgUUID(providerID),
	}); err != nil {
		return fmt.Errorf("delete provider availability: %w", err)
	}
	return nil
}

type CreateAppointmentParams struct {
	PatientID  uuid.UUID
	ProviderID uuid.UUID
	FacilityID uuid.UUID
	StartAt    time.Time
	EndAt      time.Time
	Reason     *string
}

func (r *Repository) CreateAppointment(ctx context.Context, p CreateAppointmentParams) (Appointment, error) {
	row, err := r.q.CreateAppointment(ctx, sqlcgen.CreateAppointmentParams{
		PatientID: platform.PgUUID(p.PatientID), ProviderID: platform.PgUUID(p.ProviderID),
		FacilityID: platform.PgUUID(p.FacilityID), StartAt: platform.PgTimestamptz(p.StartAt),
		EndAt: platform.PgTimestamptz(p.EndAt), Reason: platform.PgText(p.Reason),
	})
	if err != nil {
		return Appointment{}, fmt.Errorf("insert appointment: %w", err)
	}
	return appointmentFromRow(row), nil
}

func (r *Repository) FindAppointmentByID(ctx context.Context, id uuid.UUID) (Appointment, error) {
	row, err := r.q.FindAppointmentByID(ctx, platform.PgUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Appointment{}, ErrAppointmentNotFound
	}
	if err != nil {
		return Appointment{}, fmt.Errorf("query appointment: %w", err)
	}
	return appointmentFromRow(row), nil
}

// ListAppointmentsForProviderInRange returns every non-cancelled
// appointment for providerID overlapping [rangeStart, rangeEnd) — used both
// by generateSlots' overlap filter (see GetAvailableSlots) and a provider's
// own calendar view (see Service.ListForProvider).
func (r *Repository) ListAppointmentsForProviderInRange(ctx context.Context, providerID uuid.UUID, rangeStart, rangeEnd time.Time) ([]Appointment, error) {
	rows, err := r.q.ListAppointmentsForProviderInRange(ctx, sqlcgen.ListAppointmentsForProviderInRangeParams{
		ProviderID: platform.PgUUID(providerID),
		StartAt:    platform.PgTimestamptz(rangeStart),
		EndAt:      platform.PgTimestamptz(rangeEnd),
	})
	if err != nil {
		return nil, fmt.Errorf("list appointments for provider in range: %w", err)
	}
	out := make([]Appointment, len(rows))
	for i, row := range rows {
		out[i] = appointmentFromRow(row)
	}
	return out, nil
}

func (r *Repository) ListAppointmentsForPatient(ctx context.Context, patientID uuid.UUID) ([]Appointment, error) {
	rows, err := r.q.ListAppointmentsForPatient(ctx, platform.PgUUID(patientID))
	if err != nil {
		return nil, fmt.Errorf("list appointments for patient: %w", err)
	}
	out := make([]Appointment, len(rows))
	for i, row := range rows {
		out[i] = appointmentFromRow(row)
	}
	return out, nil
}

func (r *Repository) UpdateAppointmentTime(ctx context.Context, id uuid.UUID, startAt, endAt time.Time) (Appointment, error) {
	row, err := r.q.UpdateAppointmentTime(ctx, sqlcgen.UpdateAppointmentTimeParams{
		ID: platform.PgUUID(id), StartAt: platform.PgTimestamptz(startAt), EndAt: platform.PgTimestamptz(endAt),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Appointment{}, ErrAppointmentNotModifiable
	}
	if err != nil {
		return Appointment{}, fmt.Errorf("update appointment time: %w", err)
	}
	return appointmentFromRow(row), nil
}

func (r *Repository) UpdateAppointmentStatus(ctx context.Context, id uuid.UUID, status Status) (Appointment, error) {
	row, err := r.q.UpdateAppointmentStatus(ctx, sqlcgen.UpdateAppointmentStatusParams{
		ID: platform.PgUUID(id), Status: sqlcgen.AppointmentStatus(status),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Appointment{}, ErrAppointmentNotFound
	}
	if err != nil {
		return Appointment{}, fmt.Errorf("update appointment status: %w", err)
	}
	return appointmentFromRow(row), nil
}

func (r *Repository) ListUpcomingUnreminded(ctx context.Context) ([]Appointment, error) {
	rows, err := r.q.ListUpcomingUnreminded(ctx)
	if err != nil {
		return nil, fmt.Errorf("list upcoming unreminded appointments: %w", err)
	}
	out := make([]Appointment, len(rows))
	for i, row := range rows {
		out[i] = appointmentFromRow(row)
	}
	return out, nil
}

func (r *Repository) MarkReminderSent(ctx context.Context, id uuid.UUID) error {
	if err := r.q.MarkReminderSent(ctx, platform.PgUUID(id)); err != nil {
		return fmt.Errorf("mark reminder sent: %w", err)
	}
	return nil
}

func appointmentFromRow(row sqlcgen.Appointment) Appointment {
	return Appointment{
		ID:             platform.FromPgUUID(row.ID),
		PatientID:      platform.FromPgUUID(row.PatientID),
		ProviderID:     platform.FromPgUUID(row.ProviderID),
		FacilityID:     platform.FromPgUUID(row.FacilityID),
		StartAt:        row.StartAt.Time,
		EndAt:          row.EndAt.Time,
		Status:         Status(row.Status),
		Reason:         platform.FromPgText(row.Reason),
		ReminderSentAt: platform.FromPgTimestamptzPtr(row.ReminderSentAt),
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
	}
}

func availabilityFromRow(row sqlcgen.ProviderAvailability) AvailabilityRule {
	return AvailabilityRule{
		ID:         platform.FromPgUUID(row.ID),
		ProviderID: platform.FromPgUUID(row.ProviderID),
		Weekday:    row.Weekday,
		StartTime:  platform.FromPgTimeOfDay(row.StartTime),
		EndTime:    platform.FromPgTimeOfDay(row.EndTime),
		CreatedAt:  row.CreatedAt.Time,
	}
}
