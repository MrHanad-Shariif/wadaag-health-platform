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
	Gender      *string
	BloodGroup  *string
	Version     int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// PatientMedicalHistory is one row per patient (created lazily on first
// write — see Service.GetMedicalHistory/UpdateMedicalHistory). Each of the
// six fields is a raw JSON array passed through as-is, same pattern as
// ClinicalObservation.Payload — no application-level schema is enforced on
// their contents.
type PatientMedicalHistory struct {
	PatientID          uuid.UUID
	Allergies          []byte
	ChronicConditions  []byte
	CurrentMedications []byte
	PastSurgeries      []byte
	FamilyHistory      []byte
	VaccinationHistory []byte
	UpdatedBy          *uuid.UUID
	CreatedAt          time.Time
	UpdatedAt          time.Time
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
