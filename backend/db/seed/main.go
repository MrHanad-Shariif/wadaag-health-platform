// Command seed populates local dev Postgres with synthetic accounts, two
// facilities, and provider affiliations, so the referral+consent+audit
// flow is exercisable immediately after `make up`. Run with `make seed`
// (or `go run ./db/seed`). Never point this at anything but a local/dev
// database — it's meant to be re-run freely and skips rows that already
// exist rather than erroring.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wadaag/health-platform/backend/internal/facilities"
	"github.com/wadaag/health-platform/backend/internal/identity"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"golang.org/x/crypto/bcrypt"
)

// All seed accounts share this password for convenience in local dev only.
const devPassword = "wadaag-dev-2026"

type seedUser struct {
	email    string
	password string
	role     platform.Role
	facility string // key into seedFacilities, or "" for none
}

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

func seedUsers() []seedUser {
	return []seedUser{
		{email: "physician@wadaag.dev", password: devPassword, role: platform.RolePhysician, facility: "hodan_hospital"},
		{email: "specialist@wadaag.dev", password: devPassword, role: platform.RolePhysician, facility: "banadir_specialist"},
		{email: "hospital-admin@wadaag.dev", password: devPassword, role: platform.RoleHospitalAdmin, facility: "hodan_hospital"},
		{email: "lab-tech@wadaag.dev", password: devPassword, role: platform.RoleLabTech},
		{email: "pharmacist@wadaag.dev", password: devPassword, role: platform.RolePharmacist},
		{email: "insurer@wadaag.dev", password: devPassword, role: platform.RoleInsurer},
		{email: "admin@wadaag.dev", password: devPassword, role: platform.RoleSystemAdmin},
		{email: "patient@wadaag.dev", password: devPassword, role: platform.RolePatient},
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

	fmt.Println("seed complete — all accounts use password:", devPassword)
}

func run(ctx context.Context, db *pgxpool.Pool) error {
	identityRepo := identity.NewRepository(db)
	facilitiesRepo := facilities.NewRepository(db)

	facilityIDs, err := seedFacilityRows(ctx, facilitiesRepo)
	if err != nil {
		return err
	}

	for _, su := range seedUsers() {
		email := su.email
		hash, err := bcrypt.GenerateFromPassword([]byte(su.password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash password for %s: %w", email, err)
		}

		user, err := identityRepo.CreateUser(ctx, &email, nil, string(hash), su.role)
		if err != nil {
			if !errors.Is(err, identity.ErrDuplicateUser) {
				return fmt.Errorf("create user %s: %w", email, err)
			}
			fmt.Fprintf(os.Stderr, "skip %s: already seeded\n", email)
			user, err = identityRepo.FindByEmailOrPhone(ctx, email)
			if err != nil {
				return fmt.Errorf("look up existing user %s: %w", email, err)
			}
		} else {
			fmt.Printf("seeded %s (%s)\n", email, su.role)
		}

		if su.facility == "" {
			continue
		}
		facilityID, ok := facilityIDs[su.facility]
		if !ok {
			return fmt.Errorf("unknown facility key %q for %s", su.facility, email)
		}
		if _, err := facilitiesRepo.CreateProvider(ctx, user.ID, facilityID, nil, nil); err != nil {
			if errors.Is(err, facilities.ErrDuplicateProvider) {
				continue
			}
			return fmt.Errorf("affiliate %s with facility: %w", email, err)
		}
		fmt.Printf("  affiliated with %s\n", su.facility)
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
