// Command seed populates local dev Postgres with the two demo facilities
// and exactly one account: the system administrator. Every other user and
// role is created from the webapp's Authentication module by the admin,
// not hardcoded here. Run with `make seed` (or `go run ./db/seed`). Never
// point this at anything but a local/dev database — it's meant to be
// re-run freely and skips rows that already exist rather than erroring.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wadaag/health-platform/backend/internal/authz"
	"github.com/wadaag/health-platform/backend/internal/facilities"
	"github.com/wadaag/health-platform/backend/internal/identity"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"golang.org/x/crypto/bcrypt"
)

const (
	adminEmail    = "administrator@wadaaghealthy.com"
	adminPassword = "wadaaghealthy.com@2026"
	adminRoleName = "Administrator"
)

func seedFacilities() map[string]struct {
	Name string
	Type facilities.Type
} {
	return map[string]struct {
		Name string
		Type facilities.Type
	}{
		"hodan_hospital":     {Name: "Hodan District Hospital", Type: facilities.TypeHospital},
		"banadir_specialist": {Name: "Banadir Specialist Clinic", Type: facilities.TypeClinic},
	}
}

func main() {
	ctx := context.Background()

	cfg, err := platform.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := platform.NewDBPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer db.Close()

	if err := run(ctx, db); err != nil {
		log.Fatalf("seed failed: %v", err)
	}

	fmt.Printf("seed complete — sign in as %s / %s\n", adminEmail, adminPassword)
}

func run(ctx context.Context, db *pgxpool.Pool) error {
	facilitiesRepo := facilities.NewRepository(db)
	if _, err := seedFacilityRows(ctx, facilitiesRepo); err != nil {
		return err
	}

	authzRepo := authz.NewRepository(db)
	adminRoleID, err := seedAdministratorRole(ctx, authzRepo)
	if err != nil {
		return err
	}

	identityRepo := identity.NewRepository(db)
	if err := seedAdministratorUser(ctx, identityRepo, authzRepo, adminRoleID); err != nil {
		return err
	}

	return nil
}

// seedAdministratorRole creates (or finds) the full-access role and makes
// sure it holds every permission in the catalog — re-running the seed
// after a new resource/action is added to the catalog keeps it complete.
func seedAdministratorRole(ctx context.Context, repo *authz.Repository) (uuid.UUID, error) {
	roles, err := repo.ListRoles(ctx)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("list existing roles: %w", err)
	}

	var roleID uuid.UUID
	found := false
	for _, r := range roles {
		if r.Name == adminRoleName {
			roleID = r.ID
			found = true
			break
		}
	}

	if !found {
		description := "Full access to every module, including Authentication."
		role, err := repo.CreateRole(ctx, adminRoleName, &description, true)
		if err != nil {
			return uuid.UUID{}, fmt.Errorf("create administrator role: %w", err)
		}
		roleID = role.ID
		fmt.Printf("seeded role %s (full access)\n", adminRoleName)
	} else {
		fmt.Fprintf(os.Stderr, "skip role %s: already seeded\n", adminRoleName)
	}

	catalog, err := repo.ListPermissions(ctx)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("list permission catalog: %w", err)
	}
	permissionIDs := make([]uuid.UUID, len(catalog))
	for i, p := range catalog {
		permissionIDs[i] = p.ID
	}
	if err := repo.ReplaceRolePermissions(ctx, roleID, permissionIDs); err != nil {
		return uuid.UUID{}, fmt.Errorf("grant all permissions to %s: %w", adminRoleName, err)
	}

	return roleID, nil
}

func seedAdministratorUser(ctx context.Context, identityRepo *identity.Repository, authzRepo *authz.Repository, roleID uuid.UUID) error {
	email := adminEmail

	hash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash administrator password: %w", err)
	}

	user, err := identityRepo.CreateUser(ctx, &email, nil, string(hash), platform.RoleSystemAdmin)
	if err != nil {
		if !errors.Is(err, identity.ErrDuplicateUser) {
			return fmt.Errorf("create administrator user: %w", err)
		}
		fmt.Fprintf(os.Stderr, "skip user %s: already seeded\n", email)
		user, err = identityRepo.FindByEmailOrPhone(ctx, email)
		if err != nil {
			return fmt.Errorf("look up existing administrator: %w", err)
		}
	} else {
		fmt.Printf("seeded user %s\n", email)
	}

	if err := authzRepo.SetUserRole(ctx, user.ID, &roleID); err != nil {
		return fmt.Errorf("assign %s role to administrator: %w", adminRoleName, err)
	}

	return nil
}

func seedFacilityRows(ctx context.Context, repo *facilities.Repository) (map[string]uuid.UUID, error) {
	existing, err := repo.ListFacilities(ctx)
	if err != nil {
		return nil, fmt.Errorf("list existing facilities: %w", err)
	}
	byName := make(map[string]uuid.UUID, len(existing))
	for _, f := range existing {
		byName[f.Name] = f.ID
	}

	ids := make(map[string]uuid.UUID)
	for key, f := range seedFacilities() {
		if id, ok := byName[f.Name]; ok {
			ids[key] = id
			fmt.Fprintf(os.Stderr, "skip facility %s: already seeded\n", f.Name)
			continue
		}

		created, err := repo.CreateFacility(ctx, f.Name, f.Type, nil, nil, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("create facility %s: %w", f.Name, err)
		}
		ids[key] = created.ID
		fmt.Printf("seeded facility %s (%s)\n", f.Name, key)
	}
	return ids, nil
}
