package records

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/consent"
)

// ProviderResolver maps the calling user to their provider identity — the
// providers.id an encounter is recorded under, distinct from users.id.
// Implemented by facilities.Service, wired in main.go.
type ProviderResolver interface {
	ResolveProviderID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

type Service struct {
	repo     *Repository
	consent  *consent.Checker
	provider ProviderResolver
}

func NewService(repo *Repository, consentChecker *consent.Checker, provider ProviderResolver) *Service {
	return &Service{repo: repo, consent: consentChecker, provider: provider}
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

// ListPatients returns every registered patient across all facilities —
// unlike GetPatient this bypasses consent grants, so callers must gate it
// to a role that's meant to see system-wide patient data (currently
// system_admin only, checked in the handler).
func (s *Service) ListPatients(ctx context.Context) ([]Patient, error) {
	return s.repo.ListPatients(ctx)
}

func (s *Service) CountPatients(ctx context.Context) (int64, error) {
	return s.repo.CountPatients(ctx)
}

func (s *Service) CountEncounters(ctx context.Context) (int64, error) {
	return s.repo.CountEncounters(ctx)
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

	return s.repo.CreateEncounter(ctx, CreateEncounterParams{
		PatientID: in.PatientID, FacilityID: actorFacilityID, ProviderID: providerID,
		Type: in.Type, Notes: in.Notes, OccurredAt: in.OccurredAt,
	})
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
	return s.repo.UpsertPatientMedicalHistory(
		ctx, patientID, updatedBy,
		defaultJSONArray(in.Allergies),
		defaultJSONArray(in.ChronicConditions),
		defaultJSONArray(in.CurrentMedications),
		defaultJSONArray(in.PastSurgeries),
		defaultJSONArray(in.FamilyHistory),
		defaultJSONArray(in.VaccinationHistory),
	)
}
