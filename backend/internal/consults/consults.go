// Package consults implements the doctor-to-doctor "second opinion" flow:
// one named provider asks another named provider about one patient.
//
// Unlike referrals (internal/referrals), which transfer a patient to a
// whole receiving FACILITY and therefore need a consent-grant-table entry
// plus a 7-state workflow, a consultation is a direct request between two
// named providers — nobody else needs access via this mechanism. There is
// no facility routing here and no consent.Middleware/consent.Checker.Grant
// involvement; authorization is simply "the requesting provider, the
// target provider, or system_admin", checked directly in the handler/
// service (see Service's party-check helper and ErrNotConsultParty).
package consults

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusRequested Status = "requested"
	StatusResponded Status = "responded"
	StatusClosed    Status = "closed"
)

type Consultation struct {
	ID                   uuid.UUID
	PatientID            uuid.UUID
	RequestingProviderID uuid.UUID
	TargetProviderID     uuid.UUID
	Reason               string
	Status               Status
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type Message struct {
	ID             uuid.UUID
	ConsultationID uuid.UUID
	SenderID       uuid.UUID
	Body           string
	AttachmentIDs  []uuid.UUID
	CreatedAt      time.Time
}
