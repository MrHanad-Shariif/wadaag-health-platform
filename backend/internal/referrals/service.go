package referrals

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/consent"
)

// ProviderResolver maps the calling user to their provider identity.
// Implemented by facilities.Service, wired in main.go.
type ProviderResolver interface {
	ResolveProviderID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

// consentGrants is the narrow slice of consent.Checker this service needs —
// granting the receiving facility implied access, not the full HasAccess
// surface (that's used by the consent middleware in the handler instead).
type consentGrants interface {
	Grant(ctx context.Context, in consent.GrantInput) (consent.Grant, error)
}

// consentImpliedAccessDays is how long a referral's implied consent grant
// lasts before it needs to be renewed by an explicit patient grant or a new
// referral — long enough to cover a typical course of care, short enough
// that stale access doesn't linger indefinitely.
const consentImpliedAccessDays = 30

type Service struct {
	repo     *Repository
	consent  consentGrants
	provider ProviderResolver
}

func NewService(repo *Repository, consentGranter consentGrants, provider ProviderResolver) *Service {
	return &Service{repo: repo, consent: consentGranter, provider: provider}
}

type CreateReferralInput struct {
	PatientID                  uuid.UUID
	ReceivingFacilityID        uuid.UUID
	SpecialtyRequested         string
	Urgency                    Urgency
	Reason                     string
	ClinicalSummaryEncounterID *uuid.UUID
}

// Create records the referral, then grants the receiving facility
// time-boxed access to the patient's record — this is the "referring
// providers get automatic, time-boxed implied consent" rule from the
// platform's consent design, generalized to whichever facility receives
// the referral rather than one named individual.
func (s *Service) Create(ctx context.Context, actorUserID, referringFacilityID uuid.UUID, in CreateReferralInput) (Referral, error) {
	providerID, err := s.provider.ResolveProviderID(ctx, actorUserID)
	if err != nil {
		return Referral{}, fmt.Errorf("resolve provider identity: %w", err)
	}

	referral, err := s.repo.Create(ctx, CreateReferralParams{
		PatientID: in.PatientID, ReferringProviderID: providerID, ReferringFacilityID: referringFacilityID,
		ReceivingFacilityID: in.ReceivingFacilityID, SpecialtyRequested: in.SpecialtyRequested,
		Urgency: in.Urgency, Reason: in.Reason, ClinicalSummaryEncounterID: in.ClinicalSummaryEncounterID,
	})
	if err != nil {
		return Referral{}, err
	}

	if err := s.repo.CreateStatusEvent(ctx, referral.ID, nil, StatusRouted, actorUserID, nil); err != nil {
		return Referral{}, err
	}

	expiresAt := time.Now().AddDate(0, 0, consentImpliedAccessDays)
	_, err = s.consent.Grant(ctx, consent.GrantInput{
		PatientID: in.PatientID, Grantee: consent.GranteeTypeFacility, GranteeID: in.ReceivingFacilityID,
		Scope: consent.ScopeReferralOnly, ScopeRef: &referral.ID,
		GrantedVia: consent.GrantedViaReferralImpliedTemporary, ExpiresAt: &expiresAt,
	})
	if err != nil {
		return Referral{}, fmt.Errorf("grant receiving-facility consent: %w", err)
	}

	return referral, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (Referral, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) ListForFacility(ctx context.Context, facilityID uuid.UUID) ([]Referral, error) {
	return s.repo.ListForFacility(ctx, facilityID)
}

func (s *Service) ListForPatient(ctx context.Context, patientID uuid.UUID) ([]Referral, error) {
	return s.repo.ListForPatient(ctx, patientID)
}

func (s *Service) ListStatusEvents(ctx context.Context, referralID uuid.UUID) ([]StatusEvent, error) {
	return s.repo.ListStatusEvents(ctx, referralID)
}

var ErrNotReceivingFacility = fmt.Errorf("only the receiving facility can perform this action")
var ErrNotReferralParty = fmt.Errorf("only the referring or receiving facility can perform this action")
var ErrInvalidTransition = fmt.Errorf("referral is not in a state that allows this transition")

func (s *Service) Accept(ctx context.Context, actorUserID, actorFacilityID, referralID uuid.UUID, version int32) (Referral, error) {
	current, err := s.repo.FindByID(ctx, referralID)
	if err != nil {
		return Referral{}, err
	}
	if current.ReceivingFacilityID == nil || *current.ReceivingFacilityID != actorFacilityID {
		return Referral{}, ErrNotReceivingFacility
	}

	providerID, err := s.provider.ResolveProviderID(ctx, actorUserID)
	if err != nil {
		return Referral{}, fmt.Errorf("resolve provider identity: %w", err)
	}

	updated, err := s.repo.Accept(ctx, referralID, version, providerID)
	if err != nil {
		return Referral{}, err
	}

	from := StatusRouted
	_ = s.repo.CreateStatusEvent(ctx, referralID, &from, StatusAccepted, actorUserID, nil)
	return updated, nil
}

// Decline, StartProgress, Complete, and Cancel share the same shape: verify
// the actor's facility is allowed to make this transition from the
// referral's current status, then apply it. Kept as one method rather than
// four near-identical ones.
type Transition struct {
	To              Status
	AllowedFrom     Status
	RequireFacility func(r Referral) *uuid.UUID // which facility_id must match the actor's
}

func (s *Service) Transition(ctx context.Context, actorUserID, actorFacilityID, referralID uuid.UUID, version int32, t Transition, note *string) (Referral, error) {
	current, err := s.repo.FindByID(ctx, referralID)
	if err != nil {
		return Referral{}, err
	}
	if current.Status != t.AllowedFrom {
		return Referral{}, ErrInvalidTransition
	}

	required := t.RequireFacility(current)
	if required == nil || *required != actorFacilityID {
		return Referral{}, ErrNotReceivingFacility
	}

	updated, err := s.repo.UpdateStatus(ctx, referralID, version, t.To)
	if err != nil {
		return Referral{}, err
	}

	from := t.AllowedFrom
	_ = s.repo.CreateStatusEvent(ctx, referralID, &from, t.To, actorUserID, note)
	return updated, nil
}

func ReceivingFacility(r Referral) *uuid.UUID { return r.ReceivingFacilityID }

// Cancel is its own method rather than a Transition because either party —
// referring or receiving facility — may cancel, and it's valid from three
// different starting states.
func (s *Service) Cancel(ctx context.Context, actorUserID, actorFacilityID, referralID uuid.UUID, version int32, note *string) (Referral, error) {
	current, err := s.repo.FindByID(ctx, referralID)
	if err != nil {
		return Referral{}, err
	}

	switch current.Status {
	case StatusRouted, StatusAccepted, StatusInProgress:
	default:
		return Referral{}, ErrInvalidTransition
	}

	isReferring := current.ReferringFacilityID == actorFacilityID
	isReceiving := current.ReceivingFacilityID != nil && *current.ReceivingFacilityID == actorFacilityID
	if !isReferring && !isReceiving {
		return Referral{}, ErrNotReferralParty
	}

	updated, err := s.repo.UpdateStatus(ctx, referralID, version, StatusCancelled)
	if err != nil {
		return Referral{}, err
	}

	from := current.Status
	_ = s.repo.CreateStatusEvent(ctx, referralID, &from, StatusCancelled, actorUserID, note)
	return updated, nil
}
