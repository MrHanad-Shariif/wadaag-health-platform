package consults

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/attachments"
	"github.com/wadaag/health-platform/backend/internal/facilities"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"github.com/wadaag/health-platform/backend/internal/records"
)

// ProviderResolver maps the calling user to their provider identity. Same
// interface shape as referrals.ProviderResolver — satisfied by
// *facilities.Service, wired in main.go.
type ProviderResolver interface {
	ResolveProviderID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

// ProviderLookup resolves a provider profile by id, used to validate that
// target_provider_id refers to a real provider before a consultation is
// created against it. Satisfied by *facilities.Service.
type ProviderLookup interface {
	GetProvider(ctx context.Context, id uuid.UUID) (facilities.Provider, error)
}

var (
	// ErrNotConsultParty covers view/reply/close: only the requesting
	// provider, the target provider, or system_admin may act on a given
	// consultation.
	ErrNotConsultParty       = errors.New("only the requesting provider, target provider, or an administrator may access this consultation")
	ErrConsultClosed         = errors.New("consultation is closed and cannot be modified")
	ErrTargetProviderInvalid = errors.New("target_provider_id must refer to an existing provider")
)

// NotificationPublisher is the narrow surface this package needs from
// notifications.Service, defined locally so consults doesn't import that
// package directly. Optional (nil by default); set via
// SetNotificationPublisher. Satisfied by *notifications.Service.
type NotificationPublisher interface {
	Publish(ctx context.Context, userID uuid.UUID, notifType string, payload map[string]any) error
}

type Service struct {
	repo        *Repository
	provider    ProviderResolver
	providers   ProviderLookup
	records     *records.Service
	attachments *attachments.Service

	// notify is optional (nil by default) — set via SetNotificationPublisher
	// in main.go. Nil-tolerant wherever it's used, so existing construction
	// paths and tests keep working unchanged.
	notify NotificationPublisher
}

func NewService(repo *Repository, provider ProviderResolver, providers ProviderLookup, recordsService *records.Service, attachmentsService *attachments.Service) *Service {
	return &Service{repo: repo, provider: provider, providers: providers, records: recordsService, attachments: attachmentsService}
}

// SetNotificationPublisher wires the optional in-app notification publisher.
func (s *Service) SetNotificationPublisher(p NotificationPublisher) { s.notify = p }

type CreateInput struct {
	PatientID        uuid.UUID
	TargetProviderID uuid.UUID
	Reason           string
}

// Create resolves the actor's own provider id as requesting_provider_id,
// validates the patient and target provider both exist, and inserts the
// consultation in status=requested.
func (s *Service) Create(ctx context.Context, actorUserID uuid.UUID, in CreateInput) (Consultation, error) {
	requestingProviderID, err := s.provider.ResolveProviderID(ctx, actorUserID)
	if err != nil {
		return Consultation{}, fmt.Errorf("resolve provider identity: %w", err)
	}

	if _, err := s.records.GetPatient(ctx, in.PatientID); err != nil {
		return Consultation{}, fmt.Errorf("resolve patient: %w", err)
	}

	if _, err := s.providers.GetProvider(ctx, in.TargetProviderID); err != nil {
		if errors.Is(err, facilities.ErrProviderNotFound) {
			return Consultation{}, ErrTargetProviderInvalid
		}
		return Consultation{}, fmt.Errorf("resolve target provider: %w", err)
	}

	return s.repo.Create(ctx, CreateConsultationParams{
		PatientID: in.PatientID, RequestingProviderID: requestingProviderID,
		TargetProviderID: in.TargetProviderID, Reason: in.Reason,
	})
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (Consultation, error) {
	return s.repo.FindByID(ctx, id)
}

// ResolveProviderID exposes the caller's own provider identity — used by
// the handler's inbox endpoint to resolve whose consultations to list and
// to compute each item's incoming/outgoing direction.
func (s *Service) ResolveProviderID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	return s.provider.ResolveProviderID(ctx, userID)
}

// ListForProvider returns every consultation where providerID is either
// the requesting or the target provider (the underlying query already does
// this OR), most recently updated first.
func (s *Service) ListForProvider(ctx context.Context, providerID uuid.UUID) ([]Consultation, error) {
	return s.repo.ListForProvider(ctx, providerID)
}

// ListAll is the system_admin oversight path — a platform administrator has
// no provider identity of their own to scope ListForProvider by, so this
// returns every consultation instead. Restricting who may call this is the
// handler's job (see Handler.inbox).
func (s *Service) ListAll(ctx context.Context) ([]Consultation, error) {
	return s.repo.ListAll(ctx)
}

// SearchForProvider is ListForProvider's text-filtered counterpart, reusing
// the exact same party scoping — see search.Service.Search.
func (s *Service) SearchForProvider(ctx context.Context, providerID uuid.UUID, query string, limit int) ([]Consultation, error) {
	return s.repo.SearchForProvider(ctx, providerID, query, limit)
}

// SearchAll is ListAll's text-filtered counterpart — system_admin oversight
// path only, restricting who may call it is the caller's job (see
// search.Service.Search).
func (s *Service) SearchAll(ctx context.Context, query string, limit int) ([]Consultation, error) {
	return s.repo.SearchAll(ctx, query, limit)
}

func (s *Service) ListMessages(ctx context.Context, consultationID uuid.UUID) ([]Message, error) {
	return s.repo.ListMessages(ctx, consultationID)
}

// CountPendingConsultsForProvider implements dashboard.ConsultPendingCounter
// — how many consultations are awaiting providerID's first reply as the
// target provider.
func (s *Service) CountPendingConsultsForProvider(ctx context.Context, providerID uuid.UUID) (int64, error) {
	return s.repo.CountPendingForProvider(ctx, providerID)
}

// IsParty reports whether actorUserID/actorRole is allowed to view or act
// on consultation c: system_admin unconditionally, or either named
// provider on the consultation (resolved from the actor's own user id).
// The actual membership decision is the pure isParty helper below, kept
// separate so it's testable without a ProviderResolver/DB round trip.
func (s *Service) IsParty(ctx context.Context, c Consultation, actorUserID uuid.UUID, actorRole platform.Role) (bool, error) {
	if actorRole == platform.RoleSystemAdmin {
		return true, nil
	}
	actorProviderID, err := s.provider.ResolveProviderID(ctx, actorUserID)
	if err != nil {
		return false, fmt.Errorf("resolve provider identity: %w", err)
	}
	return isParty(c, actorProviderID, actorRole), nil
}

// isParty is the pure "who may access this consultation" rule: system_admin
// unconditionally, or a provider whose id matches either the requesting or
// target provider on c. Split out from IsParty so it can be unit-tested
// directly against a table of (consultation, provider id, role) inputs
// without standing up a ProviderResolver or a database.
func isParty(c Consultation, actorProviderID uuid.UUID, actorRole platform.Role) bool {
	if actorRole == platform.RoleSystemAdmin {
		return true
	}
	return actorProviderID == c.RequestingProviderID || actorProviderID == c.TargetProviderID
}

// PostMessage validates attachmentIDs (if any) each belong to the
// consultation's patient, inserts the message, and — if the consultation is
// currently "requested" and the sender is the target provider (i.e. this is
// the target's first reply, not the requester following up) — transitions
// the consultation to "responded". Callers must have already verified the
// actor is a party to the consultation (see IsParty); PostMessage does not
// re-check that here since who *sent* it is what decides the status
// transition, not whether they were allowed to.
func (s *Service) PostMessage(ctx context.Context, actorUserID uuid.UUID, consultationID uuid.UUID, body string, attachmentIDs []uuid.UUID) (Message, error) {
	c, err := s.repo.FindByID(ctx, consultationID)
	if err != nil {
		return Message{}, err
	}
	if c.Status == StatusClosed {
		return Message{}, ErrConsultClosed
	}

	for _, attachmentID := range attachmentIDs {
		a, err := s.attachments.Get(ctx, attachmentID)
		if err != nil {
			return Message{}, fmt.Errorf("resolve attachment %s: %w", attachmentID, err)
		}
		if a.PatientID != c.PatientID {
			return Message{}, fmt.Errorf("attachment %s does not belong to this consultation's patient", attachmentID)
		}
	}

	message, err := s.repo.CreateMessage(ctx, CreateMessageParams{
		ConsultationID: consultationID, SenderID: actorUserID, Body: body, AttachmentIDs: attachmentIDs,
	})
	if err != nil {
		return Message{}, err
	}

	if c.Status == StatusRequested {
		actorProviderID, err := s.provider.ResolveProviderID(ctx, actorUserID)
		if err == nil && shouldRespond(c, actorProviderID) {
			if _, err := s.repo.UpdateStatus(ctx, consultationID, StatusResponded); err != nil {
				return Message{}, err
			}
		}
	}

	s.notifyOtherParty(ctx, c, actorUserID)

	return message, nil
}

// notifyOtherParty publishes a "consult_message" notification to whichever
// of the consultation's two providers did NOT send this message. Never
// returns an error — a failed or skipped notification must never block
// sending the message itself (same philosophy as audit.Logger.Record).
func (s *Service) notifyOtherParty(ctx context.Context, c Consultation, actorUserID uuid.UUID) {
	if s.notify == nil {
		return
	}
	actorProviderID, err := s.provider.ResolveProviderID(ctx, actorUserID)
	if err != nil {
		slog.Error("failed to resolve sender's provider identity for notification", "error", err, "consultation_id", c.ID)
		return
	}
	otherProviderID := c.TargetProviderID
	if actorProviderID == c.TargetProviderID {
		otherProviderID = c.RequestingProviderID
	}
	other, err := s.providers.GetProvider(ctx, otherProviderID)
	if err != nil {
		slog.Error("failed to resolve other party's provider for notification", "error", err, "consultation_id", c.ID)
		return
	}
	err = s.notify.Publish(ctx, other.UserID, "consult_message", map[string]any{
		"consultation_id": c.ID.String(), "sender_id": actorUserID.String(),
	})
	if err != nil {
		slog.Error("failed to publish consult message notification", "error", err, "consultation_id", c.ID)
	}
}

// shouldRespond is the pure "does this message flip requested -> responded"
// rule: only the *target* provider's first reply advances the status — the
// requester following up (e.g. adding more context before the target has
// answered) must not. c.Status is deliberately not checked here (the only
// caller already guards on StatusRequested); this isolates just the
// "whose message counts as the response" decision for unit testing.
func shouldRespond(c Consultation, senderProviderID uuid.UUID) bool {
	return senderProviderID == c.TargetProviderID
}

// Close is terminal — either party (or system_admin, via the handler's
// party check) can close a consultation from any non-closed state; there
// is no reopening.
func (s *Service) Close(ctx context.Context, consultationID uuid.UUID) (Consultation, error) {
	c, err := s.repo.FindByID(ctx, consultationID)
	if err != nil {
		return Consultation{}, err
	}
	if c.Status == StatusClosed {
		return c, nil
	}
	return s.repo.UpdateStatus(ctx, consultationID, StatusClosed)
}
