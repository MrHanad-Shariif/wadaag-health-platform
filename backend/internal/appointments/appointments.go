// Package appointments implements Phase 8: patient/provider booking against
// a simple weekly availability schedule. Deliberately not a full
// calendaring engine — see generateSlots in service.go for the entire
// slot-generation algorithm, and the reminder ticker in cmd/api/main.go for
// the "lightweight cron, not a job queue" reminder delivery.
package appointments

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusScheduled Status = "scheduled"
	StatusConfirmed Status = "confirmed"
	StatusCancelled Status = "cancelled"
	StatusCompleted Status = "completed"
	StatusNoShow    Status = "no_show"
)

type Appointment struct {
	ID             uuid.UUID
	PatientID      uuid.UUID
	ProviderID     uuid.UUID
	FacilityID     uuid.UUID
	StartAt        time.Time
	EndAt          time.Time
	Status         Status
	Reason         *string
	ReminderSentAt *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// AvailabilityRule is one weekly recurring availability window for a
// provider, e.g. "every Monday 09:00-17:00". StartTime/EndTime are kept as
// plain "HH:MM" strings at the domain level, converting to/from Postgres
// TIME only in the repository (see platform.PgTimeOfDay/FromPgTimeOfDay) —
// simpler than inventing a dedicated time-of-day type for what's otherwise
// used as plain text everywhere else (request bodies, generateSlots' pure
// logic, JSON responses).
type AvailabilityRule struct {
	ID         uuid.UUID
	ProviderID uuid.UUID
	Weekday    int16 // 0=Sunday..6=Saturday, matching time.Weekday
	StartTime  string
	EndTime    string
	CreatedAt  time.Time
}

// Slot is one bookable window generateSlots produced — not itself
// persisted anywhere, just the shape GetAvailableSlots returns.
type Slot struct {
	StartAt time.Time
	EndAt   time.Time
}
