package identity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"github.com/wadaag/health-platform/backend/internal/platform/sqlcgen"
)

var ErrUserNotFound = errors.New("user not found")
var ErrDuplicateUser = errors.New("email or phone already registered")
var ErrRefreshTokenNotFound = errors.New("refresh token not found")
var ErrEmailVerificationTokenNotFound = errors.New("email verification token not found")
var ErrPasswordResetTokenNotFound = errors.New("password reset token not found")

type Repository struct {
	q *sqlcgen.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{q: sqlcgen.New(db)}
}

func (r *Repository) CreateUser(ctx context.Context, email, phone *string, passwordHash string, role platform.Role, fullName *string) (User, error) {
	row, err := r.q.CreateUser(ctx, sqlcgen.CreateUserParams{
		Email:        platform.PgText(email),
		Phone:        platform.PgText(phone),
		PasswordHash: passwordHash,
		Role:         sqlcgen.UserRole(role),
		FullName:     platform.PgText(fullName),
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

// CreateRefreshTokenParams is the repository-facing input for storing a new
// session — device/IP/user-agent are all optional metadata for a future
// "active sessions" screen, not used for any access decision today.
type CreateRefreshTokenParams struct {
	UserID      uuid.UUID
	TokenHash   string
	DeviceLabel *string
	IP          *string
	UserAgent   *string
	ExpiresAt   time.Time
}

func (r *Repository) CreateRefreshToken(ctx context.Context, in CreateRefreshTokenParams) (RefreshToken, error) {
	row, err := r.q.CreateRefreshToken(ctx, sqlcgen.CreateRefreshTokenParams{
		UserID:      platform.PgUUID(in.UserID),
		TokenHash:   in.TokenHash,
		DeviceLabel: platform.PgText(in.DeviceLabel),
		Ip:          platform.PgText(in.IP),
		UserAgent:   platform.PgText(in.UserAgent),
		ExpiresAt:   platform.PgTimestamptz(in.ExpiresAt),
	})
	if err != nil {
		return RefreshToken{}, fmt.Errorf("insert refresh token: %w", err)
	}

	return fromRefreshTokenRow(row), nil
}

func (r *Repository) FindRefreshTokenByHash(ctx context.Context, tokenHash string) (RefreshToken, error) {
	row, err := r.q.FindRefreshTokenByHash(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return RefreshToken{}, ErrRefreshTokenNotFound
	}
	if err != nil {
		return RefreshToken{}, fmt.Errorf("query refresh token: %w", err)
	}

	return fromRefreshTokenRow(row), nil
}

func (r *Repository) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	if err := r.q.RevokeRefreshToken(ctx, platform.PgUUID(id)); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

// RevokeAllRefreshTokensForUser is not wired to any route yet — added for a
// later task (e.g. password reset should invalidate every existing
// session) since it's trivial to add alongside the rest of this table.
func (r *Repository) RevokeAllRefreshTokensForUser(ctx context.Context, userID uuid.UUID) error {
	if err := r.q.RevokeAllRefreshTokensForUser(ctx, platform.PgUUID(userID)); err != nil {
		return fmt.Errorf("revoke refresh tokens for user: %w", err)
	}
	return nil
}

// ListActiveRefreshTokensForUser returns the caller's currently-usable
// sessions (not revoked, not expired), newest first — the data backing an
// "active sessions" screen.
func (r *Repository) ListActiveRefreshTokensForUser(ctx context.Context, userID uuid.UUID) ([]RefreshToken, error) {
	rows, err := r.q.ListActiveRefreshTokensForUser(ctx, platform.PgUUID(userID))
	if err != nil {
		return nil, fmt.Errorf("list refresh tokens: %w", err)
	}

	tokens := make([]RefreshToken, len(rows))
	for i, row := range rows {
		tokens[i] = fromRefreshTokenRow(row)
	}
	return tokens, nil
}

// RevokeRefreshTokenForUser revokes a session by id, but only if it belongs
// to userID — the ownership check happens atomically in the UPDATE's WHERE
// clause rather than via a separate fetch-then-check, so there's no race
// between verifying ownership and revoking. Returns ErrRefreshTokenNotFound
// if no row matched (wrong id, wrong owner, or already revoked) — the
// caller can't distinguish those cases, which is intentional.
func (r *Repository) RevokeRefreshTokenForUser(ctx context.Context, id, userID uuid.UUID) error {
	rows, err := r.q.RevokeRefreshTokenForUser(ctx, sqlcgen.RevokeRefreshTokenForUserParams{
		ID:     platform.PgUUID(id),
		UserID: platform.PgUUID(userID),
	})
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	if rows == 0 {
		return ErrRefreshTokenNotFound
	}
	return nil
}

// CreateEmailVerificationToken stores the hash of a newly issued email
// verification token.
func (r *Repository) CreateEmailVerificationToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (EmailVerificationToken, error) {
	row, err := r.q.CreateEmailVerificationToken(ctx, sqlcgen.CreateEmailVerificationTokenParams{
		UserID:    platform.PgUUID(userID),
		TokenHash: tokenHash,
		ExpiresAt: platform.PgTimestamptz(expiresAt),
	})
	if err != nil {
		return EmailVerificationToken{}, fmt.Errorf("insert email verification token: %w", err)
	}

	return fromEmailVerificationTokenRow(row), nil
}

func (r *Repository) FindEmailVerificationTokenByHash(ctx context.Context, tokenHash string) (EmailVerificationToken, error) {
	row, err := r.q.FindEmailVerificationTokenByHash(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return EmailVerificationToken{}, ErrEmailVerificationTokenNotFound
	}
	if err != nil {
		return EmailVerificationToken{}, fmt.Errorf("query email verification token: %w", err)
	}

	return fromEmailVerificationTokenRow(row), nil
}

func (r *Repository) MarkEmailVerificationTokenUsed(ctx context.Context, id uuid.UUID) error {
	if err := r.q.MarkEmailVerificationTokenUsed(ctx, platform.PgUUID(id)); err != nil {
		return fmt.Errorf("mark email verification token used: %w", err)
	}
	return nil
}

func (r *Repository) MarkUserVerified(ctx context.Context, userID uuid.UUID) error {
	if err := r.q.MarkUserVerified(ctx, platform.PgUUID(userID)); err != nil {
		return fmt.Errorf("mark user verified: %w", err)
	}
	return nil
}

// CreatePasswordResetToken stores the hash of a newly issued password reset
// token.
func (r *Repository) CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (PasswordResetToken, error) {
	row, err := r.q.CreatePasswordResetToken(ctx, sqlcgen.CreatePasswordResetTokenParams{
		UserID:    platform.PgUUID(userID),
		TokenHash: tokenHash,
		ExpiresAt: platform.PgTimestamptz(expiresAt),
	})
	if err != nil {
		return PasswordResetToken{}, fmt.Errorf("insert password reset token: %w", err)
	}

	return fromPasswordResetTokenRow(row), nil
}

func (r *Repository) FindPasswordResetTokenByHash(ctx context.Context, tokenHash string) (PasswordResetToken, error) {
	row, err := r.q.FindPasswordResetTokenByHash(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return PasswordResetToken{}, ErrPasswordResetTokenNotFound
	}
	if err != nil {
		return PasswordResetToken{}, fmt.Errorf("query password reset token: %w", err)
	}

	return fromPasswordResetTokenRow(row), nil
}

func (r *Repository) MarkPasswordResetTokenUsed(ctx context.Context, id uuid.UUID) error {
	if err := r.q.MarkPasswordResetTokenUsed(ctx, platform.PgUUID(id)); err != nil {
		return fmt.Errorf("mark password reset token used: %w", err)
	}
	return nil
}

// UpdateUserPasswordHash overwrites userID's stored password hash — used by
// the reset-password flow after the presented token has been validated,
// and by Service.ChangePassword's self-service counterpart after the
// caller has proven they know their current password.
func (r *Repository) UpdateUserPasswordHash(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	if err := r.q.UpdateUserPasswordHash(ctx, sqlcgen.UpdateUserPasswordHashParams{
		ID:           platform.PgUUID(userID),
		PasswordHash: passwordHash,
	}); err != nil {
		return fmt.Errorf("update user password hash: %w", err)
	}
	return nil
}

// UpdateFullName overwrites userID's stored display name, returning the
// updated row.
func (r *Repository) UpdateFullName(ctx context.Context, userID uuid.UUID, fullName string) (User, error) {
	name := fullName
	row, err := r.q.UpdateUserFullName(ctx, sqlcgen.UpdateUserFullNameParams{
		ID:       platform.PgUUID(userID),
		FullName: platform.PgText(&name),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("update user full name: %w", err)
	}
	return fromRow(row), nil
}

func fromEmailVerificationTokenRow(row sqlcgen.EmailVerificationToken) EmailVerificationToken {
	return EmailVerificationToken{
		ID:        platform.FromPgUUID(row.ID),
		UserID:    platform.FromPgUUID(row.UserID),
		TokenHash: row.TokenHash,
		CreatedAt: row.CreatedAt.Time,
		ExpiresAt: row.ExpiresAt.Time,
		UsedAt:    platform.FromPgTimestamptzPtr(row.UsedAt),
	}
}

func fromPasswordResetTokenRow(row sqlcgen.PasswordResetToken) PasswordResetToken {
	return PasswordResetToken{
		ID:        platform.FromPgUUID(row.ID),
		UserID:    platform.FromPgUUID(row.UserID),
		TokenHash: row.TokenHash,
		CreatedAt: row.CreatedAt.Time,
		ExpiresAt: row.ExpiresAt.Time,
		UsedAt:    platform.FromPgTimestamptzPtr(row.UsedAt),
	}
}

func fromRefreshTokenRow(row sqlcgen.RefreshToken) RefreshToken {
	return RefreshToken{
		ID:          platform.FromPgUUID(row.ID),
		UserID:      platform.FromPgUUID(row.UserID),
		TokenHash:   row.TokenHash,
		DeviceLabel: platform.FromPgText(row.DeviceLabel),
		IP:          platform.FromPgText(row.Ip),
		UserAgent:   platform.FromPgText(row.UserAgent),
		CreatedAt:   row.CreatedAt.Time,
		ExpiresAt:   row.ExpiresAt.Time,
		RevokedAt:   platform.FromPgTimestamptzPtr(row.RevokedAt),
	}
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
		VerifiedAt:   platform.FromPgTimestamptzPtr(row.VerifiedAt),
		FullName:     platform.FromPgText(row.FullName),
	}
}
