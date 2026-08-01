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
