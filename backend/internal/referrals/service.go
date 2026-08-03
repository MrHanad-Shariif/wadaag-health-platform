package referrals

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/consent"
	"github.com/wadaag/health-platform/backend/internal/facilities"
)

// ProviderResolver maps the calling user to their provider identity.
// Implemented by facilities.Service, wired in main.go.
type ProviderResolver interface {
	ResolveProviderID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

// ProviderLookup resolves a provider profile (including its user_id) by
// provider id — the reverse direction of ProviderResolver. Needed to notify
// the referring provider's user account on a status change. Optional (nil by
// default, see Service.providers); set via SetProviderLookup. Satisfied by
// *facilities.Service.
type ProviderLookup interface {
	GetProvider(ctx context.Context, id uuid.UUID) (facilities.Provider, error)
}

// NotificationPublisher is the narrow surface this package needs from
// notifications.Service, defined locally so referrals doesn't import that
// package directly. Optional (nil by default); set via
// SetNotificationPublisher. Satisfied by *notifications.Service.
type NotificationPublisher interface {
	Publish(ctx context.Context, userID uuid.UUID, notifType string, payload map[string]any) error
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

	// providers and notify are optional (nil by default) — set via
	// SetProviderLookup / SetNotificationPublisher in main.go. Nil-tolerant
	// everywhere they're used, so existing construction paths and any future
	// tests that build a Service without wiring them keep working unchanged.
	providers ProviderLookup
	notify    NotificationPublisher
}

func NewService(repo *Repository, consentGranter consentGrants, provider ProviderResolver) *Service {
	return &Service{repo: repo, consent: consentGranter, provider: provider}
}

// SetProviderLookup wires the provider-lookup dependency used to resolve the
// referring provider's user_id for notifications. Nil-safe: if never called,
// notifyReferringProvider silently skips.
func (s *Service) SetProviderLookup(p ProviderLookup) { s.providers = p }

// SetNotificationPublisher wires the optional in-app notification publisher.
// Nil-safe: if never called, notifyReferringProvider silently skips.
func (s *Service) SetNotificationPublisher(p NotificationPublisher) { s.notify = p }

// notifyReferringProvider publishes a "referral_status_changed" notification
// to the referring provider's user account, if both the provider-lookup and
// notification-publisher dependencies are wired. Never returns an error to
// the caller — a failed or skipped notification must never block the status
// transition it's describing (same "log loudly, don't fail the primary
// operation" philosophy as audit.Logger.Record).
func (s *Service) notifyReferringProvider(ctx context.Context, referral Referral) {
	if s.notify == nil || s.providers == nil {
		return
	}
	provider, err := s.providers.GetProvider(ctx, referral.ReferringProviderID)
	if err != nil {
		slog.Error("failed to resolve referring provider for notification", "error", err, "referral_id", referral.ID)
		return
	}
	err = s.notify.Publish(ctx, provider.UserID, "referral_status_changed", map[string]any{
		"referral_id": referral.ID.String(), "status": string(referral.Status),
	})
	if err != nil {
		slog.Error("failed to publish referral status notification", "error", err, "referral_id", referral.ID)
	}
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

func (s *Service) CountByStatus(ctx context.Context) (map[string]int64, error) {
	return s.repo.CountByStatus(ctx)
}

func (s *Service) ListForFacility(ctx context.Context, facilityID uuid.UUID) ([]Referral, error) {
	return s.repo.ListForFacility(ctx, facilityID)
}

// ListAll is the system_admin oversight path — a platform administrator has
// no facility affiliation of their own to scope ListForFacility by, so this
// returns every referral instead. Restricting who may call this is the
// handler's job (see Handler.inbox).
func (s *Service) ListAll(ctx context.Context) ([]Referral, error) {
	return s.repo.ListAll(ctx)
}

// SearchForFacility is ListForFacility's text-filtered counterpart, reusing
// the exact same receiving-facility scoping — see search.Service.Search.
func (s *Service) SearchForFacility(ctx context.Context, facilityID uuid.UUID, query string, limit int) ([]Referral, error) {
	return s.repo.SearchForFacility(ctx, facilityID, query, limit)
}

// SearchAll is ListAll's text-filtered counterpart — system_admin oversight
// path only, restricting who may call it is the caller's job (see
// search.Service.Search).
func (s *Service) SearchAll(ctx context.Context, query string, limit int) ([]Referral, error) {
	return s.repo.SearchAll(ctx, query, limit)
}

func (s *Service) ListForPatient(ctx context.Context, patientID uuid.UUID) ([]Referral, error) {
	return s.repo.ListForPatient(ctx, patientID)
}

func (s *Service) ListStatusEvents(ctx context.Context, referralID uuid.UUID) ([]StatusEvent, error) {
	return s.repo.ListStatusEvents(ctx, referralID)
}

// MostActiveDoctors implements a narrow slice of dashboard.DoctorActivityCounter
// (via main.go's adapter — see ProviderReferralActivity) — the top `limit`
// providers by how many referrals they've received, most-referred first.
func (s *Service) MostActiveDoctors(ctx context.Context, limit int) ([]ProviderReferralActivity, error) {
	return s.repo.CountReferralsByReceivingProvider(ctx, limit)
}

// CountPendingReferralsForProvider implements dashboard.ReferralPendingCounter
// — how many referrals are currently assigned to providerID as the
// receiving provider and still need action (accepted or in_progress).
func (s *Service) CountPendingReferralsForProvider(ctx context.Context, providerID uuid.UUID) (int64, error) {
	return s.repo.CountPendingForProvider(ctx, providerID)
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
	s.notifyReferringProvider(ctx, updated)
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
	s.notifyReferringProvider(ctx, updated)
	return updated, nil
}
