package notifications

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Publish is the single write path every other module calls to raise an
// in-app notification — see the Publisher interface each of
// referrals/consults/messaging/records defines locally against this same
// method signature, wired via each package's SetNotificationPublisher setter
// in cmd/api/main.go.
//
// payload is marshaled to JSON here so callers can pass a plain map rather
// than pre-encoding it themselves.
func (s *Service) Publish(ctx context.Context, userID uuid.UUID, notifType string, payload map[string]any) error {
	encoded, err := marshalPayload(payload)
	if err != nil {
		return fmt.Errorf("marshal notification payload: %w", err)
	}
	_, err = s.repo.Create(ctx, userID, notifType, encoded)
	return err
}

// marshalPayload is the pure "map -> JSON bytes" step split out of Publish so
// it's unit-testable without a database — a nil map encodes as "{}" rather
// than the literal string "null", matching the migration's
// `payload JSONB NOT NULL DEFAULT '{}'::jsonb` default.
func marshalPayload(payload map[string]any) ([]byte, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	return json.Marshal(payload)
}

func (s *Service) ListForUser(ctx context.Context, userID uuid.UUID) ([]Notification, error) {
	return s.repo.ListForUser(ctx, userID)
}

func (s *Service) CountUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.repo.CountUnread(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, userID, notificationID uuid.UUID) error {
	return s.repo.MarkRead(ctx, userID, notificationID)
}

func (s *Service) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return s.repo.MarkAllRead(ctx, userID)
}

func (s *Service) GetPreferences(ctx context.Context, userID uuid.UUID) ([]Preference, error) {
	return s.repo.GetPreferences(ctx, userID)
}

func (s *Service) SetPreference(ctx context.Context, userID uuid.UUID, channel string, enabled bool) (Preference, error) {
	return s.repo.UpsertPreference(ctx, userID, channel, enabled)
}
