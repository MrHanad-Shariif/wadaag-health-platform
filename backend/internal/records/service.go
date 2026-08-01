package records

import (
	"context"
	"encoding/json"
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
