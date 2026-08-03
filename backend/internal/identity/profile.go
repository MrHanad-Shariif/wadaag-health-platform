package identity

import (
	"time"

	"github.com/google/uuid"
)

// UserProfile is the self-service "manage my own profile" data: separate
// from User (identity.go) since auth fields and profile fields are
// different concerns with different write patterns — see the migration
// comment in db/migrations/0010_create_user_profiles.up.sql.
//
// A UserProfile value with no backing row (nothing ever PATCHed or
// uploaded) is represented as the zero-value fields below with UserID set —
// see Service.GetProfile — never as a "not found" error to the caller.
type UserProfile struct {
	UserID             uuid.UUID
	PhotoURL           *string
	Bio                *string
	Languages          []string
	AvailabilityStatus *string
	IsOnline           bool
	LastSeenAt         *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
