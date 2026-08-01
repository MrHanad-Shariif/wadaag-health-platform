package records

import (
	"time"

	"github.com/google/uuid"
)

type Patient struct {
	ID          uuid.UUID
	UserID      *uuid.UUID
	FullName    string
	DateOfBirth *time.Time
	Sex         *string
	NationalID  *string
	Phone       *string
	Address     *string
	NextOfKin   *string
	Version     int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type EncounterType string

const (
	EncounterTypeReferral     EncounterType = "referral"
	EncounterTypeConsult      EncounterType = "consult"
	EncounterTypeGeneralVisit EncounterType = "general_visit"
)

type Encounter struct {
	ID         uuid.UUID
	PatientID  uuid.UUID
	FacilityID uuid.UUID
	ProviderID uuid.UUID
	Type       EncounterType
	Notes      *string
	OccurredAt time.Time
	Version    int32
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type ObservationType string

const (
	ObservationTypeVitals     ObservationType = "vitals"
	ObservationTypeDiagnosis  ObservationType = "diagnosis"
	ObservationTypeNote       ObservationType = "note"
	ObservationTypeAttachment ObservationType = "attachment"
)

type ClinicalObservation struct {
	ID          uuid.UUID
	EncounterID uuid.UUID
	Type        ObservationType
	Payload     []byte
	RecordedBy  uuid.UUID
	RecordedAt  time.Time
}
