package identity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"github.com/wadaag/health-platform/backend/internal/platform/sqlcgen"
)

// ErrProfileNotFound signals no user_profiles row exists yet for a user —
// not itself an error condition callers should surface: Service.GetProfile
// catches it and returns a zero-value UserProfile instead (see the row
// lifecycle note in the migration).
var ErrProfileNotFound = errors.New("user profile not found")

func (r *Repository) GetUserProfile(ctx context.Context, userID uuid.UUID) (UserProfile, error) {
	row, err := r.q.GetUserProfile(ctx, platform.PgUUID(userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return UserProfile{}, ErrProfileNotFound
	}
	if err != nil {
		return UserProfile{}, fmt.Errorf("query user profile: %w", err)
	}

	return fromProfileRow(row), nil
}

// UpsertProfileFields overwrites the caller-editable profile fields
// (bio/languages/availability_status), creating the row if it doesn't exist
// yet. Callers wanting partial-update semantics (only overwrite fields the
// caller actually supplied) must pre-merge against the current row — see
// Service.UpdateProfile / mergeProfileUpdate — this method itself always
// writes exactly what it's given.
func (r *Repository) UpsertProfileFields(ctx context.Context, userID uuid.UUID, bio *string, languages []string, availabilityStatus *string) (UserProfile, error) {
	if languages == nil {
		languages = []string{}
	}

	row, err := r.q.UpsertUserProfile(ctx, sqlcgen.UpsertUserProfileParams{
		UserID:             platform.PgUUID(userID),
		Bio:                platform.PgText(bio),
		Languages:          languages,
		AvailabilityStatus: platform.PgText(availabilityStatus),
	})
	if err != nil {
		return UserProfile{}, fmt.Errorf("upsert user profile: %w", err)
	}

	return fromProfileRow(row), nil
}

// UpsertProfilePhoto overwrites just photo_url, creating the row if it
// doesn't exist yet — kept separate from UpsertProfileFields so a photo
// upload never has to read-then-write the other profile fields.
func (r *Repository) UpsertProfilePhoto(ctx context.Context, userID uuid.UUID, photoURL string) (UserProfile, error) {
	row, err := r.q.UpsertUserProfilePhoto(ctx, sqlcgen.UpsertUserProfilePhotoParams{
		UserID:   platform.PgUUID(userID),
		PhotoUrl: platform.PgText(&photoURL),
	})
	if err != nil {
		return UserProfile{}, fmt.Errorf("upsert user profile photo: %w", err)
	}

	return fromProfileRow(row), nil
}

// SetProfilePresence overwrites is_online/last_seen_at, creating the row if
// it doesn't exist yet — called from Service.Login/Service.Logout, never
// directly from a handler.
func (r *Repository) SetProfilePresence(ctx context.Context, userID uuid.UUID, online bool, lastSeenAt time.Time) error {
	if err := r.q.SetUserProfilePresence(ctx, sqlcgen.SetUserProfilePresenceParams{
		UserID:     platform.PgUUID(userID),
		IsOnline:   online,
		LastSeenAt: platform.PgTimestamptz(lastSeenAt),
	}); err != nil {
		return fmt.Errorf("set user profile presence: %w", err)
	}
	return nil
}

func fromProfileRow(row sqlcgen.UserProfile) UserProfile {
	return UserProfile{
		UserID:             platform.FromPgUUID(row.UserID),
		PhotoURL:           platform.FromPgText(row.PhotoUrl),
		Bio:                platform.FromPgText(row.Bio),
		Languages:          row.Languages,
		AvailabilityStatus: platform.FromPgText(row.AvailabilityStatus),
		IsOnline:           row.IsOnline,
		LastSeenAt:         platform.FromPgTimestamptzPtr(row.LastSeenAt),
		CreatedAt:          row.CreatedAt.Time,
		UpdatedAt:          row.UpdatedAt.Time,
	}
}
