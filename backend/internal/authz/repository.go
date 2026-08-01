package authz

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"github.com/wadaag/health-platform/backend/internal/platform/sqlcgen"
)

var ErrRoleNotFound = errors.New("role not found")
var ErrDuplicateRole = errors.New("a role with this name already exists")
var ErrRoleInUse = errors.New("role is assigned to at least one user and cannot be deleted")
var ErrUserNotFound = errors.New("user not found")

type Repository struct {
	q *sqlcgen.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{q: sqlcgen.New(db)}
}

func (r *Repository) CreateRole(ctx context.Context, name string, description *string, fullAccess bool) (Role, error) {
	row, err := r.q.CreateRole(ctx, sqlcgen.CreateRoleParams{
		Name: name, Description: platform.PgText(description), FullAccess: fullAccess,
	})
	if err != nil {
		if platform.IsUniqueViolation(err) {
			return Role{}, ErrDuplicateRole
		}
		return Role{}, fmt.Errorf("insert role: %w", err)
	}
	return roleFromRow(row), nil
}

func (r *Repository) FindRoleByID(ctx context.Context, id uuid.UUID) (Role, error) {
	row, err := r.q.FindRoleByID(ctx, platform.PgUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Role{}, ErrRoleNotFound
	}
	if err != nil {
		return Role{}, fmt.Errorf("query role: %w", err)
	}
	return roleFromRow(row), nil
}

func (r *Repository) ListRoles(ctx context.Context) ([]Role, error) {
	rows, err := r.q.ListRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	roles := make([]Role, len(rows))
	for i, row := range rows {
		roles[i] = roleFromRow(row)
	}
	return roles, nil
}

func (r *Repository) UpdateRole(ctx context.Context, id uuid.UUID, name string, description *string, fullAccess bool) (Role, error) {
	row, err := r.q.UpdateRole(ctx, sqlcgen.UpdateRoleParams{
		ID: platform.PgUUID(id), Name: name, Description: platform.PgText(description), FullAccess: fullAccess,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Role{}, ErrRoleNotFound
	}
	if err != nil {
		if platform.IsUniqueViolation(err) {
			return Role{}, ErrDuplicateRole
		}
		return Role{}, fmt.Errorf("update role: %w", err)
	}
	return roleFromRow(row), nil
}

func (r *Repository) DeleteRole(ctx context.Context, id uuid.UUID) error {
	err := r.q.DeleteRole(ctx, platform.PgUUID(id))
	if err != nil {
		if isForeignKeyViolation(err) {
			return ErrRoleInUse
		}
		return fmt.Errorf("delete role: %w", err)
	}
	return nil
}

func (r *Repository) ListPermissions(ctx context.Context) ([]Permission, error) {
	rows, err := r.q.ListPermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}
	permissions := make([]Permission, len(rows))
	for i, row := range rows {
		permissions[i] = Permission{ID: platform.FromPgUUID(row.ID), Resource: row.Resource, Action: row.Action}
	}
	return permissions, nil
}

func (r *Repository) ListPermissionsForRole(ctx context.Context, roleID uuid.UUID) ([]Permission, error) {
	rows, err := r.q.ListPermissionsForRole(ctx, platform.PgUUID(roleID))
	if err != nil {
		return nil, fmt.Errorf("list role permissions: %w", err)
	}
	permissions := make([]Permission, len(rows))
	for i, row := range rows {
		permissions[i] = Permission{ID: platform.FromPgUUID(row.ID), Resource: row.Resource, Action: row.Action}
	}
	return permissions, nil
}

// ReplaceRolePermissions is called within a transaction-free two-step
// (delete then bulk insert) — acceptable here since a role's permission
// set is only ever edited by one admin action at a time, not a hot path
// needing atomicity guarantees beyond "the last write wins".
func (r *Repository) ReplaceRolePermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	if err := r.q.ReplaceRolePermissions(ctx, platform.PgUUID(roleID)); err != nil {
		return fmt.Errorf("clear role permissions: %w", err)
	}
	if len(permissionIDs) == 0 {
		return nil
	}

	pgIDs := make([]pgtype.UUID, len(permissionIDs))
	for i, id := range permissionIDs {
		pgIDs[i] = platform.PgUUID(id)
	}
	if err := r.q.InsertRolePermissions(ctx, sqlcgen.InsertRolePermissionsParams{
		RoleID: platform.PgUUID(roleID), Column2: pgIDs,
	}); err != nil {
		return fmt.Errorf("insert role permissions: %w", err)
	}
	return nil
}

func (r *Repository) GetUserAccess(ctx context.Context, userID uuid.UUID) (roleID *uuid.UUID, fullAccess bool, err error) {
	row, err := r.q.GetUserAccess(ctx, platform.PgUUID(userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("query user access: %w", err)
	}
	id := platform.FromPgUUID(row.RoleID)
	return &id, row.FullAccess, nil
}

func (r *Repository) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := r.q.ListUsersWithRole(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	users := make([]User, len(rows))
	for i, row := range rows {
		users[i] = User{
			ID: platform.FromPgUUID(row.ID), Email: platform.FromPgText(row.Email), Phone: platform.FromPgText(row.Phone),
			LegacyRole: platform.Role(row.Role), Status: row.Status, CreatedAt: row.CreatedAt.Time,
			RoleID: platform.FromPgUUIDPtr(row.RoleID), RoleName: platform.FromPgText(row.RoleName),
		}
	}
	return users, nil
}

func (r *Repository) FindUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	row, err := r.q.FindUserWithRoleByID(ctx, platform.PgUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("query user: %w", err)
	}
	return User{
		ID: platform.FromPgUUID(row.ID), Email: platform.FromPgText(row.Email), Phone: platform.FromPgText(row.Phone),
		LegacyRole: platform.Role(row.Role), Status: row.Status, CreatedAt: row.CreatedAt.Time,
		RoleID: platform.FromPgUUIDPtr(row.RoleID), RoleName: platform.FromPgText(row.RoleName),
	}, nil
}

func (r *Repository) SetUserRole(ctx context.Context, userID uuid.UUID, roleID *uuid.UUID) error {
	_, err := r.q.UpdateUserRoleID(ctx, sqlcgen.UpdateUserRoleIDParams{
		ID: platform.PgUUID(userID), RoleID: platform.PgUUIDPtr(roleID),
	})
	if err != nil {
		return fmt.Errorf("update user role: %w", err)
	}
	return nil
}

func (r *Repository) SetUserStatus(ctx context.Context, userID uuid.UUID, status string) error {
	_, err := r.q.UpdateUserStatus(ctx, sqlcgen.UpdateUserStatusParams{ID: platform.PgUUID(userID), Status: status})
	if err != nil {
		return fmt.Errorf("update user status: %w", err)
	}
	return nil
}

func (r *Repository) CountUsers(ctx context.Context) (int64, error) {
	count, err := r.q.CountUsers(ctx)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

func roleFromRow(row sqlcgen.Role) Role {
	return Role{
		ID: platform.FromPgUUID(row.ID), Name: row.Name, Description: platform.FromPgText(row.Description),
		FullAccess: row.FullAccess, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
	}
}

func isForeignKeyViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23503"
	}
	return false
}
