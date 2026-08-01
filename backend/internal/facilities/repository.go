package facilities

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

var ErrFacilityNotFound = errors.New("facility not found")
var ErrProviderNotFound = errors.New("provider not found")
var ErrDuplicateProvider = errors.New("user is already a provider")

type Repository struct {
	q *sqlcgen.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{q: sqlcgen.New(db)}
}

func (r *Repository) CreateFacility(ctx context.Context, name string, facilityType Type, region, district, phone, address *string) (Facility, error) {
	row, err := r.q.CreateFacility(ctx, sqlcgen.CreateFacilityParams{
		Name:     name,
		Type:     sqlcgen.FacilityType(facilityType),
		Region:   platform.PgText(region),
		District: platform.PgText(district),
		Phone:    platform.PgText(phone),
		Address:  platform.PgText(address),
	})
	if err != nil {
		return Facility{}, fmt.Errorf("insert facility: %w", err)
	}

	return facilityFromRow(row), nil
}

func (r *Repository) ListFacilities(ctx context.Context) ([]Facility, error) {
	rows, err := r.q.ListFacilities(ctx)
	if err != nil {
		return nil, fmt.Errorf("list facilities: %w", err)
	}

	facilities := make([]Facility, len(rows))
	for i, row := range rows {
		facilities[i] = facilityFromRow(row)
	}
	return facilities, nil
}

func (r *Repository) FindFacilityByID(ctx context.Context, id uuid.UUID) (Facility, error) {
	row, err := r.q.FindFacilityByID(ctx, platform.PgUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Facility{}, ErrFacilityNotFound
	}
	if err != nil {
		return Facility{}, fmt.Errorf("query facility: %w", err)
	}

	return facilityFromRow(row), nil
}

func (r *Repository) CreateProvider(ctx context.Context, userID, facilityID uuid.UUID, specialty, licenseNumber *string) (Provider, error) {
	row, err := r.q.CreateProvider(ctx, sqlcgen.CreateProviderParams{
		UserID:        platform.PgUUID(userID),
		FacilityID:    platform.PgUUID(facilityID),
		Specialty:     platform.PgText(specialty),
		LicenseNumber: platform.PgText(licenseNumber),
	})
	if err != nil {
		if platform.IsUniqueViolation(err) {
			return Provider{}, ErrDuplicateProvider
		}
		return Provider{}, fmt.Errorf("insert provider: %w", err)
	}

	return providerFromRow(row), nil
}

func (r *Repository) FindProviderByUserID(ctx context.Context, userID uuid.UUID) (Provider, error) {
	row, err := r.q.FindProviderByUserID(ctx, platform.PgUUID(userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Provider{}, ErrProviderNotFound
	}
	if err != nil {
		return Provider{}, fmt.Errorf("query provider: %w", err)
	}

	return providerFromRow(row), nil
}

func facilityFromRow(row sqlcgen.Facility) Facility {
	return Facility{
		ID:        platform.FromPgUUID(row.ID),
		Name:      row.Name,
		Type:      Type(row.Type),
		Region:    platform.FromPgText(row.Region),
		District:  platform.FromPgText(row.District),
		Phone:     platform.FromPgText(row.Phone),
		Address:   platform.FromPgText(row.Address),
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}

func providerFromRow(row sqlcgen.Provider) Provider {
	return Provider{
		ID:            platform.FromPgUUID(row.ID),
		UserID:        platform.FromPgUUID(row.UserID),
		FacilityID:    platform.FromPgUUID(row.FacilityID),
		Specialty:     platform.FromPgText(row.Specialty),
		LicenseNumber: platform.FromPgText(row.LicenseNumber),
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}
}
