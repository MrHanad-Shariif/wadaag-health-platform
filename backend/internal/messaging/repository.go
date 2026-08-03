package messaging

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"github.com/wadaag/health-platform/backend/internal/platform/sqlcgen"
)

var (
	ErrConversationNotFound = errors.New("conversation not found")
	ErrMemberNotFound       = errors.New("conversation member not found")
)

type Repository struct {
	q *sqlcgen.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{q: sqlcgen.New(db)}
}

func (r *Repository) CreateConversation(ctx context.Context, convType Type, createdBy uuid.UUID) (Conversation, error) {
	row, err := r.q.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		Type:      sqlcgen.ConversationType(convType),
		CreatedBy: platform.PgUUID(createdBy),
	})
	if err != nil {
		return Conversation{}, fmt.Errorf("insert conversation: %w", err)
	}
	return fromConversationRow(row), nil
}

func (r *Repository) AddMember(ctx context.Context, conversationID, userID uuid.UUID) error {
	if err := r.q.AddConversationMember(ctx, sqlcgen.AddConversationMemberParams{
		ConversationID: platform.PgUUID(conversationID),
		UserID:         platform.PgUUID(userID),
	}); err != nil {
		return fmt.Errorf("insert conversation member: %w", err)
	}
	return nil
}

func (r *Repository) FindDirectBetween(ctx context.Context, userA, userB uuid.UUID) (Conversation, error) {
	row, err := r.q.FindDirectConversationBetween(ctx, sqlcgen.FindDirectConversationBetweenParams{
		UserID: platform.PgUUID(userA), UserID_2: platform.PgUUID(userB),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Conversation{}, ErrConversationNotFound
	}
	if err != nil {
		return Conversation{}, fmt.Errorf("query direct conversation: %w", err)
	}
	return fromConversationRow(row), nil
}

func (r *Repository) ListForUser(ctx context.Context, userID uuid.UUID) ([]Conversation, error) {
	rows, err := r.q.ListConversationsForUser(ctx, platform.PgUUID(userID))
	if err != nil {
		return nil, fmt.Errorf("list conversations for user: %w", err)
	}
	out := make([]Conversation, len(rows))
	for i, row := range rows {
		out[i] = fromConversationListRow(row)
	}
	return out, nil
}

func (r *Repository) FindMember(ctx context.Context, conversationID, userID uuid.UUID) (Member, error) {
	row, err := r.q.FindConversationMember(ctx, sqlcgen.FindConversationMemberParams{
		ConversationID: platform.PgUUID(conversationID), UserID: platform.PgUUID(userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Member{}, ErrMemberNotFound
	}
	if err != nil {
		return Member{}, fmt.Errorf("query conversation member: %w", err)
	}
	return fromMemberRow(row), nil
}

func (r *Repository) ListMembers(ctx context.Context, conversationID uuid.UUID) ([]Member, error) {
	rows, err := r.q.ListConversationMembers(ctx, platform.PgUUID(conversationID))
	if err != nil {
		return nil, fmt.Errorf("list conversation members: %w", err)
	}
	out := make([]Member, len(rows))
	for i, row := range rows {
		out[i] = fromMemberRow(row)
	}
	return out, nil
}

func (r *Repository) TouchConversation(ctx context.Context, conversationID uuid.UUID) error {
	if err := r.q.TouchConversation(ctx, platform.PgUUID(conversationID)); err != nil {
		return fmt.Errorf("touch conversation: %w", err)
	}
	return nil
}

type CreateMessageParams struct {
	ConversationID uuid.UUID
	SenderID       uuid.UUID
	Body           string
	AttachmentIDs  []uuid.UUID
}

func (r *Repository) CreateMessage(ctx context.Context, p CreateMessageParams) (Message, error) {
	row, err := r.q.CreateMessage(ctx, sqlcgen.CreateMessageParams{
		ConversationID: platform.PgUUID(p.ConversationID),
		SenderID:       platform.PgUUID(p.SenderID),
		Body:           p.Body,
		AttachmentIds:  toPgUUIDs(p.AttachmentIDs),
	})
	if err != nil {
		return Message{}, fmt.Errorf("insert message: %w", err)
	}
	return fromMessageRow(row), nil
}

func (r *Repository) ListRecentMessages(ctx context.Context, conversationID uuid.UUID) ([]Message, error) {
	rows, err := r.q.ListRecentMessages(ctx, platform.PgUUID(conversationID))
	if err != nil {
		return nil, fmt.Errorf("list recent messages: %w", err)
	}
	out := make([]Message, len(rows))
	for i, row := range rows {
		out[i] = fromMessageRow(row)
	}
	return out, nil
}

func (r *Repository) MarkRead(ctx context.Context, conversationID, userID uuid.UUID) error {
	if err := r.q.MarkConversationRead(ctx, sqlcgen.MarkConversationReadParams{
		ConversationID: platform.PgUUID(conversationID), UserID: platform.PgUUID(userID),
	}); err != nil {
		return fmt.Errorf("mark conversation read: %w", err)
	}
	return nil
}

func toPgUUIDs(ids []uuid.UUID) []pgtype.UUID {
	out := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		out[i] = platform.PgUUID(id)
	}
	return out
}

func fromPgUUIDs(ids []pgtype.UUID) []uuid.UUID {
	out := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		out[i] = platform.FromPgUUID(id)
	}
	return out
}

func fromConversationRow(row sqlcgen.Conversation) Conversation {
	return Conversation{
		ID:        platform.FromPgUUID(row.ID),
		Type:      Type(row.Type),
		CreatedBy: platform.FromPgUUID(row.CreatedBy),
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}

// fromConversationListRow maps ListConversationsForUser's row, which carries
// the caller's own last_read_at, a last-message preview, and (for direct
// conversations) the other member's user_id, all from its LATERAL joins.
// LastMessageAt.Valid is used as the "has a message" signal rather than
// trusting LastMessageBody's zero value, since sqlc types that column as a
// plain (non-nullable) string despite the LEFT JOIN — an empty string there
// would otherwise be indistinguishable from "no message yet".
func fromConversationListRow(row sqlcgen.ListConversationsForUserRow) Conversation {
	c := Conversation{
		ID:          platform.FromPgUUID(row.ID),
		Type:        Type(row.Type),
		CreatedBy:   platform.FromPgUUID(row.CreatedBy),
		OtherUserID: platform.FromPgUUIDPtr(row.OtherUserID),
		LastReadAt:  platform.FromPgTimestamptzPtr(row.CallerLastReadAt),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
	if row.LastMessageAt.Valid {
		body := row.LastMessageBody
		at := row.LastMessageAt.Time
		c.LastMessageBody = &body
		c.LastMessageAt = &at
	}
	return c
}

func fromMemberRow(row sqlcgen.ConversationMember) Member {
	return Member{
		ConversationID: platform.FromPgUUID(row.ConversationID),
		UserID:         platform.FromPgUUID(row.UserID),
		LastReadAt:     platform.FromPgTimestamptzPtr(row.LastReadAt),
		JoinedAt:       row.JoinedAt.Time,
	}
}

func fromMessageRow(row sqlcgen.Message) Message {
	return Message{
		ID:             platform.FromPgUUID(row.ID),
		ConversationID: platform.FromPgUUID(row.ConversationID),
		SenderID:       platform.FromPgUUID(row.SenderID),
		Body:           row.Body,
		AttachmentIDs:  fromPgUUIDs(row.AttachmentIds),
		CreatedAt:      row.CreatedAt.Time,
	}
}
