package messaging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

// ErrNotMember is the only authorization error in this package — any
// authenticated user, any role, can start a conversation with any other
// user and send messages in a conversation they're a member of; the only
// check anywhere is "is the caller a member of this conversation".
var ErrNotMember = errors.New("caller is not a member of this conversation")

// NotificationPublisher is the narrow surface this package needs from
// notifications.Service, defined locally so messaging doesn't import that
// package directly. Optional (nil by default); set via
// SetNotificationPublisher. Satisfied by *notifications.Service.
type NotificationPublisher interface {
	Publish(ctx context.Context, userID uuid.UUID, notifType string, payload map[string]any) error
}

type Service struct {
	repo *Repository

	// notify is optional (nil by default) — set via SetNotificationPublisher
	// in main.go. Nil-tolerant wherever it's used, so existing construction
	// paths and tests keep working unchanged.
	notify NotificationPublisher
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// SetNotificationPublisher wires the optional in-app notification publisher.
func (s *Service) SetNotificationPublisher(p NotificationPublisher) { s.notify = p }

// CreateDirect returns the existing 1:1 conversation between actorUserID and
// otherUserID if one already exists (so repeatedly "messaging" the same
// person doesn't spawn duplicate conversations), or creates a new one and
// adds both users as members.
//
// Group conversations are schema-supported (conversations.type) but this
// pass only implements direct conversation creation — no group-creation UI
// is being built yet.
func (s *Service) CreateDirect(ctx context.Context, actorUserID, otherUserID uuid.UUID) (Conversation, error) {
	existing, err := s.repo.FindDirectBetween(ctx, actorUserID, otherUserID)
	if err == nil {
		existing.OtherUserID = &otherUserID
		return s.withLastRead(ctx, existing, actorUserID)
	}
	if !errors.Is(err, ErrConversationNotFound) {
		return Conversation{}, fmt.Errorf("look up existing direct conversation: %w", err)
	}

	conversation, err := s.repo.CreateConversation(ctx, TypeDirect, actorUserID)
	if err != nil {
		return Conversation{}, err
	}
	if err := s.repo.AddMember(ctx, conversation.ID, actorUserID); err != nil {
		return Conversation{}, err
	}
	if err := s.repo.AddMember(ctx, conversation.ID, otherUserID); err != nil {
		return Conversation{}, err
	}
	conversation.OtherUserID = &otherUserID
	return conversation, nil
}

// ListForUser returns every conversation the given user belongs to, most
// recently updated first. The caller's own last_read_at and (for direct
// conversations) the other member's user_id are already attached by the
// repository's query — no per-conversation follow-up lookup needed.
func (s *Service) ListForUser(ctx context.Context, userID uuid.UUID) ([]Conversation, error) {
	return s.repo.ListForUser(ctx, userID)
}

// withLastRead attaches the caller's own last_read_at to a single
// conversation — used by CreateDirect, whose return value goes straight to
// the handler rather than through ListForUser.
func (s *Service) withLastRead(ctx context.Context, c Conversation, userID uuid.UUID) (Conversation, error) {
	member, err := s.repo.FindMember(ctx, c.ID, userID)
	if err != nil {
		return Conversation{}, fmt.Errorf("resolve caller's membership: %w", err)
	}
	c.LastReadAt = member.LastReadAt
	return c, nil
}

// IsMember reports whether userID belongs to conversationID — the sole
// authorization check used across this package's handler.
func (s *Service) IsMember(ctx context.Context, conversationID, userID uuid.UUID) (bool, error) {
	_, err := s.repo.FindMember(ctx, conversationID, userID)
	if errors.Is(err, ErrMemberNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ListMembers returns every member of a conversation — used by the WS hub
// to know who to notify of a new message/typing/read event. Does not itself
// verify the caller's membership; callers must do that first (see
// Handler.requireMember).
func (s *Service) ListMembers(ctx context.Context, conversationID uuid.UUID) ([]Member, error) {
	return s.repo.ListMembers(ctx, conversationID)
}

// PostMessage verifies actorUserID is a member of conversationID, inserts
// the message, and touches the conversation's updated_at so the list stays
// sorted by recency.
func (s *Service) PostMessage(ctx context.Context, actorUserID, conversationID uuid.UUID, body string, attachmentIDs []uuid.UUID) (Message, error) {
	isMember, err := s.IsMember(ctx, conversationID, actorUserID)
	if err != nil {
		return Message{}, err
	}
	if !isMember {
		return Message{}, ErrNotMember
	}

	message, err := s.repo.CreateMessage(ctx, CreateMessageParams{
		ConversationID: conversationID, SenderID: actorUserID, Body: body, AttachmentIDs: attachmentIDs,
	})
	if err != nil {
		return Message{}, err
	}
	if err := s.repo.TouchConversation(ctx, conversationID); err != nil {
		return Message{}, err
	}

	s.notifyOtherMembers(ctx, conversationID, actorUserID)

	return message, nil
}

// notifyOtherMembers publishes a "new_message" notification to every member
// of conversationID other than the sender (for direct conversations, that's
// the one other member). Never returns an error — a failed or skipped
// notification must never block sending the message itself.
func (s *Service) notifyOtherMembers(ctx context.Context, conversationID, actorUserID uuid.UUID) {
	if s.notify == nil {
		return
	}
	members, err := s.repo.ListMembers(ctx, conversationID)
	if err != nil {
		slog.Error("failed to list conversation members for notification", "error", err, "conversation_id", conversationID)
		return
	}
	for _, member := range members {
		if member.UserID == actorUserID {
			continue
		}
		err := s.notify.Publish(ctx, member.UserID, "new_message", map[string]any{
			"conversation_id": conversationID.String(), "sender_id": actorUserID.String(),
		})
		if err != nil {
			slog.Error("failed to publish new message notification", "error", err, "conversation_id", conversationID, "recipient", member.UserID)
		}
	}
}

// ListMessages verifies actorUserID is a member of conversationID before
// returning the most recent messages, oldest-first.
func (s *Service) ListMessages(ctx context.Context, actorUserID, conversationID uuid.UUID) ([]Message, error) {
	isMember, err := s.IsMember(ctx, conversationID, actorUserID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotMember
	}
	return s.repo.ListRecentMessages(ctx, conversationID)
}

// MarkRead verifies actorUserID is a member of conversationID before
// updating their last_read_at.
func (s *Service) MarkRead(ctx context.Context, actorUserID, conversationID uuid.UUID) error {
	isMember, err := s.IsMember(ctx, conversationID, actorUserID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrNotMember
	}
	return s.repo.MarkRead(ctx, conversationID, actorUserID)
}
