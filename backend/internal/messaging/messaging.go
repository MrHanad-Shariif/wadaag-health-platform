// Package messaging implements secure, general-purpose two-way messaging
// between any two authenticated users — not scoped to a single patient the
// way records/referrals/consults are. REST is the only write path (sending
// a message is always POST /conversations/{id}/messages); the WebSocket
// endpoint (see hub.go, handler.go's serveWS) is receive-only push: when a
// message is created via the REST endpoint, the handler also broadcasts it
// to any currently-connected WS clients in that conversation, while a
// disconnected client can still poll the REST GET endpoint as a fallback.
//
// Unlike records/referrals/consents, this module is deliberately NOT wired
// through consent.Middleware or audit.Logger — there is no single patient_id
// to scope by, and the roadmap doesn't call for audit-gating messaging the
// way patient-record access is gated. The only authorization check anywhere
// in this package is "is the caller a member of this conversation" (see
// Service.IsMember).
//
// attachment_ids is accepted and stored as-is (a plain []uuid.UUID) with no
// validation against the attachments table — Phase 3's attachments module is
// patient-scoped (POST /attachments/patients/{patientID}) and general
// messaging has no patient context to validate against, so that integration
// is deliberately left for a later pass.
package messaging

import (
	"time"

	"github.com/google/uuid"
)

type Type string

const (
	TypeDirect Type = "direct"
	TypeGroup  Type = "group"
)

// Conversation is a plain domain struct. LastMessageBody/LastMessageAt/
// OtherUserID come from ListConversationsForUser's LATERAL joins (nil if
// the conversation has no messages yet / isn't a direct 1:1 conversation);
// LastReadAt is the CALLING user's own last_read_at for this conversation —
// it is not a property of the conversation itself, just whichever caller
// asked for it.
type Conversation struct {
	ID              uuid.UUID
	Type            Type
	CreatedBy       uuid.UUID
	OtherUserID     *uuid.UUID
	LastMessageBody *string
	LastMessageAt   *time.Time
	LastReadAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Member struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID
	LastReadAt     *time.Time
	JoinedAt       time.Time
}

type Message struct {
	ID             uuid.UUID
	ConversationID uuid.UUID
	SenderID       uuid.UUID
	Body           string
	AttachmentIDs  []uuid.UUID
	CreatedAt      time.Time
}
