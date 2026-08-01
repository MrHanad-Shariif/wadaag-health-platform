package identity

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid email/phone or password")

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
}

func NewService(repo *Repository, tm *platform.TokenManager, facilities FacilityResolver) *Service {
	return &Service{repo: repo, tm: tm, facilities: facilities}
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

func (s *Service) Login(ctx context.Context, identifier, password string) (User, TokenPair, Access, error) {
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

	tokens, access, err := s.issueTokens(ctx, user)
	if err != nil {
		return User{}, TokenPair{}, Access{}, err
	}

	return user, tokens, access, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) issueTokens(ctx context.Context, user User) (TokenPair, Access, error) {
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

	refreshToken, err := s.tm.IssueRefreshToken(tc)
	if err != nil {
		return TokenPair{}, Access{}, fmt.Errorf("issue refresh token: %w", err)
	}

	return TokenPair{AccessToken: accessToken, RefreshToken: refreshToken}, access, nil
}
