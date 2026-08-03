package notifications

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"github.com/wadaag/health-platform/backend/internal/platform/sqlcgen"
)

type Repository struct {
	q *sqlcgen.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{q: sqlcgen.New(db)}
}

func (r *Repository) Create(ctx context.Context, userID uuid.UUID, notifType string, payload []byte) (Notification, error) {
	row, err := r.q.CreateNotification(ctx, sqlcgen.CreateNotificationParams{
		UserID: platform.PgUUID(userID), Type: notifType, Payload: payload,
	})
	if err != nil {
		return Notification{}, fmt.Errorf("insert notification: %w", err)
	}
	return fromNotificationRow(row), nil
}

func (r *Repository) ListForUser(ctx context.Context, userID uuid.UUID) ([]Notification, error) {
	rows, err := r.q.ListNotificationsForUser(ctx, platform.PgUUID(userID))
	if err != nil {
		return nil, fmt.Errorf("list notifications for user: %w", err)
	}
	out := make([]Notification, len(rows))
	for i, row := range rows {
		out[i] = fromNotificationRow(row)
	}
	return out, nil
}

func (r *Repository) CountUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	count, err := r.q.CountUnreadNotifications(ctx, platform.PgUUID(userID))
	if err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return count, nil
}

func (r *Repository) MarkRead(ctx context.Context, userID, notificationID uuid.UUID) error {
	if err := r.q.MarkNotificationRead(ctx, sqlcgen.MarkNotificationReadParams{
		ID: platform.PgUUID(notificationID), UserID: platform.PgUUID(userID),
	}); err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	return nil
}

func (r *Repository) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	if err := r.q.MarkAllNotificationsRead(ctx, platform.PgUUID(userID)); err != nil {
		return fmt.Errorf("mark all notifications read: %w", err)
	}
	return nil
}

func (r *Repository) GetPreferences(ctx context.Context, userID uuid.UUID) ([]Preference, error) {
	rows, err := r.q.GetNotificationPreferences(ctx, platform.PgUUID(userID))
	if err != nil {
		return nil, fmt.Errorf("get notification preferences: %w", err)
	}
	out := make([]Preference, len(rows))
	for i, row := range rows {
		out[i] = fromPreferenceRow(row)
	}
	return out, nil
}

func (r *Repository) UpsertPreference(ctx context.Context, userID uuid.UUID, channel string, enabled bool) (Preference, error) {
	row, err := r.q.UpsertNotificationPreference(ctx, sqlcgen.UpsertNotificationPreferenceParams{
		UserID: platform.PgUUID(userID), Channel: channel, Enabled: enabled,
	})
	if err != nil {
		return Preference{}, fmt.Errorf("upsert notification preference: %w", err)
	}
	return fromPreferenceRow(row), nil
}

func fromNotificationRow(row sqlcgen.Notification) Notification {
	return Notification{
		ID:        platform.FromPgUUID(row.ID),
		UserID:    platform.FromPgUUID(row.UserID),
		Type:      row.Type,
		Payload:   row.Payload,
		ReadAt:    platform.FromPgTimestamptzPtr(row.ReadAt),
		CreatedAt: row.CreatedAt.Time,
	}
}

func fromPreferenceRow(row sqlcgen.NotificationPreference) Preference {
	return Preference{
		UserID:  platform.FromPgUUID(row.UserID),
		Channel: row.Channel,
		Enabled: row.Enabled,
	}
}
