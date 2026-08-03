package audit

import (
	"time"

	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/platform"
)

type Result string

const (
	ResultAllowed Result = "allowed"
	ResultDenied  Result = "denied"
)

// Action names are free-form strings (not a closed enum) so new modules can
// add their own without a schema migration — the audit_log table's job is
// to record everything, not to gatekeep what gets recorded.
type Action string

const (
	ActionViewPatient          Action = "view_patient"
	ActionCreatePatient        Action = "create_patient"
	ActionUpdatePatient        Action = "update_patient"
	ActionViewEncounter        Action = "view_encounter"
	ActionCreateEncounter      Action = "create_encounter"
	ActionCreateReferral       Action = "create_referral"
	ActionUpdateReferral       Action = "update_referral"
	ActionGrantConsent         Action = "grant_consent"
	ActionRevokeConsent        Action = "revoke_consent"
	ActionViewMedicalHistory   Action = "view_medical_history"
	ActionUpdateMedicalHistory Action = "update_medical_history"
)

type Entry struct {
	ID           uuid.UUID
	ActorUserID  uuid.UUID
	ActorRole    platform.Role
	Action       Action
	ResourceType string
	ResourceID   *uuid.UUID
	PatientID    *uuid.UUID
	Result       Result
	OccurredAt   time.Time
}
