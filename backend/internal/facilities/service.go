package facilities

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

type CreateFacilityInput struct {
	Name     string
	Type     Type
	Region   *string
	District *string
	Phone    *string
	Address  *string
}

func (s *Service) CreateFacility(ctx context.Context, in CreateFacilityInput) (Facility, error) {
	return s.repo.CreateFacility(ctx, in.Name, in.Type, in.Region, in.District, in.Phone, in.Address)
}

func (s *Service) ListFacilities(ctx context.Context) ([]Facility, error) {
	return s.repo.ListFacilities(ctx)
}

// SearchFacilities is a directory-style search by facility name — open to
// any authenticated user, see search.Service.Search.
func (s *Service) SearchFacilities(ctx context.Context, query string, limit int) ([]Facility, error) {
	return s.repo.SearchFacilities(ctx, query, limit)
}

// SearchProviders is a directory-style "doctor" search by the provider's
// linked user's full_name or specialty — open to any authenticated user.
func (s *Service) SearchProviders(ctx context.Context, query string, limit int) ([]ProviderSearchResult, error) {
	return s.repo.SearchProviders(ctx, query, limit)
}

func (s *Service) CountFacilities(ctx context.Context) (int64, error) {
	return s.repo.CountFacilities(ctx)
}

func (s *Service) GetFacility(ctx context.Context, id uuid.UUID) (Facility, error) {
	return s.repo.FindFacilityByID(ctx, id)
}

// UpdateFacilityInput is PATCH /{facilityID}'s body, pre-decoded: a nil
// field means "leave this field unchanged" — same partial-update contract as
// identity.UpdateProfileInput. Only logo_url/working_hours/referral_policies
// are settable here (name/type/region/etc. are out of scope for this
// endpoint).
type UpdateFacilityInput struct {
	LogoURL          *string
	WorkingHours     *[]byte
	ReferralPolicies *string
}

// UpdateFacility is the "read current, overlay supplied fields, write
// merged result" partial update for a facility's logo/working-hours/
// referral-policies, so a PATCH that only sends {"logo_url": "..."}
// doesn't clear working_hours/referral_policies a caller set earlier.
// Mirrors identity.Service.UpdateProfile / mergeProfileUpdate; see
// Repository.UpdateFacilityLogoAndHours for why the merge happens here
// rather than in the repository.
func (s *Service) UpdateFacility(ctx context.Context, id uuid.UUID, in UpdateFacilityInput) (Facility, error) {
	current, err := s.repo.FindFacilityByID(ctx, id)
	if err != nil {
		return Facility{}, err
	}

	logoURL, workingHours, referralPolicies := mergeFacilityUpdate(current, in)
	return s.repo.UpdateFacilityLogoAndHours(ctx, id, logoURL, workingHours, referralPolicies)
}

// mergeFacilityUpdate overlays in's non-nil fields onto current, leaving
// everything else as-is. Pulled out as a pure function (no DB) so the
// partial-update semantics can be unit-tested directly — mirrors
// identity.mergeProfileUpdate.
func mergeFacilityUpdate(current Facility, in UpdateFacilityInput) (logoURL *string, workingHours []byte, referralPolicies *string) {
	logoURL = current.LogoURL
	if in.LogoURL != nil {
		logoURL = in.LogoURL
	}

	workingHours = current.WorkingHours
	if in.WorkingHours != nil {
		workingHours = *in.WorkingHours
	}

	referralPolicies = current.ReferralPolicies
	if in.ReferralPolicies != nil {
		referralPolicies = in.ReferralPolicies
	}

	return logoURL, workingHours, referralPolicies
}

type CreateBranchInput struct {
	FacilityID uuid.UUID
	Name       string
	Address    *string
	Phone      *string
}

func (s *Service) CreateBranch(ctx context.Context, in CreateBranchInput) (Branch, error) {
	if _, err := s.repo.FindFacilityByID(ctx, in.FacilityID); err != nil {
		return Branch{}, err
	}
	return s.repo.CreateBranch(ctx, in.FacilityID, in.Name, in.Address, in.Phone)
}

func (s *Service) ListBranchesByFacility(ctx context.Context, facilityID uuid.UUID) ([]Branch, error) {
	return s.repo.ListBranchesByFacility(ctx, facilityID)
}

type UpdateBranchInput struct {
	Name    string
	Address *string
	Phone   *string
}

// GetBranch is used by the handler to resolve a branch's facility_id ahead
// of an update/delete, so a hospital_admin's facility-scoping check has
// something to compare against before the write happens.
func (s *Service) GetBranch(ctx context.Context, id uuid.UUID) (Branch, error) {
	return s.repo.FindBranchByID(ctx, id)
}

func (s *Service) UpdateBranch(ctx context.Context, id uuid.UUID, in UpdateBranchInput) (Branch, error) {
	return s.repo.UpdateBranch(ctx, id, in.Name, in.Address, in.Phone)
}

func (s *Service) DeleteBranch(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteBranch(ctx, id)
}

type CreateDepartmentInput struct {
	FacilityID uuid.UUID
	Name       string
}

func (s *Service) CreateDepartment(ctx context.Context, in CreateDepartmentInput) (Department, error) {
	if _, err := s.repo.FindFacilityByID(ctx, in.FacilityID); err != nil {
		return Department{}, err
	}
	return s.repo.CreateDepartment(ctx, in.FacilityID, in.Name)
}

func (s *Service) ListDepartmentsByFacility(ctx context.Context, facilityID uuid.UUID) ([]Department, error) {
	return s.repo.ListDepartmentsByFacility(ctx, facilityID)
}

// GetDepartment is used by the handler to resolve a department's
// facility_id ahead of an update/delete, mirroring GetBranch.
func (s *Service) GetDepartment(ctx context.Context, id uuid.UUID) (Department, error) {
	return s.repo.FindDepartmentByID(ctx, id)
}

func (s *Service) UpdateDepartment(ctx context.Context, id uuid.UUID, name string) (Department, error) {
	return s.repo.UpdateDepartment(ctx, id, name)
}

func (s *Service) DeleteDepartment(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteDepartment(ctx, id)
}

type CreateProviderInput struct {
	UserID        uuid.UUID
	FacilityID    uuid.UUID
	Specialty     *string
	LicenseNumber *string
}

func (s *Service) CreateProvider(ctx context.Context, in CreateProviderInput) (Provider, error) {
	if _, err := s.repo.FindFacilityByID(ctx, in.FacilityID); err != nil {
		return Provider{}, err
	}
	return s.repo.CreateProvider(ctx, in.UserID, in.FacilityID, in.Specialty, in.LicenseNumber)
}

// FacilityIDForUser implements identity.FacilityResolver, letting identity
// populate the facility claim on a provider's JWT without depending on this
// package directly.
func (s *Service) FacilityIDForUser(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error) {
	provider, err := s.repo.FindProviderByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrProviderNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &provider.FacilityID, nil
}

// ResolveProviderID implements records.ProviderResolver — maps a user_id
// to the providers.id their encounters get recorded under.
func (s *Service) ResolveProviderID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	provider, err := s.repo.FindProviderByUserID(ctx, userID)
	if err != nil {
		return uuid.UUID{}, err
	}
	return provider.ID, nil
}

// GetProvider looks up a provider profile by its id. Used both by the
// "fetch a provider by id" endpoint and, indirectly, by
// UpdateOwnProviderProfile's read-merge-write.
func (s *Service) GetProvider(ctx context.Context, id uuid.UUID) (Provider, error) {
	return s.repo.FindProviderByID(ctx, id)
}

// GetOwnProviderProfile returns the caller's own provider profile — the
// self-service counterpart to GetProvider, mirroring
// identity.Service.GetProfile's "look the row up by the authenticated
// user's id, not a caller-supplied id" convention.
func (s *Service) GetOwnProviderProfile(ctx context.Context, callerUserID uuid.UUID) (Provider, error) {
	return s.repo.FindProviderByUserID(ctx, callerUserID)
}

// UpdateProviderProfileInput is PATCH /providers/me's body, pre-decoded: a
// nil field means "leave this field unchanged" — same partial-update
// contract as UpdateFacilityInput / identity.UpdateProfileInput.
// Deliberately excludes VerificationStatus (admin-only, see
// UpdateProviderVerificationStatus) and Specialty/LicenseNumber (set at
// creation, not editable via this endpoint).
type UpdateProviderProfileInput struct {
	Qualifications   *[]string
	YearsExperience  *int
	ConsultationFee  *string
	Languages        *[]string
	Certificates     *[]byte
	AreasOfExpertise *[]string
}

// UpdateOwnProviderProfile is the "read current, overlay supplied fields,
// write merged result" partial update for a provider editing their own
// profile. The caller is identified purely by callerUserID (taken from the
// authenticated request's claims, never a client-supplied provider id), so
// there is no separate "is this caller allowed to edit this provider"
// check to get wrong — a user can only ever resolve and update the
// provider row that FindProviderByUserID maps to their own user id, the
// same way identity.Service.UpdateProfile is scoped to the caller's own
// user id. Mirrors Service.UpdateFacility / mergeFacilityUpdate.
func (s *Service) UpdateOwnProviderProfile(ctx context.Context, callerUserID uuid.UUID, in UpdateProviderProfileInput) (Provider, error) {
	current, err := s.repo.FindProviderByUserID(ctx, callerUserID)
	if err != nil {
		return Provider{}, err
	}

	qualifications, yearsExperience, consultationFee, languages, certificates, areasOfExpertise := mergeProviderProfileUpdate(current, in)
	return s.repo.UpdateProviderProfile(ctx, current.ID, qualifications, yearsExperience, consultationFee, languages, certificates, areasOfExpertise)
}

// mergeProviderProfileUpdate overlays in's non-nil fields onto current,
// leaving everything else as-is. Pulled out as a pure function (no DB) so
// the partial-update semantics can be unit-tested directly — mirrors
// mergeFacilityUpdate / identity.mergeProfileUpdate.
func mergeProviderProfileUpdate(current Provider, in UpdateProviderProfileInput) (qualifications []string, yearsExperience *int, consultationFee *string, languages []string, certificates []byte, areasOfExpertise []string) {
	qualifications = current.Qualifications
	if in.Qualifications != nil {
		qualifications = *in.Qualifications
	}

	yearsExperience = current.YearsExperience
	if in.YearsExperience != nil {
		yearsExperience = in.YearsExperience
	}

	consultationFee = current.ConsultationFee
	if in.ConsultationFee != nil {
		consultationFee = in.ConsultationFee
	}

	languages = current.Languages
	if in.Languages != nil {
		languages = *in.Languages
	}

	certificates = current.Certificates
	if in.Certificates != nil {
		certificates = *in.Certificates
	}

	areasOfExpertise = current.AreasOfExpertise
	if in.AreasOfExpertise != nil {
		areasOfExpertise = *in.AreasOfExpertise
	}

	return qualifications, yearsExperience, consultationFee, languages, certificates, areasOfExpertise
}

// ErrInvalidVerificationStatus is returned by
// UpdateProviderVerificationStatus for any status outside the
// providers_verification_status_check constraint's allowed values —
// checked here too (not just left to the DB constraint) so callers get a
// clear 400 instead of a raw constraint-violation error.
var ErrInvalidVerificationStatus = errors.New("invalid verification status")

var validVerificationStatuses = map[string]bool{
	"pending":  true,
	"verified": true,
	"rejected": true,
}

// UpdateProviderVerificationStatus is the admin-only counterpart to
// UpdateOwnProviderProfile. Route-level auth (platform.RequireRoles in
// Handler.Routes) already restricts who can reach this method; it does not
// re-check the caller's role itself, matching how every other admin-only
// method in this service (CreateFacility, CreateProvider, etc.) relies on
// the handler's route grouping rather than duplicating the check.
func (s *Service) UpdateProviderVerificationStatus(ctx context.Context, providerID uuid.UUID, status string) (Provider, error) {
	if !validVerificationStatuses[status] {
		return Provider{}, ErrInvalidVerificationStatus
	}
	return s.repo.UpdateProviderVerificationStatus(ctx, providerID, status)
}
