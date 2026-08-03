package identity

import (
	"time"

	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/platform"
)

type User struct {
	ID           uuid.UUID
	Email        *string
	Phone        *string
	PasswordHash string
	Role         platform.Role
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	VerifiedAt   *time.Time
	FullName     *string
}

// RefreshToken is a DB-backed session: the raw opaque token handed to the
// client is never stored, only its SHA-256 hash — so a leaked database
// dump doesn't itself grant sessions. Revoking (logout, rotation) sets
// RevokedAt instead of deleting the row, keeping an audit trail.
type RefreshToken struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	TokenHash   string
	DeviceLabel *string
	IP          *string
	UserAgent   *string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	RevokedAt   *time.Time
}

// EmailVerificationToken mirrors RefreshToken's shape: the raw opaque token
// handed to the client (via the logged "email") is never stored, only its
// SHA-256 hash. UsedAt is set once the token has been redeemed, so a
// captured/replayed link can't verify twice.
type EmailVerificationToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
}

// PasswordResetToken mirrors EmailVerificationToken's shape: the raw opaque
// token handed to the caller (via the logged "email") is never stored, only
// its SHA-256 hash. UsedAt is set once the token has been redeemed, so a
// captured/replayed reset link can't be used twice.
type PasswordResetToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
}
