package identity

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"github.com/wadaag/health-platform/backend/internal/platform/sqlcgen"
)

var ErrUserNotFound = errors.New("user not found")
var ErrDuplicateUser = errors.New("email or phone already registered")

type Repository struct {
	q *sqlcgen.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{q: sqlcgen.New(db)}
}

func (r *Repository) CreateUser(ctx context.Context, email, phone *string, passwordHash string, role platform.Role) (User, error) {
	row, err := r.q.CreateUser(ctx, sqlcgen.CreateUserParams{
		Email:        platform.PgText(email),
		Phone:        platform.PgText(phone),
		PasswordHash: passwordHash,
		Role:         sqlcgen.UserRole(role),
	})
	if err != nil {
		if platform.IsUniqueViolation(err) {
			return User{}, ErrDuplicateUser
		}
		return User{}, fmt.Errorf("insert user: %w", err)
	}

	return fromRow(row), nil
}

func (r *Repository) FindByEmailOrPhone(ctx context.Context, identifier string) (User, error) {
	row, err := r.q.FindUserByEmailOrPhone(ctx, platform.PgText(&identifier))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("query user: %w", err)
	}

	return fromRow(row), nil
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (User, error) {
	row, err := r.q.FindUserByID(ctx, platform.PgUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("query user: %w", err)
	}

	return fromRow(row), nil
}

func fromRow(row sqlcgen.User) User {
	return User{
		ID:           platform.FromPgUUID(row.ID),
		Email:        platform.FromPgText(row.Email),
		Phone:        platform.FromPgText(row.Phone),
		PasswordHash: row.PasswordHash,
		Role:         platform.Role(row.Role),
		Status:       row.Status,
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}
}
