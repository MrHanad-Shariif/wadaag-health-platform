package appointments

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/facilities"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"github.com/wadaag/health-platform/backend/internal/records"
)

// ProviderResolver maps the calling user to their provider identity. Same
// interface shape as referrals.ProviderResolver/consults.ProviderResolver —
// satisfied by *facilities.Service, wired in main.go.
type ProviderResolver interface {
	ResolveProviderID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

// ProviderLookup resolves a provider profile by id — used to validate a
// caller-supplied provider_id refers to a real provider (Book) and to
// resolve a provider's home facility (SetAvailability's hospital_admin
// path). Satisfied by *facilities.Service.
type ProviderLookup interface {
	GetProvider(ctx context.Context, id uuid.UUID) (facilities.Provider, error)
}

// NotificationPublisher is the narrow surface this package needs from
// notifications.Service, defined locally so appointments doesn't import
// that package directly. Optional (nil by default); set via
// SetNotificationPublisher. Satisfied by *notifications.Service.
type NotificationPublisher interface {
	Publish(ctx context.Context, userID uuid.UUID, notifType string, payload map[string]any) error
}

var (
	// ErrNotAuthorized covers reschedule/cancel/complete/no-show: only the
	// booking patient, the assigned provider, or an administrator may act.
	ErrNotAuthorized     = errors.New("not authorized to act on this appointment")
	ErrSlotUnavailable   = errors.New("slot no longer available")
	ErrPatientIDRequired = errors.New("patient_id is required")
	ErrProviderIDInvalid = errors.New("provider_id must refer to an existing provider")
	ErrInvalidTimeRange  = errors.New("end_at must be after start_at")
	ErrInvalidWeekday    = errors.New("weekday must be between 0 (Sunday) and 6 (Saturday)")
	ErrInvalidTimeOfDay  = errors.New("time must be in HH:MM form")
	// ErrProviderIDRequired covers SetAvailability's hospital_admin/
	// system_admin path, where — unlike a physician acting on their own
	// schedule — there is no "caller's own provider" to default to.
	ErrProviderIDRequired = errors.New("provider_id is required")
)

type Service struct {
	repo      *Repository
	provider  ProviderResolver
	providers ProviderLookup
	records   *records.Service

	// notify is optional (nil by default) — set via SetNotificationPublisher
	// in main.go. Nil-tolerant wherever it's used, so existing construction
	// paths and tests keep working unchanged.
	notify NotificationPublisher
}

func NewService(repo *Repository, provider ProviderResolver, providers ProviderLookup, recordsService *records.Service) *Service {
	return &Service{repo: repo, provider: provider, providers: providers, records: recordsService}
}

// SetNotificationPublisher wires the optional in-app notification publisher.
func (s *Service) SetNotificationPublisher(p NotificationPublisher) { s.notify = p }

// ResolveProviderID exposes the caller's own provider identity — used by the
// handler's /mine and reschedule/cancel/complete/no-show paths.
func (s *Service) ResolveProviderID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	return s.provider.ResolveProviderID(ctx, userID)
}

// ResolveOwnPatientID resolves a patient-role caller's own patient_id —
// used by the handler's /mine and booking paths.
func (s *Service) ResolveOwnPatientID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	patient, err := s.records.GetOwnPatientRecord(ctx, userID)
	if err != nil {
		return uuid.UUID{}, err
	}
	return patient.ID, nil
}

// resolveManageableProviderID decides whose availability an availability
// write (SetAvailability) applies to:
//   - a physician manages only their own schedule — targetProviderID is
//     ignored even if supplied, matching how a patient can never supply
//     their own patient_id for booking (see resolveBookingPatientID).
//   - hospital_admin/system_admin act on behalf of a named provider, so
//     targetProviderID is required; hospital_admin is further restricted to
//     a provider at their own facility (system_admin has no facility of its
//     own and is never restricted, consistent with RequireRoles' blanket
//     system_admin bypass elsewhere in the codebase).
//
// This is a judgment call beyond what the design doc specified explicitly:
// it seemed safer for a hospital_admin write to be facility-scoped than to
// let them edit any provider's schedule platform-wide.
func (s *Service) resolveManageableProviderID(ctx context.Context, actorRole platform.Role, actorUserID uuid.UUID, actorFacilityID *uuid.UUID, targetProviderID *uuid.UUID) (uuid.UUID, error) {
	if actorRole == platform.RolePhysician {
		return s.provider.ResolveProviderID(ctx, actorUserID)
	}

	if targetProviderID == nil {
		return uuid.UUID{}, ErrProviderIDRequired
	}
	provider, err := s.providers.GetProvider(ctx, *targetProviderID)
	if err != nil {
		if errors.Is(err, facilities.ErrProviderNotFound) {
			return uuid.UUID{}, ErrProviderIDInvalid
		}
		return uuid.UUID{}, fmt.Errorf("resolve provider: %w", err)
	}
	if actorRole == platform.RoleHospitalAdmin {
		if actorFacilityID == nil || provider.FacilityID != *actorFacilityID {
			return uuid.UUID{}, ErrNotAuthorized
		}
	}
	return provider.ID, nil
}

// SetAvailability creates one weekly recurring availability window. See
// resolveManageableProviderID for who may set availability for which
// provider.
func (s *Service) SetAvailability(ctx context.Context, actorUserID uuid.UUID, actorRole platform.Role, actorFacilityID *uuid.UUID, targetProviderID *uuid.UUID, weekday int16, startTime, endTime string) (AvailabilityRule, error) {
	providerID, err := s.resolveManageableProviderID(ctx, actorRole, actorUserID, actorFacilityID, targetProviderID)
	if err != nil {
		return AvailabilityRule{}, err
	}

	if weekday < 0 || weekday > 6 {
		return AvailabilityRule{}, ErrInvalidWeekday
	}
	start, err := time.Parse("15:04", startTime)
	if err != nil {
		return AvailabilityRule{}, fmt.Errorf("%w: start_time", ErrInvalidTimeOfDay)
	}
	end, err := time.Parse("15:04", endTime)
	if err != nil {
		return AvailabilityRule{}, fmt.Errorf("%w: end_time", ErrInvalidTimeOfDay)
	}
	if !end.After(start) {
		return AvailabilityRule{}, ErrInvalidTimeRange
	}

	return s.repo.CreateAvailability(ctx, providerID, weekday, startTime, endTime)
}

func (s *Service) ListAvailability(ctx context.Context, providerID uuid.UUID) ([]AvailabilityRule, error) {
	return s.repo.ListAvailabilityForProvider(ctx, providerID)
}

// DeleteAvailability is the owning provider only — no hospital_admin/
// system_admin delegation, unlike SetAvailability (see the handler's
// Routes doc comment for why this endpoint is more restrictive).
func (s *Service) DeleteAvailability(ctx context.Context, actorUserID, ruleID uuid.UUID) error {
	providerID, err := s.provider.ResolveProviderID(ctx, actorUserID)
	if err != nil {
		return fmt.Errorf("resolve provider identity: %w", err)
	}
	return s.repo.DeleteAvailability(ctx, ruleID, providerID)
}

// GetAvailableSlots looks up providerID's recurring availability for date's
// weekday and the appointments already booked that day, then delegates to
// the pure generateSlots to do the actual math — see that function's doc
// comment for the algorithm.
func (s *Service) GetAvailableSlots(ctx context.Context, providerID uuid.UUID, date time.Time, slotMinutes int) ([]Slot, error) {
	rules, err := s.repo.ListAvailabilityForProvider(ctx, providerID)
	if err != nil {
		return nil, err
	}

	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	dayEnd := dayStart.Add(24 * time.Hour)
	existing, err := s.repo.ListAppointmentsForProviderInRange(ctx, providerID, dayStart, dayEnd)
	if err != nil {
		return nil, err
	}

	return generateSlots(rules, existing, date, slotMinutes), nil
}

// generateSlots is the entire slot-generation algorithm, deliberately kept
// pure (no DB, no clock) so it's unit-testable directly — see
// service_test.go. For every availability rule matching date's weekday, it
// walks fixed-slotMinutes-duration windows from the rule's start to end
// time, dropping any slot whose half-open interval [start, end) overlaps an
// existing (already-fetched, non-cancelled) appointment. A slot exactly
// filling the remainder of the window is kept (equal end times don't
// count as "past the end"); a slot that would run past the window's end is
// dropped entirely rather than truncated.
func generateSlots(rules []AvailabilityRule, existing []Appointment, date time.Time, slotMinutes int) []Slot {
	if slotMinutes <= 0 {
		slotMinutes = 30
	}
	duration := time.Duration(slotMinutes) * time.Minute
	weekday := int16(date.Weekday())

	var slots []Slot
	for _, rule := range rules {
		if rule.Weekday != weekday {
			continue
		}
		windowStart, err := timeOfDayOn(date, rule.StartTime)
		if err != nil {
			continue
		}
		windowEnd, err := timeOfDayOn(date, rule.EndTime)
		if err != nil {
			continue
		}

		for slotStart := windowStart; !slotStart.Add(duration).After(windowEnd); slotStart = slotStart.Add(duration) {
			slotEnd := slotStart.Add(duration)
			if overlapsAny(slotStart, slotEnd, existing) {
				continue
			}
			slots = append(slots, Slot{StartAt: slotStart, EndAt: slotEnd})
		}
	}
	return slots
}

// overlapsAny reports whether [start, end) intersects any non-cancelled
// appointment's own [a.StartAt, a.EndAt) — the standard half-open-interval
// overlap test, which is also why back-to-back slots/appointments (one
// ending exactly when the next starts) don't count as overlapping.
func overlapsAny(start, end time.Time, existing []Appointment) bool {
	for _, a := range existing {
		if start.Before(a.EndAt) && end.After(a.StartAt) {
			return true
		}
	}
	return false
}

// timeOfDayOn combines date's year/month/day with an "HH:MM" time-of-day
// string, in date's own location.
func timeOfDayOn(date time.Time, hhmm string) (time.Time, error) {
	t, err := time.Parse("15:04", hhmm)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(date.Year(), date.Month(), date.Day(), t.Hour(), t.Minute(), 0, 0, date.Location()), nil
}

// BookInput is Service.Book's input, pre-decoded/parsed by the handler.
type BookInput struct {
	// PatientID is only read for a physician/hospital_admin/system_admin
	// caller — a patient-role caller's own patient_id is always resolved
	// server-side (see resolveBookingPatientID) and this field ignored, so
	// a patient can never book on someone else's behalf.
	PatientID  *uuid.UUID
	ProviderID uuid.UUID
	FacilityID uuid.UUID
	StartAt    time.Time
	EndAt      time.Time
	Reason     *string
}

// Book validates the requested window and inserts the appointment. The
// slot's availability is re-checked here against the live table (not just
// trusting whatever GetAvailableSlots showed the caller a moment earlier),
// closing the race between two people booking the same slot.
func (s *Service) Book(ctx context.Context, actorUserID uuid.UUID, actorRole platform.Role, in BookInput) (Appointment, error) {
	if !in.EndAt.After(in.StartAt) {
		return Appointment{}, ErrInvalidTimeRange
	}

	patientID, err := s.resolveBookingPatientID(ctx, actorUserID, actorRole, in.PatientID)
	if err != nil {
		return Appointment{}, err
	}

	if _, err := s.providers.GetProvider(ctx, in.ProviderID); err != nil {
		if errors.Is(err, facilities.ErrProviderNotFound) {
			return Appointment{}, ErrProviderIDInvalid
		}
		return Appointment{}, fmt.Errorf("resolve provider: %w", err)
	}

	if err := s.checkSlotFree(ctx, in.ProviderID, in.StartAt, in.EndAt, nil); err != nil {
		return Appointment{}, err
	}

	appointment, err := s.repo.CreateAppointment(ctx, CreateAppointmentParams{
		PatientID: patientID, ProviderID: in.ProviderID, FacilityID: in.FacilityID,
		StartAt: in.StartAt, EndAt: in.EndAt, Reason: in.Reason,
	})
	if err != nil {
		return Appointment{}, err
	}
	return appointment, nil
}

// resolveBookingPatientID never trusts a patient-supplied patient_id — a
// patient-role caller always gets their own patient_id resolved server-side
// instead of whatever (if anything) they put in the request body.
func (s *Service) resolveBookingPatientID(ctx context.Context, actorUserID uuid.UUID, actorRole platform.Role, suppliedPatientID *uuid.UUID) (uuid.UUID, error) {
	if actorRole == platform.RolePatient {
		patient, err := s.records.GetOwnPatientRecord(ctx, actorUserID)
		if err != nil {
			return uuid.UUID{}, fmt.Errorf("resolve own patient record: %w", err)
		}
		return patient.ID, nil
	}

	if suppliedPatientID == nil {
		return uuid.UUID{}, ErrPatientIDRequired
	}
	if _, err := s.records.GetPatient(ctx, *suppliedPatientID); err != nil {
		return uuid.UUID{}, fmt.Errorf("resolve patient: %w", err)
	}
	return *suppliedPatientID, nil
}

// checkSlotFree re-verifies [start, end) doesn't overlap an existing
// non-cancelled appointment for providerID, excluding excludeID itself
// (used by Reschedule so an appointment being moved doesn't conflict with
// its own current slot).
func (s *Service) checkSlotFree(ctx context.Context, providerID uuid.UUID, start, end time.Time, excludeID *uuid.UUID) error {
	existing, err := s.repo.ListAppointmentsForProviderInRange(ctx, providerID, start, end)
	if err != nil {
		return fmt.Errorf("check existing appointments: %w", err)
	}
	for _, a := range existing {
		if excludeID != nil && a.ID == *excludeID {
			continue
		}
		return ErrSlotUnavailable
	}
	return nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (Appointment, error) {
	return s.repo.FindAppointmentByID(ctx, id)
}

// isPartyOrAdmin is Reschedule/Cancel's authorization rule: the booking
// patient, the assigned provider, or hospital_admin/system_admin.
// hospital_admin is scoped to appointments at their own facility — see the
// doc comment on resolveManageableProviderID for the same judgment call
// applied here.
func (s *Service) isPartyOrAdmin(ctx context.Context, a Appointment, actorUserID uuid.UUID, actorRole platform.Role, actorFacilityID *uuid.UUID) bool {
	switch actorRole {
	case platform.RoleSystemAdmin:
		return true
	case platform.RoleHospitalAdmin:
		return actorFacilityID != nil && *actorFacilityID == a.FacilityID
	case platform.RolePhysician:
		providerID, err := s.provider.ResolveProviderID(ctx, actorUserID)
		if err != nil {
			return false
		}
		return providerID == a.ProviderID
	case platform.RolePatient:
		patient, err := s.records.GetOwnPatientRecord(ctx, actorUserID)
		if err != nil {
			return false
		}
		return patient.ID == a.PatientID
	default:
		return false
	}
}

// isProviderOrAdmin is Complete/NoShow's stricter authorization rule — a
// clinical-outcome call the booking patient never makes about their own
// appointment, unlike reschedule/cancel.
func (s *Service) isProviderOrAdmin(ctx context.Context, a Appointment, actorUserID uuid.UUID, actorRole platform.Role, actorFacilityID *uuid.UUID) bool {
	switch actorRole {
	case platform.RoleSystemAdmin:
		return true
	case platform.RoleHospitalAdmin:
		return actorFacilityID != nil && *actorFacilityID == a.FacilityID
	case platform.RolePhysician:
		providerID, err := s.provider.ResolveProviderID(ctx, actorUserID)
		if err != nil {
			return false
		}
		return providerID == a.ProviderID
	default:
		return false
	}
}

// Reschedule re-validates the new window is free (excluding the
// appointment's own current slot) before moving it — see checkSlotFree.
func (s *Service) Reschedule(ctx context.Context, actorUserID uuid.UUID, actorRole platform.Role, actorFacilityID *uuid.UUID, appointmentID uuid.UUID, newStart, newEnd time.Time) (Appointment, error) {
	if !newEnd.After(newStart) {
		return Appointment{}, ErrInvalidTimeRange
	}

	appointment, err := s.repo.FindAppointmentByID(ctx, appointmentID)
	if err != nil {
		return Appointment{}, err
	}
	if !s.isPartyOrAdmin(ctx, appointment, actorUserID, actorRole, actorFacilityID) {
		return Appointment{}, ErrNotAuthorized
	}

	if err := s.checkSlotFree(ctx, appointment.ProviderID, newStart, newEnd, &appointment.ID); err != nil {
		return Appointment{}, err
	}

	return s.repo.UpdateAppointmentTime(ctx, appointment.ID, newStart, newEnd)
}

func (s *Service) Cancel(ctx context.Context, actorUserID uuid.UUID, actorRole platform.Role, actorFacilityID *uuid.UUID, appointmentID uuid.UUID) (Appointment, error) {
	appointment, err := s.repo.FindAppointmentByID(ctx, appointmentID)
	if err != nil {
		return Appointment{}, err
	}
	if !s.isPartyOrAdmin(ctx, appointment, actorUserID, actorRole, actorFacilityID) {
		return Appointment{}, ErrNotAuthorized
	}
	return s.repo.UpdateAppointmentStatus(ctx, appointment.ID, StatusCancelled)
}

func (s *Service) Complete(ctx context.Context, actorUserID uuid.UUID, actorRole platform.Role, actorFacilityID *uuid.UUID, appointmentID uuid.UUID) (Appointment, error) {
	appointment, err := s.repo.FindAppointmentByID(ctx, appointmentID)
	if err != nil {
		return Appointment{}, err
	}
	if !s.isProviderOrAdmin(ctx, appointment, actorUserID, actorRole, actorFacilityID) {
		return Appointment{}, ErrNotAuthorized
	}
	return s.repo.UpdateAppointmentStatus(ctx, appointment.ID, StatusCompleted)
}

func (s *Service) NoShow(ctx context.Context, actorUserID uuid.UUID, actorRole platform.Role, actorFacilityID *uuid.UUID, appointmentID uuid.UUID) (Appointment, error) {
	appointment, err := s.repo.FindAppointmentByID(ctx, appointmentID)
	if err != nil {
		return Appointment{}, err
	}
	if !s.isProviderOrAdmin(ctx, appointment, actorUserID, actorRole, actorFacilityID) {
		return Appointment{}, ErrNotAuthorized
	}
	return s.repo.UpdateAppointmentStatus(ctx, appointment.ID, StatusNoShow)
}

func (s *Service) ListForPatient(ctx context.Context, patientID uuid.UUID) ([]Appointment, error) {
	return s.repo.ListAppointmentsForPatient(ctx, patientID)
}

func (s *Service) ListForProvider(ctx context.Context, providerID uuid.UUID, from, to time.Time) ([]Appointment, error) {
	return s.repo.ListAppointmentsForProviderInRange(ctx, providerID, from, to)
}

// SendDueReminders is the reminder ticker's entry point (see the goroutine
// started in cmd/api/main.go). If no publisher has been wired at all, it's
// a deliberate no-op rather than marking appointments "sent" with nobody
// ever notified — see SetNotificationPublisher. For each due appointment
// with a linked user account, a failed publish is logged and left
// unmarked so the next tick retries it; an appointment whose patient has no
// linked user account is marked sent anyway (there's nobody to notify, so
// retrying forever would be pointless). Returns the count actually
// notified, for the caller to log.
func (s *Service) SendDueReminders(ctx context.Context) (int, error) {
	if s.notify == nil {
		return 0, nil
	}

	due, err := s.repo.ListUpcomingUnreminded(ctx)
	if err != nil {
		return 0, fmt.Errorf("list upcoming unreminded appointments: %w", err)
	}

	sent := 0
	for _, a := range due {
		patient, err := s.records.GetPatient(ctx, a.PatientID)
		if err != nil {
			slog.Error("failed to resolve patient for appointment reminder", "error", err, "appointment_id", a.ID)
			continue
		}

		if patient.UserID != nil {
			err := s.notify.Publish(ctx, *patient.UserID, "appointment_reminder", map[string]any{
				"appointment_id": a.ID.String(), "start_at": a.StartAt.Format(time.RFC3339),
			})
			if err != nil {
				slog.Error("failed to publish appointment reminder notification", "error", err, "appointment_id", a.ID)
				continue
			}
			sent++
		}

		if err := s.repo.MarkReminderSent(ctx, a.ID); err != nil {
			slog.Error("failed to mark appointment reminder sent", "error", err, "appointment_id", a.ID)
		}
	}
	return sent, nil
}
