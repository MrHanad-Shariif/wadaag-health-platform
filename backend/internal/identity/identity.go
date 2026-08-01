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
}
