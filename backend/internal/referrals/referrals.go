package referrals

import (
	"time"

	"github.com/google/uuid"
)

type Urgency string

const (
	UrgencyRoutine   Urgency = "routine"
	UrgencyUrgent    Urgency = "urgent"
	UrgencyEmergency Urgency = "emergency"
)

type Status string

const (
	StatusCreated    Status = "created"
	StatusRouted     Status = "routed"
	StatusAccepted   Status = "accepted"
	StatusDeclined   Status = "declined"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusCancelled  Status = "cancelled"
)

type Referral struct {
	ID                         uuid.UUID
	PatientID                  uuid.UUID
	ReferringProviderID        uuid.UUID
	ReferringFacilityID        uuid.UUID
	ReceivingFacilityID        *uuid.UUID
	ReceivingProviderID        *uuid.UUID
	SpecialtyRequested         string
	Urgency                    Urgency
	Status                     Status
	Reason                     string
	ClinicalSummaryEncounterID *uuid.UUID
	Version                    int32
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

type StatusEvent struct {
	ID          uuid.UUID
	ReferralID  uuid.UUID
	FromStatus  *Status
	ToStatus    Status
	ActorUserID uuid.UUID
	Note        *string
	OccurredAt  time.Time
}

// ProviderReferralActivity is one row of Service.MostActiveDoctors — how
// many referrals a given provider has received as receiving_provider_id,
// plus their display name (joined from users.full_name), for the admin
// dashboard's "most active doctors" panel. Kept as this package's own type
// (rather than dashboard.DoctorActivity directly) so referrals doesn't
// import the dashboard package — main.go adapts between the two when
// wiring dashboard.Service. See dashboard.DoctorActivityCounter.
type ProviderReferralActivity struct {
	ProviderID    uuid.UUID
	FullName      *string
	ReferralCount int64
}
