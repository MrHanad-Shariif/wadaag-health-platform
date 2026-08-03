package records

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/consent"
	"github.com/wadaag/health-platform/backend/internal/platform"
)

// ProviderResolver maps the calling user to their provider identity — the
// providers.id an encounter is recorded under, distinct from users.id.
// Implemented by facilities.Service, wired in main.go.
type ProviderResolver interface {
	ResolveProviderID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

// NotificationPublisher is the narrow surface this package needs from
// notifications.Service, defined locally so records doesn't import that
// package directly. Optional (nil by default); set via
// SetNotificationPublisher. Satisfied by *notifications.Service.
type NotificationPublisher interface {
	Publish(ctx context.Context, userID uuid.UUID, notifType string, payload map[string]any) error
}

type Service struct {
	repo     *Repository
	consent  *consent.Checker
	provider ProviderResolver

	// notify is optional (nil by default) — set via SetNotificationPublisher
	// in main.go. Nil-tolerant wherever it's used, so existing construction
	// paths and tests keep working unchanged.
	notify NotificationPublisher
}

func NewService(repo *Repository, consentChecker *consent.Checker, provider ProviderResolver) *Service {
	return &Service{repo: repo, consent: consentChecker, provider: provider}
}

// SetNotificationPublisher wires the optional in-app notification publisher.
func (s *Service) SetNotificationPublisher(p NotificationPublisher) { s.notify = p }

// notifyPatientUser publishes a "record_updated" notification to the
// patient's linked user account, if one exists and a publisher is wired.
// Never returns an error — a failed or skipped notification must never
// block the write it's describing (same philosophy as
// audit.Logger.Record).
func (s *Service) notifyPatientUser(ctx context.Context, patientID uuid.UUID, kind string) {
	if s.notify == nil {
		return
	}
	patientUserID, err := s.GetPatientUserID(ctx, patientID)
	if err != nil {
		slog.Error("failed to resolve patient user for notification", "error", err, "patient_id", patientID)
		return
	}
	if patientUserID == nil {
		return
	}
	err = s.notify.Publish(ctx, *patientUserID, "record_updated", map[string]any{
		"patient_id": patientID.String(), "kind": kind,
	})
	if err != nil {
		slog.Error("failed to publish record updated notification", "error", err, "patient_id", patientID)
	}
}

type CreatePatientInput struct {
	UserID      *uuid.UUID
	FullName    string
	DateOfBirth *time.Time
	Sex         *string
	NationalID  *string
	Phone       *string
	Address     *string
	NextOfKin   *string
}

// CreatePatient registers a new patient and grants the creating provider's
// facility standing (non-expiring) access to the record — the facility
// that onboarded a patient is treated as their home facility. Any other
// facility needs an explicit or referral-driven grant to see this patient.
func (s *Service) CreatePatient(ctx context.Context, actorUserID uuid.UUID, actorFacilityID uuid.UUID, in CreatePatientInput) (Patient, error) {
	patient, err := s.repo.CreatePatient(ctx, CreatePatientParams{
		UserID: in.UserID, FullName: in.FullName, DateOfBirth: in.DateOfBirth,
		Sex: in.Sex, NationalID: in.NationalID, Phone: in.Phone, Address: in.Address, NextOfKin: in.NextOfKin,
	})
	if err != nil {
		return Patient{}, err
	}

	_, err = s.consent.Grant(ctx, consent.GrantInput{
		PatientID: patient.ID, Grantee: consent.GranteeTypeFacility, GranteeID: actorFacilityID,
		Scope: consent.ScopeFullRecord, GrantedVia: consent.GrantedViaProviderCreated,
	})
	if err != nil {
		return Patient{}, fmt.Errorf("grant creating-facility consent: %w", err)
	}

	return patient, nil
}

func (s *Service) GetPatient(ctx context.Context, id uuid.UUID) (Patient, error) {
	return s.repo.FindPatientByID(ctx, id)
}

// GetOwnPatientRecord resolves a patient-role user's own patient_id from
// their user_id — the reverse of GetPatientUserID. Needed anywhere a
// patient-role caller must act on their own record without being able to
// supply a patient_id themselves (see appointments.Service.Book, which
// never trusts a patient-supplied patient_id).
func (s *Service) GetOwnPatientRecord(ctx context.Context, userID uuid.UUID) (Patient, error) {
	return s.repo.FindPatientByUserID(ctx, userID)
}

// ListPatients returns every registered patient across all facilities —
// unlike GetPatient this bypasses consent grants, so callers must gate it
// to a role that's meant to see system-wide patient data (currently
// system_admin only, checked in the handler).
func (s *Service) ListPatients(ctx context.Context) ([]Patient, error) {
	return s.repo.ListPatients(ctx)
}

// SearchPatients performs a directory-style patient search, scoped by the
// caller's role — see the doc comment on ListPatients for why patient
// visibility can't be a single unscoped query. system_admin gets the
// unscoped, system-wide view; every other caller gets results scoped to
// patients their facility already holds an active consent grant for (the
// same boundary consent.Checker.HasAccess enforces per-patient), never a
// system-wide view — this is a search, not a way to bypass consent.
// Restricting *who* may call SearchPatients at all (system_admin,
// physician, hospital_admin) is the caller's job (see search.Service),
// not this method's; a caller with neither an admin role nor a facility
// affiliation gets an empty result rather than an error.
func (s *Service) SearchPatients(ctx context.Context, actorRole platform.Role, actorFacilityID *uuid.UUID, query string, limit int) ([]Patient, error) {
	if actorRole == platform.RoleSystemAdmin {
		return s.repo.SearchPatientsUnscoped(ctx, query, limit)
	}
	if actorFacilityID == nil {
		return nil, nil
	}
	return s.repo.SearchPatientsForFacility(ctx, *actorFacilityID, query, limit)
}

func (s *Service) CountPatients(ctx context.Context) (int64, error) {
	return s.repo.CountPatients(ctx)
}

func (s *Service) CountEncounters(ctx context.Context) (int64, error) {
	return s.repo.CountEncounters(ctx)
}

// CountEncountersForProviderToday implements dashboard.EncounterTodayCounter
// — how many encounters providerID has recorded today.
func (s *Service) CountEncountersForProviderToday(ctx context.Context, providerID uuid.UUID) (int64, error) {
	return s.repo.CountEncountersForProviderToday(ctx, providerID)
}

// GetPatientUserID implements referrals.PatientLookup — lets referrals
// resolve the patient-self identity for its own consent check without
// importing this package's full Patient type.
func (s *Service) GetPatientUserID(ctx context.Context, patientID uuid.UUID) (*uuid.UUID, error) {
	patient, err := s.repo.FindPatientByID(ctx, patientID)
	if err != nil {
		return nil, err
	}
	return patient.UserID, nil
}

// IsPatientUser implements audit.PatientIdentityResolver and
// consent.PatientIdentityResolver.
func (s *Service) IsPatientUser(ctx context.Context, patientID, userID uuid.UUID) (bool, error) {
	patient, err := s.repo.FindPatientByID(ctx, patientID)
	if err != nil {
		return false, err
	}
	return patient.UserID != nil && *patient.UserID == userID, nil
}

type CreateEncounterInput struct {
	PatientID  uuid.UUID
	Type       EncounterType
	Notes      *string
	OccurredAt time.Time
}

func (s *Service) CreateEncounter(ctx context.Context, actorUserID uuid.UUID, actorFacilityID uuid.UUID, in CreateEncounterInput) (Encounter, error) {
	providerID, err := s.provider.ResolveProviderID(ctx, actorUserID)
	if err != nil {
		return Encounter{}, fmt.Errorf("resolve provider identity: %w", err)
	}

	encounter, err := s.repo.CreateEncounter(ctx, CreateEncounterParams{
		PatientID: in.PatientID, FacilityID: actorFacilityID, ProviderID: providerID,
		Type: in.Type, Notes: in.Notes, OccurredAt: in.OccurredAt,
	})
	if err != nil {
		return Encounter{}, err
	}

	s.notifyPatientUser(ctx, in.PatientID, "encounter")

	return encounter, nil
}

func (s *Service) GetEncounter(ctx context.Context, id uuid.UUID) (Encounter, error) {
	return s.repo.FindEncounterByID(ctx, id)
}

func (s *Service) ListEncounters(ctx context.Context, patientID uuid.UUID) ([]Encounter, error) {
	return s.repo.ListEncountersByPatient(ctx, patientID)
}

func (s *Service) CreateObservation(ctx context.Context, actorUserID uuid.UUID, encounterID uuid.UUID, obsType ObservationType, payload json.RawMessage) (ClinicalObservation, error) {
	return s.repo.CreateObservation(ctx, encounterID, obsType, payload, actorUserID)
}

func (s *Service) ListObservations(ctx context.Context, encounterID uuid.UUID) ([]ClinicalObservation, error) {
	return s.repo.ListObservationsByEncounter(ctx, encounterID)
}

// UpdatePatientDemographics is a partial update over the two fields it owns
// (gender, blood_group): nil means "leave this field unchanged," so it
// reads the current row first and overrides only the non-nil inputs before
// writing — same idea as facilities' logo/hours PATCH — rather than blindly
// writing whatever was passed (which would clear an omitted field to NULL).
func (s *Service) UpdatePatientDemographics(ctx context.Context, patientID uuid.UUID, gender, bloodGroup *string) (Patient, error) {
	current, err := s.repo.FindPatientByID(ctx, patientID)
	if err != nil {
		return Patient{}, err
	}

	if gender == nil {
		gender = current.Gender
	}
	if bloodGroup == nil {
		bloodGroup = current.BloodGroup
	}

	return s.repo.UpdatePatientGenderBloodGroup(ctx, patientID, gender, bloodGroup)
}

// emptyJSONArray is the default for any medical-history field omitted from
// an UpdateMedicalHistoryInput or never written at all.
var emptyJSONArray = []byte("[]")

// GetMedicalHistory returns patientID's medical history, or a zero-value
// PatientMedicalHistory (every field an empty JSON array) if no row has
// ever been written for them — see the row lifecycle note on
// PatientMedicalHistory. Never returns ErrMedicalHistoryNotFound to the
// caller, matching identity.Service.GetProfile's "created lazily" handling.
func (s *Service) GetMedicalHistory(ctx context.Context, patientID uuid.UUID) (PatientMedicalHistory, error) {
	history, err := s.repo.FindPatientMedicalHistoryByPatientID(ctx, patientID)
	return resolveMedicalHistory(patientID, history, err)
}

// resolveMedicalHistory is the pure "never 404" decision GetMedicalHistory
// delegates to: given whatever the repository returned (including a
// not-found error), decide what to hand back to the caller. Pulled out as a
// DB-free function — same reasoning as identity's mergeProfileUpdate — so
// the "first GET returns zero-value, not an error" behavior can be
// unit-tested directly without a fake repository or a live database.
func resolveMedicalHistory(patientID uuid.UUID, history PatientMedicalHistory, err error) (PatientMedicalHistory, error) {
	if errors.Is(err, ErrMedicalHistoryNotFound) {
		return PatientMedicalHistory{
			PatientID:          patientID,
			Allergies:          emptyJSONArray,
			ChronicConditions:  emptyJSONArray,
			CurrentMedications: emptyJSONArray,
			PastSurgeries:      emptyJSONArray,
			FamilyHistory:      emptyJSONArray,
			VaccinationHistory: emptyJSONArray,
		}, nil
	}
	if err != nil {
		return PatientMedicalHistory{}, err
	}
	return history, nil
}

// UpdateMedicalHistoryInput is PATCH /patients/{id}/medical-history's body,
// pre-decoded. Unlike the profile PATCH, this is always a full replacement
// of all six fields, not a per-field merge — a field omitted from the
// request body (nil/empty RawMessage) is written as an empty JSON array
// `[]`, not left unchanged. This is a simpler contract than partial-update
// and matches the "resend the whole list" shape a medical-history edit form
// naturally produces.
type UpdateMedicalHistoryInput struct {
	Allergies          json.RawMessage
	ChronicConditions  json.RawMessage
	CurrentMedications json.RawMessage
	PastSurgeries      json.RawMessage
	FamilyHistory      json.RawMessage
	VaccinationHistory json.RawMessage
}

func defaultJSONArray(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return emptyJSONArray
	}
	return []byte(raw)
}

func (s *Service) UpdateMedicalHistory(ctx context.Context, patientID uuid.UUID, updatedBy uuid.UUID, in UpdateMedicalHistoryInput) (PatientMedicalHistory, error) {
	history, err := s.repo.UpsertPatientMedicalHistory(
		ctx, patientID, updatedBy,
		defaultJSONArray(in.Allergies),
		defaultJSONArray(in.ChronicConditions),
		defaultJSONArray(in.CurrentMedications),
		defaultJSONArray(in.PastSurgeries),
		defaultJSONArray(in.FamilyHistory),
		defaultJSONArray(in.VaccinationHistory),
	)
	if err != nil {
		return PatientMedicalHistory{}, err
	}

	s.notifyPatientUser(ctx, patientID, "medical_history")

	return history, nil
}
