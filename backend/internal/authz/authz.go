package authz

import (
	"time"

	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/platform"
)

type Permission struct {
	ID       uuid.UUID
	Resource string
	Action   string
}

func (p Permission) Code() string {
	return p.Resource + ":" + p.Action
}

type Role struct {
	ID          uuid.UUID
	Name        string
	Description *string
	FullAccess  bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type RoleWithPermissions struct {
	Role
	Permissions []Permission
}

// User is the Authentication module's view of an account — the legacy
// Role is included for visibility (it's what still gates records/
// referrals/facilities) even though this module only manages RoleID.
type User struct {
	ID         uuid.UUID
	Email      *string
	Phone      *string
	LegacyRole platform.Role
	Status     string
	CreatedAt  time.Time
	RoleID     *uuid.UUID
	RoleName   *string
}
