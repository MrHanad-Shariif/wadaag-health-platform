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
var ErrBranchNotFound = errors.New("branch not found")
var ErrDepartmentNotFound = errors.New("department not found")

type Repository struct {
	q *sqlcgen.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{q: sqlcgen.New(db)}
}

func (r *Repository) CountFacilities(ctx context.Context) (int64, error) {
	count, err := r.q.CountFacilities(ctx)
	if err != nil {
		return 0, fmt.Errorf("count facilities: %w", err)
	}
	return count, nil
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

// UpdateFacilityLogoAndHours overwrites logo_url, working_hours, and
// referral_policies with exactly the values given. It does not itself
// implement partial-update (merge) semantics — see Service.UpdateFacility,
// which reads the current facility, overlays only the fields the caller
// supplied, and passes the fully-resolved result here, following the same
// read-then-merge-then-write pattern as identity.Service.UpdateProfile /
// mergeProfileUpdate.
func (r *Repository) UpdateFacilityLogoAndHours(ctx context.Context, id uuid.UUID, logoURL *string, workingHours []byte, referralPolicies *string) (Facility, error) {
	row, err := r.q.UpdateFacilityLogoAndHours(ctx, sqlcgen.UpdateFacilityLogoAndHoursParams{
		ID:               platform.PgUUID(id),
		LogoUrl:          platform.PgText(logoURL),
		WorkingHours:     workingHours,
		ReferralPolicies: platform.PgText(referralPolicies),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Facility{}, ErrFacilityNotFound
	}
	if err != nil {
		return Facility{}, fmt.Errorf("update facility logo/hours: %w", err)
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

func (r *Repository) FindProviderByID(ctx context.Context, id uuid.UUID) (Provider, error) {
	row, err := r.q.FindProviderByID(ctx, platform.PgUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Provider{}, ErrProviderNotFound
	}
	if err != nil {
		return Provider{}, fmt.Errorf("query provider: %w", err)
	}

	return providerFromRow(row), nil
}

// UpdateProviderProfile overwrites the self-editable provider profile
// fields with exactly the values given. Like
// UpdateFacilityLogoAndHours, it does not itself implement partial-update
// (merge) semantics — see Service.UpdateOwnProviderProfile, which reads the
// current provider, overlays only the fields the caller supplied, and
// passes the fully-resolved result here.
func (r *Repository) UpdateProviderProfile(ctx context.Context, id uuid.UUID, qualifications []string, yearsExperience *int, consultationFee *string, languages []string, certificates []byte, areasOfExpertise []string) (Provider, error) {
	fee, err := platform.PgNumeric(consultationFee)
	if err != nil {
		return Provider{}, fmt.Errorf("invalid consultation fee: %w", err)
	}

	row, err := r.q.UpdateProviderProfile(ctx, sqlcgen.UpdateProviderProfileParams{
		ID:               platform.PgUUID(id),
		Qualifications:   qualifications,
		YearsExperience:  platform.PgInt4Ptr(yearsExperience),
		ConsultationFee:  fee,
		Languages:        languages,
		Certificates:     certificates,
		AreasOfExpertise: areasOfExpertise,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Provider{}, ErrProviderNotFound
	}
	if err != nil {
		return Provider{}, fmt.Errorf("update provider profile: %w", err)
	}

	return providerFromRow(row), nil
}

// UpdateProviderVerificationStatus is the admin-only counterpart to
// UpdateProviderProfile — kept as its own repository method (backed by its
// own sqlc query) so the self-edit path can never touch verification_status.
func (r *Repository) UpdateProviderVerificationStatus(ctx context.Context, id uuid.UUID, status string) (Provider, error) {
	row, err := r.q.UpdateProviderVerificationStatus(ctx, sqlcgen.UpdateProviderVerificationStatusParams{
		ID:                 platform.PgUUID(id),
		VerificationStatus: status,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Provider{}, ErrProviderNotFound
	}
	if err != nil {
		return Provider{}, fmt.Errorf("update provider verification status: %w", err)
	}

	return providerFromRow(row), nil
}

func (r *Repository) CreateBranch(ctx context.Context, facilityID uuid.UUID, name string, address, phone *string) (Branch, error) {
	row, err := r.q.CreateBranch(ctx, sqlcgen.CreateBranchParams{
		FacilityID: platform.PgUUID(facilityID),
		Name:       name,
		Address:    platform.PgText(address),
		Phone:      platform.PgText(phone),
	})
	if err != nil {
		return Branch{}, fmt.Errorf("insert branch: %w", err)
	}
	return branchFromRow(row), nil
}

func (r *Repository) ListBranchesByFacility(ctx context.Context, facilityID uuid.UUID) ([]Branch, error) {
	rows, err := r.q.ListBranchesByFacility(ctx, platform.PgUUID(facilityID))
	if err != nil {
		return nil, fmt.Errorf("list branches: %w", err)
	}

	branches := make([]Branch, len(rows))
	for i, row := range rows {
		branches[i] = branchFromRow(row)
	}
	return branches, nil
}

func (r *Repository) FindBranchByID(ctx context.Context, id uuid.UUID) (Branch, error) {
	row, err := r.q.FindBranchByID(ctx, platform.PgUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Branch{}, ErrBranchNotFound
	}
	if err != nil {
		return Branch{}, fmt.Errorf("query branch: %w", err)
	}
	return branchFromRow(row), nil
}

func (r *Repository) UpdateBranch(ctx context.Context, id uuid.UUID, name string, address, phone *string) (Branch, error) {
	row, err := r.q.UpdateBranch(ctx, sqlcgen.UpdateBranchParams{
		ID:      platform.PgUUID(id),
		Name:    name,
		Address: platform.PgText(address),
		Phone:   platform.PgText(phone),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Branch{}, ErrBranchNotFound
	}
	if err != nil {
		return Branch{}, fmt.Errorf("update branch: %w", err)
	}
	return branchFromRow(row), nil
}

func (r *Repository) DeleteBranch(ctx context.Context, id uuid.UUID) error {
	if err := r.q.DeleteBranch(ctx, platform.PgUUID(id)); err != nil {
		return fmt.Errorf("delete branch: %w", err)
	}
	return nil
}

func (r *Repository) CreateDepartment(ctx context.Context, facilityID uuid.UUID, name string) (Department, error) {
	row, err := r.q.CreateDepartment(ctx, sqlcgen.CreateDepartmentParams{
		FacilityID: platform.PgUUID(facilityID),
		Name:       name,
	})
	if err != nil {
		return Department{}, fmt.Errorf("insert department: %w", err)
	}
	return departmentFromRow(row), nil
}

func (r *Repository) ListDepartmentsByFacility(ctx context.Context, facilityID uuid.UUID) ([]Department, error) {
	rows, err := r.q.ListDepartmentsByFacility(ctx, platform.PgUUID(facilityID))
	if err != nil {
		return nil, fmt.Errorf("list departments: %w", err)
	}

	departments := make([]Department, len(rows))
	for i, row := range rows {
		departments[i] = departmentFromRow(row)
	}
	return departments, nil
}

func (r *Repository) FindDepartmentByID(ctx context.Context, id uuid.UUID) (Department, error) {
	row, err := r.q.FindDepartmentByID(ctx, platform.PgUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Department{}, ErrDepartmentNotFound
	}
	if err != nil {
		return Department{}, fmt.Errorf("query department: %w", err)
	}
	return departmentFromRow(row), nil
}

func (r *Repository) UpdateDepartment(ctx context.Context, id uuid.UUID, name string) (Department, error) {
	row, err := r.q.UpdateDepartment(ctx, sqlcgen.UpdateDepartmentParams{
		ID:   platform.PgUUID(id),
		Name: name,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Department{}, ErrDepartmentNotFound
	}
	if err != nil {
		return Department{}, fmt.Errorf("update department: %w", err)
	}
	return departmentFromRow(row), nil
}

func (r *Repository) DeleteDepartment(ctx context.Context, id uuid.UUID) error {
	if err := r.q.DeleteDepartment(ctx, platform.PgUUID(id)); err != nil {
		return fmt.Errorf("delete department: %w", err)
	}
	return nil
}

// SearchFacilities is a directory-style search by facility name — open to
// any authenticated user, no consent scoping needed (see
// search.Service.Search).
func (r *Repository) SearchFacilities(ctx context.Context, query string, limit int) ([]Facility, error) {
	rows, err := r.q.SearchFacilities(ctx, sqlcgen.SearchFacilitiesParams{
		Query: query, ResultLimit: int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("search facilities: %w", err)
	}
	out := make([]Facility, len(rows))
	for i, row := range rows {
		out[i] = facilityFromRow(row)
	}
	return out, nil
}

// SearchProviders is a directory-style search by the provider's linked
// user's full_name or specialty — open to any authenticated user, same
// reasoning as SearchFacilities.
func (r *Repository) SearchProviders(ctx context.Context, query string, limit int) ([]ProviderSearchResult, error) {
	rows, err := r.q.SearchProviders(ctx, sqlcgen.SearchProvidersParams{
		Query: query, ResultLimit: int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("search providers: %w", err)
	}
	out := make([]ProviderSearchResult, len(rows))
	for i, row := range rows {
		out[i] = ProviderSearchResult{
			ProviderID:   platform.FromPgUUID(row.ID),
			FacilityID:   platform.FromPgUUID(row.FacilityID),
			UserFullName: platform.FromPgText(row.UserFullName),
			Specialty:    platform.FromPgText(row.Specialty),
			CreatedAt:    row.CreatedAt.Time,
		}
	}
	return out, nil
}

func facilityFromRow(row sqlcgen.Facility) Facility {
	return Facility{
		ID:               platform.FromPgUUID(row.ID),
		Name:             row.Name,
		Type:             Type(row.Type),
		Region:           platform.FromPgText(row.Region),
		District:         platform.FromPgText(row.District),
		Phone:            platform.FromPgText(row.Phone),
		Address:          platform.FromPgText(row.Address),
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
		LogoURL:          platform.FromPgText(row.LogoUrl),
		WorkingHours:     row.WorkingHours,
		ReferralPolicies: platform.FromPgText(row.ReferralPolicies),
	}
}

func branchFromRow(row sqlcgen.Branch) Branch {
	return Branch{
		ID:         platform.FromPgUUID(row.ID),
		FacilityID: platform.FromPgUUID(row.FacilityID),
		Name:       row.Name,
		Address:    platform.FromPgText(row.Address),
		Phone:      platform.FromPgText(row.Phone),
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}
}

func departmentFromRow(row sqlcgen.Department) Department {
	return Department{
		ID:         platform.FromPgUUID(row.ID),
		FacilityID: platform.FromPgUUID(row.FacilityID),
		Name:       row.Name,
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}
}

func providerFromRow(row sqlcgen.Provider) Provider {
	return Provider{
		ID:                 platform.FromPgUUID(row.ID),
		UserID:             platform.FromPgUUID(row.UserID),
		FacilityID:         platform.FromPgUUID(row.FacilityID),
		Specialty:          platform.FromPgText(row.Specialty),
		LicenseNumber:      platform.FromPgText(row.LicenseNumber),
		CreatedAt:          row.CreatedAt.Time,
		UpdatedAt:          row.UpdatedAt.Time,
		Qualifications:     row.Qualifications,
		YearsExperience:    platform.FromPgInt4Ptr(row.YearsExperience),
		ConsultationFee:    platform.FromPgNumeric(row.ConsultationFee),
		VerificationStatus: row.VerificationStatus,
		Languages:          row.Languages,
		Certificates:       row.Certificates,
		AreasOfExpertise:   row.AreasOfExpertise,
	}
}
