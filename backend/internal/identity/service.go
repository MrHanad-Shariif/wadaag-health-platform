package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid email/phone or password")

// ErrInvalidRefreshToken covers every reason a presented refresh token
// isn't usable — not found, already revoked, or expired. Deliberately not
// distinguished in the response so a caller can't probe which of those it
// is.
var ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")

// ErrInvalidVerificationToken covers every reason a presented email
// verification token isn't usable — not found, already used, or expired —
// deliberately not distinguished in the response for the same reason as
// ErrInvalidRefreshToken.
var ErrInvalidVerificationToken = errors.New("invalid or expired verification token")

// ErrAlreadyVerified is returned when a verification email is requested for
// an account that's already verified.
var ErrAlreadyVerified = errors.New("email already verified")

// ErrInvalidResetToken covers every reason a presented password reset token
// isn't usable — not found, already used, or expired — deliberately not
// distinguished in the response for the same reason as
// ErrInvalidVerificationToken.
var ErrInvalidResetToken = errors.New("invalid or expired reset token")

// emailVerificationTokenTTL is how long an issued verification link stays
// redeemable.
const emailVerificationTokenTTL = 24 * time.Hour

// passwordResetTokenTTL is how long an issued password reset link stays
// redeemable — short-lived since it grants a password change.
const passwordResetTokenTTL = 45 * time.Minute

// FacilityResolver lets identity populate a provider's facility claim on
// login without depending on the facilities module's internals — it's
// implemented by facilities.Service and wired in main.go.
type FacilityResolver interface {
	FacilityIDForUser(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error)
}

// PermissionResolver looks up the dynamic Authentication-module access a
// user has — implemented by authz.Service and wired in main.go. A user
// with no assigned role gets full_access=false and an empty permission
// set, which is the correct "no access to the new admin features" default
// for both public self-registrations and not-yet-assigned admin-created
// accounts.
type PermissionResolver interface {
	ResolveAccess(ctx context.Context, userID uuid.UUID) (roleID *uuid.UUID, fullAccess bool, permissions []string, err error)
}

type Service struct {
	repo        *Repository
	tm          *platform.TokenManager
	facilities  FacilityResolver
	permissions PermissionResolver
	// uploadDir is the local-disk root for uploaded files (profile photos
	// today) — see platform.Config.UploadDir. Profile photos are written to
	// {uploadDir}/profile-photos/, see Service.UploadProfilePhoto.
	uploadDir string
}

func NewService(repo *Repository, tm *platform.TokenManager, facilities FacilityResolver, uploadDir string) *Service {
	return &Service{repo: repo, tm: tm, facilities: facilities, uploadDir: uploadDir}
}

// SetPermissionResolver breaks what would otherwise be a circular
// constructor dependency: authz.Service needs identity.Service (to create
// accounts) and identity.Service needs authz.Service (to resolve
// permissions at login). main.go constructs identity first, then authz,
// then wires this in — by the time any request actually logs in, both are
// fully built.
func (s *Service) SetPermissionResolver(permissions PermissionResolver) {
	s.permissions = permissions
}

type RegisterInput struct {
	Email    *string
	Phone    *string
	Password string
}

// Register is the public self-registration path. It always creates a
// plain patient account with no Authentication-module role — granting
// staff-level legacy roles or admin permissions requires an administrator
// to do it explicitly via the Users screen, closing the hole where the
// caller used to be able to pick any role including system_admin.
func (s *Service) Register(ctx context.Context, in RegisterInput) (User, error) {
	if in.Email == nil && in.Phone == nil {
		return User{}, fmt.Errorf("email or phone is required")
	}
	if len(in.Password) < 8 {
		return User{}, fmt.Errorf("password must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}

	return s.repo.CreateUser(ctx, in.Email, in.Phone, string(hash), platform.RolePatient)
}

// CreateAccount is the admin-driven path (Authentication > Users > Create):
// unlike Register, the caller picks the legacy role explicitly. Used by
// authz.Service, which assigns the dynamic permission role afterward.
func (s *Service) CreateAccount(ctx context.Context, email, phone *string, password string, role platform.Role) (User, error) {
	if email == nil && phone == nil {
		return User{}, fmt.Errorf("email or phone is required")
	}
	if len(password) < 8 {
		return User{}, fmt.Errorf("password must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}

	return s.repo.CreateUser(ctx, email, phone, string(hash), role)
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// Access mirrors the dynamic-permission fields embedded in the issued
// tokens, returned alongside them so handlers can put the same
// information in the login/me response body without re-parsing the JWT.
type Access struct {
	RoleID      *uuid.UUID
	FullAccess  bool
	Permissions []string
}

// SessionMetadata is optional context about the client presenting
// credentials, recorded alongside the refresh token it's issued. No client
// sends DeviceLabel yet, but the field exists so a future one can; IP and
// UserAgent are read from the request by the handler.
type SessionMetadata struct {
	DeviceLabel *string
	IP          *string
	UserAgent   *string
}

func (s *Service) Login(ctx context.Context, identifier, password string, meta SessionMetadata) (User, TokenPair, Access, error) {
	user, err := s.repo.FindByEmailOrPhone(ctx, identifier)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return User{}, TokenPair{}, Access{}, ErrInvalidCredentials
		}
		return User{}, TokenPair{}, Access{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return User{}, TokenPair{}, Access{}, ErrInvalidCredentials
	}

	tokens, access, err := s.issueTokens(ctx, user, meta)
	if err != nil {
		return User{}, TokenPair{}, Access{}, err
	}

	s.setPresence(ctx, user.ID, true)

	return user, tokens, access, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	return s.repo.FindByID(ctx, id)
}

// Refresh rotates a presented refresh token: the old row is revoked and a
// new one issued in its place, along with a freshly-resolved access token.
// Rotation (rather than reuse) means a stolen-and-replayed old token stops
// working the moment the legitimate client refreshes, which is the signal
// a later task can use to detect token theft.
func (s *Service) Refresh(ctx context.Context, rawToken string, meta SessionMetadata) (TokenPair, Access, error) {
	record, err := s.repo.FindRefreshTokenByHash(ctx, hashToken(rawToken))
	if err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) {
			return TokenPair{}, Access{}, ErrInvalidRefreshToken
		}
		return TokenPair{}, Access{}, fmt.Errorf("find refresh token: %w", err)
	}

	if !isRefreshTokenUsable(record, time.Now()) {
		return TokenPair{}, Access{}, ErrInvalidRefreshToken
	}

	if err := s.repo.RevokeRefreshToken(ctx, record.ID); err != nil {
		return TokenPair{}, Access{}, fmt.Errorf("revoke refresh token: %w", err)
	}

	user, err := s.repo.FindByID(ctx, record.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return TokenPair{}, Access{}, ErrInvalidRefreshToken
		}
		return TokenPair{}, Access{}, fmt.Errorf("load user: %w", err)
	}

	tokens, access, err := s.issueTokens(ctx, user, meta)
	if err != nil {
		return TokenPair{}, Access{}, err
	}

	return tokens, access, nil
}

// Logout revokes the presented refresh token if it exists and isn't
// already revoked. It deliberately never reports whether the token was
// found — the handler always returns success — so a caller can't use
// logout to probe for valid tokens.
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	record, err := s.repo.FindRefreshTokenByHash(ctx, hashToken(rawToken))
	if err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) {
			return nil
		}
		return fmt.Errorf("find refresh token: %w", err)
	}

	if record.RevokedAt != nil {
		return nil
	}

	if err := s.repo.RevokeRefreshToken(ctx, record.ID); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	s.setPresence(ctx, record.UserID, false)

	return nil
}

// setPresence is the shared hook Login/Logout use to flip is_online and
// stamp last_seen_at on the caller's profile row (creating it if it doesn't
// exist yet — see Repository.SetProfilePresence). Presence is best-effort:
// a failure here is logged, not propagated, so a profile-table hiccup can
// never block a login or logout.
//
// Known tradeoff (documented per the design doc): this only observes
// explicit login/logout. A browser closed without hitting POST
// /auth/logout leaves is_online stuck true until the user's next login —
// acceptable for this MVP phase; a heartbeat endpoint can supersede this if
// that staleness becomes a real problem. It's also per-user, not
// per-session: logging out of one device while other sessions remain
// active still flips is_online to false for the account as a whole, since
// user_profiles has no concept of "still logged in somewhere else" today.
func (s *Service) setPresence(ctx context.Context, userID uuid.UUID, online bool) {
	if err := s.repo.SetProfilePresence(ctx, userID, online, time.Now()); err != nil {
		slog.Warn("failed to update profile presence", "user_id", userID, "online", online, "error", err)
	}
}

// ListSessions returns the caller's currently-active sessions (not revoked,
// not expired), newest first.
func (s *Service) ListSessions(ctx context.Context, userID uuid.UUID) ([]RefreshToken, error) {
	return s.repo.ListActiveRefreshTokensForUser(ctx, userID)
}

// RevokeSession revokes a specific session belonging to userID. Returns
// ErrRefreshTokenNotFound if the session doesn't exist, doesn't belong to
// userID, or is already revoked — those cases are deliberately
// indistinguishable to the caller.
func (s *Service) RevokeSession(ctx context.Context, id, userID uuid.UUID) error {
	return s.repo.RevokeRefreshTokenForUser(ctx, id, userID)
}

// SendVerificationEmail issues a fresh email verification token for userID
// and logs the raw token at Info level in place of actually emailing it — no
// email provider is wired up yet. Returns ErrAlreadyVerified as a no-op
// signal if the account is already verified.
func (s *Service) SendVerificationEmail(ctx context.Context, userID uuid.UUID) error {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.VerifiedAt != nil {
		return ErrAlreadyVerified
	}

	rawToken, err := generateOpaqueToken()
	if err != nil {
		return fmt.Errorf("generate verification token: %w", err)
	}

	if _, err := s.repo.CreateEmailVerificationToken(ctx, user.ID, hashToken(rawToken), time.Now().Add(emailVerificationTokenTTL)); err != nil {
		return fmt.Errorf("store verification token: %w", err)
	}

	// Stands in for sending the token by email until a provider is chosen —
	// logged at Info so the flow is testable end-to-end without one.
	slog.Info("email verification token issued", "user_id", user.ID, "token", rawToken)

	return nil
}

// VerifyEmail redeems a presented verification token: marks it used and
// marks the owning user verified. Both mutations happen only if the token
// is found, unused, and unexpired.
func (s *Service) VerifyEmail(ctx context.Context, rawToken string) error {
	record, err := s.repo.FindEmailVerificationTokenByHash(ctx, hashToken(rawToken))
	if err != nil {
		if errors.Is(err, ErrEmailVerificationTokenNotFound) {
			return ErrInvalidVerificationToken
		}
		return fmt.Errorf("find verification token: %w", err)
	}

	if !isEmailVerificationTokenUsable(record, time.Now()) {
		return ErrInvalidVerificationToken
	}

	if err := s.repo.MarkEmailVerificationTokenUsed(ctx, record.ID); err != nil {
		return fmt.Errorf("mark verification token used: %w", err)
	}

	if err := s.repo.MarkUserVerified(ctx, record.UserID); err != nil {
		return fmt.Errorf("mark user verified: %w", err)
	}

	return nil
}

// isEmailVerificationTokenUsable mirrors isRefreshTokenUsable: not already
// used, and not past its expiry.
func isEmailVerificationTokenUsable(record EmailVerificationToken, now time.Time) bool {
	if record.UsedAt != nil {
		return false
	}
	return now.Before(record.ExpiresAt)
}

// ForgotPassword looks up identifier (email or phone, same lookup Login
// uses) and, if it matches a real user, issues a password reset token and
// logs it server-side (standing in for an email send). It deliberately
// returns nil whether or not a user was found — the handler always returns
// the same success response — so this endpoint can't be used to probe
// whether an identifier has an account.
func (s *Service) ForgotPassword(ctx context.Context, identifier string) error {
	user, err := s.repo.FindByEmailOrPhone(ctx, identifier)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil
		}
		return fmt.Errorf("find user: %w", err)
	}

	rawToken, err := generateOpaqueToken()
	if err != nil {
		return fmt.Errorf("generate reset token: %w", err)
	}

	if _, err := s.repo.CreatePasswordResetToken(ctx, user.ID, hashToken(rawToken), time.Now().Add(passwordResetTokenTTL)); err != nil {
		return fmt.Errorf("store reset token: %w", err)
	}

	// Stands in for sending the token by email until a provider is chosen —
	// logged at Info so the flow is testable end-to-end without one.
	slog.Info("password reset token issued", "user_id", user.ID, "token", rawToken)

	return nil
}

// ResetPassword redeems a presented password reset token: validates
// newPassword against the same minimum-length rule Register enforces,
// updates the owning user's password hash, marks the token used, and
// revokes every one of that user's active refresh tokens so a stolen
// session doesn't survive the password change.
func (s *Service) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	record, err := s.repo.FindPasswordResetTokenByHash(ctx, hashToken(rawToken))
	if err != nil {
		if errors.Is(err, ErrPasswordResetTokenNotFound) {
			return ErrInvalidResetToken
		}
		return fmt.Errorf("find reset token: %w", err)
	}

	if !isPasswordResetTokenUsable(record, time.Now()) {
		return ErrInvalidResetToken
	}

	if len(newPassword) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := s.repo.UpdateUserPasswordHash(ctx, record.UserID, string(hash)); err != nil {
		return fmt.Errorf("update password hash: %w", err)
	}

	if err := s.repo.MarkPasswordResetTokenUsed(ctx, record.ID); err != nil {
		return fmt.Errorf("mark reset token used: %w", err)
	}

	if err := s.repo.RevokeAllRefreshTokensForUser(ctx, record.UserID); err != nil {
		return fmt.Errorf("revoke sessions: %w", err)
	}

	return nil
}

// isPasswordResetTokenUsable mirrors isEmailVerificationTokenUsable: not
// already used, and not past its expiry.
func isPasswordResetTokenUsable(record PasswordResetToken, now time.Time) bool {
	if record.UsedAt != nil {
		return false
	}
	return now.Before(record.ExpiresAt)
}

func (s *Service) issueTokens(ctx context.Context, user User, meta SessionMetadata) (TokenPair, Access, error) {
	facilityID, err := s.facilities.FacilityIDForUser(ctx, user.ID)
	if err != nil {
		return TokenPair{}, Access{}, fmt.Errorf("resolve facility: %w", err)
	}

	var roleID *uuid.UUID
	var fullAccess bool
	var permissions []string
	if s.permissions != nil {
		roleID, fullAccess, permissions, err = s.permissions.ResolveAccess(ctx, user.ID)
	}
	if err != nil {
		return TokenPair{}, Access{}, fmt.Errorf("resolve access: %w", err)
	}

	access := Access{RoleID: roleID, FullAccess: fullAccess, Permissions: permissions}

	tc := platform.TokenClaims{
		UserID: user.ID, Role: user.Role, FacilityID: facilityID,
		RoleID: roleID, FullAccess: fullAccess, Permissions: permissions,
	}

	accessToken, err := s.tm.IssueAccessToken(tc)
	if err != nil {
		return TokenPair{}, Access{}, fmt.Errorf("issue access token: %w", err)
	}

	rawRefreshToken, err := generateOpaqueToken()
	if err != nil {
		return TokenPair{}, Access{}, fmt.Errorf("generate refresh token: %w", err)
	}

	_, err = s.repo.CreateRefreshToken(ctx, CreateRefreshTokenParams{
		UserID:      user.ID,
		TokenHash:   hashToken(rawRefreshToken),
		DeviceLabel: meta.DeviceLabel,
		IP:          meta.IP,
		UserAgent:   meta.UserAgent,
		ExpiresAt:   time.Now().Add(s.tm.RefreshTokenTTL()),
	})
	if err != nil {
		return TokenPair{}, Access{}, fmt.Errorf("store refresh token: %w", err)
	}

	return TokenPair{AccessToken: accessToken, RefreshToken: rawRefreshToken}, access, nil
}

// generateOpaqueToken produces a 32-byte cryptographically random value,
// base64url-encoded without padding — this, not a JWT, is what clients now
// hold as their refresh token. Only its SHA-256 hash is ever persisted.
func generateOpaqueToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// isRefreshTokenUsable is the gate a presented refresh token must pass
// before it can be rotated: not already revoked, and not past its
// expiry. Pulled out as a pure function (rather than inlined in Refresh)
// so the two conditions that must independently reject reuse of a stale or
// stolen token can be unit-tested without a database.
func isRefreshTokenUsable(record RefreshToken, now time.Time) bool {
	if record.RevokedAt != nil {
		return false
	}
	return now.Before(record.ExpiresAt)
}
