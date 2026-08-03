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
	ActionUploadAttachment     Action = "upload_attachment"
	ActionViewAttachment       Action = "view_attachment"
	ActionCreateConsult        Action = "create_consult"
	ActionViewConsult          Action = "view_consult"
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

// EntryFilter is the general audit-browser listing's optional filter set
// (see Service.ListEntries / Handler's "GET /" route) — a nil field matches
// every row for that column. Limit is always resolved to a positive,
// capped value before it reaches the repository; see NormalizeLimit.
type EntryFilter struct {
	ActorUserID  *uuid.UUID
	ResourceType *string
	Result       *Result
	Limit        int
}

// maxEntryFilterLimit caps how many rows a single ListEntries call can
// return — the audit log has no natural upper bound on size, so an
// unbounded or client-supplied huge limit could otherwise turn this into
// an accidental full-table scan/dump.
const maxEntryFilterLimit = 200

// defaultEntryFilterLimit is used when the caller doesn't specify a limit
// (or specifies a non-positive one).
const defaultEntryFilterLimit = 50

// NormalizeLimit resolves f.Limit to the value that should actually be
// sent to the repository: the caller's limit if it's positive and within
// maxEntryFilterLimit, defaultEntryFilterLimit if unset/non-positive, or
// maxEntryFilterLimit if the caller asked for more than that. Pulled out
// as a pure function (no DB) so this normalization can be unit-tested
// directly.
func (f EntryFilter) NormalizeLimit() int {
	if f.Limit <= 0 {
		return defaultEntryFilterLimit
	}
	if f.Limit > maxEntryFilterLimit {
		return maxEntryFilterLimit
	}
	return f.Limit
}
