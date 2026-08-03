package authz

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/facilities"
	"github.com/wadaag/health-platform/backend/internal/identity"
	"github.com/wadaag/health-platform/backend/internal/platform"
)

// AccountCreator is the narrow slice of identity.Service the admin-driven
// "create user" flow needs — creating the login credentials themselves,
// distinct from public self-registration.
type AccountCreator interface {
	CreateAccount(ctx context.Context, email, phone *string, password string, role platform.Role, fullName *string) (identity.User, error)
}

// ProviderAffiliator lets the admin optionally attach the new user to a
// facility in the same step, reusing facilities.Service rather than
// duplicating its logic.
type ProviderAffiliator interface {
	CreateProvider(ctx context.Context, in facilities.CreateProviderInput) (facilities.Provider, error)
}

type Service struct {
	repo       *Repository
	accounts   AccountCreator
	facilities ProviderAffiliator
}

func NewService(repo *Repository, accounts AccountCreator, facilities ProviderAffiliator) *Service {
	return &Service{repo: repo, accounts: accounts, facilities: facilities}
}

type RoleInput struct {
	Name          string
	Description   *string
	FullAccess    bool
	PermissionIDs []uuid.UUID
}

func (s *Service) CreateRole(ctx context.Context, in RoleInput) (RoleWithPermissions, error) {
	role, err := s.repo.CreateRole(ctx, in.Name, in.Description, in.FullAccess)
	if err != nil {
		return RoleWithPermissions{}, err
	}
	if err := s.repo.ReplaceRolePermissions(ctx, role.ID, in.PermissionIDs); err != nil {
		return RoleWithPermissions{}, err
	}
	return s.GetRole(ctx, role.ID)
}

func (s *Service) UpdateRole(ctx context.Context, id uuid.UUID, in RoleInput) (RoleWithPermissions, error) {
	if _, err := s.repo.UpdateRole(ctx, id, in.Name, in.Description, in.FullAccess); err != nil {
		return RoleWithPermissions{}, err
	}
	if err := s.repo.ReplaceRolePermissions(ctx, id, in.PermissionIDs); err != nil {
		return RoleWithPermissions{}, err
	}
	return s.GetRole(ctx, id)
}

func (s *Service) DeleteRole(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteRole(ctx, id)
}

func (s *Service) GetRole(ctx context.Context, id uuid.UUID) (RoleWithPermissions, error) {
	role, err := s.repo.FindRoleByID(ctx, id)
	if err != nil {
		return RoleWithPermissions{}, err
	}
	permissions, err := s.repo.ListPermissionsForRole(ctx, id)
	if err != nil {
		return RoleWithPermissions{}, err
	}
	return RoleWithPermissions{Role: role, Permissions: permissions}, nil
}

func (s *Service) ListRoles(ctx context.Context) ([]Role, error) {
	return s.repo.ListRoles(ctx)
}

func (s *Service) ListPermissionCatalog(ctx context.Context) ([]Permission, error) {
	return s.repo.ListPermissions(ctx)
}

// ResolveAccess implements identity.PermissionResolver.
func (s *Service) ResolveAccess(ctx context.Context, userID uuid.UUID) (*uuid.UUID, bool, []string, error) {
	roleID, fullAccess, err := s.repo.GetUserAccess(ctx, userID)
	if err != nil {
		return nil, false, nil, err
	}
	if roleID == nil {
		return nil, false, nil, nil
	}

	permissions, err := s.repo.ListPermissionsForRole(ctx, *roleID)
	if err != nil {
		return nil, false, nil, err
	}

	codes := make([]string, len(permissions))
	for i, p := range permissions {
		codes[i] = p.Code()
	}
	return roleID, fullAccess, codes, nil
}

type CreateUserInput struct {
	Email         *string
	Phone         *string
	Password      string
	LegacyRole    platform.Role
	RoleID        *uuid.UUID
	FacilityID    *uuid.UUID
	Specialty     *string
	LicenseNumber *string
	FullName      *string
}

// CreateUser is the admin-driven account creation flow: makes the login
// credentials via identity, optionally links a facility via a providers
// row, and optionally assigns a dynamic permission role — all three are
// independent, so any combination the admin picks is valid.
func (s *Service) CreateUser(ctx context.Context, in CreateUserInput) (User, error) {
	account, err := s.accounts.CreateAccount(ctx, in.Email, in.Phone, in.Password, in.LegacyRole, in.FullName)
	if err != nil {
		return User{}, err
	}

	if in.FacilityID != nil {
		if _, err := s.facilities.CreateProvider(ctx, facilities.CreateProviderInput{
			UserID: account.ID, FacilityID: *in.FacilityID, Specialty: in.Specialty, LicenseNumber: in.LicenseNumber,
		}); err != nil {
			return User{}, fmt.Errorf("affiliate with facility: %w", err)
		}
	}

	if in.RoleID != nil {
		if err := s.repo.SetUserRole(ctx, account.ID, in.RoleID); err != nil {
			return User{}, err
		}
	}

	return s.repo.FindUserByID(ctx, account.ID)
}

func (s *Service) ListUsers(ctx context.Context) ([]User, error) {
	return s.repo.ListUsers(ctx)
}

func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (User, error) {
	return s.repo.FindUserByID(ctx, id)
}

func (s *Service) SetUserRole(ctx context.Context, userID uuid.UUID, roleID *uuid.UUID) (User, error) {
	if err := s.repo.SetUserRole(ctx, userID, roleID); err != nil {
		return User{}, err
	}
	return s.repo.FindUserByID(ctx, userID)
}

func (s *Service) SetUserStatus(ctx context.Context, userID uuid.UUID, status string) (User, error) {
	if err := s.repo.SetUserStatus(ctx, userID, status); err != nil {
		return User{}, err
	}
	return s.repo.FindUserByID(ctx, userID)
}

func (s *Service) CountUsers(ctx context.Context) (int64, error) {
	return s.repo.CountUsers(ctx)
}

// ListFeatureFlags returns every feature flag — system_admin only,
// restricting who may call this is the handler's job (route-level
// platform.RequireRoles, same as every other admin-only method in this
// service relies on).
func (s *Service) ListFeatureFlags(ctx context.Context) ([]FeatureFlag, error) {
	return s.repo.ListFeatureFlags(ctx)
}

// SetFeatureFlag creates or updates the flag identified by key. Like
// ListFeatureFlags, restricted to system_admin at the route level.
func (s *Service) SetFeatureFlag(ctx context.Context, key string, enabled bool, description *string) (FeatureFlag, error) {
	return s.repo.UpsertFeatureFlag(ctx, key, enabled, description)
}
