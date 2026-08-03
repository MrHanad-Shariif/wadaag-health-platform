package search

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/consults"
	"github.com/wadaag/health-platform/backend/internal/facilities"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"github.com/wadaag/health-platform/backend/internal/records"
	"github.com/wadaag/health-platform/backend/internal/referrals"
)

// perFacetLimit caps how many rows each individual facet query returns, so
// a mixed/fanned-out search across five facets stays fast and the response
// stays small.
const perFacetLimit = 10

// Service orchestrates by calling into each owning module's existing
// service methods (records/facilities/referrals/consults) rather than
// querying other modules' tables directly — each module still owns its own
// SQL (see the SearchPatientsForFacility-style methods added to those
// packages), this package just fans the query out and maps results into a
// single shape.
type Service struct {
	repo       *Repository
	records    *records.Service
	facilities *facilities.Service
	referrals  *referrals.Service
	consults   *consults.Service
}

func NewService(repo *Repository, recordsService *records.Service, facilitiesService *facilities.Service, referralsService *referrals.Service, consultsService *consults.Service) *Service {
	return &Service{
		repo: repo, records: recordsService, facilities: facilitiesService,
		referrals: referralsService, consults: consultsService,
	}
}

// wantsFacet reports whether facet should be evaluated given the caller's
// requested types — an empty types slice means "every facet the caller has
// access to."
func wantsFacet(types []string, facet string) bool {
	if len(types) == 0 {
		return true
	}
	return slices.Contains(types, facet)
}

// canSearchPatients gates the patient facet to the same roles that can
// already reach a patient directory (records.Service.ListPatients'
// system_admin-only doc comment, extended to physician/hospital_admin here
// because — unlike ListPatients — SearchPatients scopes those two roles to
// their own facility's consented patients rather than handing them the
// unscoped view). Every other role (patient, lab_tech, pharmacist, insurer)
// has no patient-directory concept at all, so this facet is omitted for
// them rather than silently returning nothing they could confuse for "no
// matches."
func canSearchPatients(role platform.Role) bool {
	return role == platform.RoleSystemAdmin || role == platform.RolePhysician || role == platform.RoleHospitalAdmin
}

// canSearchReferrals mirrors referrals.Handler.inbox's own gating: any role
// can call it, but it only produces something for system_admin (unscoped)
// or a caller with a facility affiliation (scoped to their inbox). A
// facility-less non-admin (e.g. a patient account) has no referral inbox to
// search, so the facet is omitted for them.
func canSearchReferrals(role platform.Role, facilityID *uuid.UUID) bool {
	return role == platform.RoleSystemAdmin || facilityID != nil
}

// canSearchConsultations mirrors consults.Handler.inbox's gating: any role
// can call it, but it only produces something for system_admin (unscoped)
// or a caller who resolves to a real provider identity.
func canSearchConsultations(role platform.Role, providerID *uuid.UUID) bool {
	return role == platform.RoleSystemAdmin || providerID != nil
}

// Search fans query out across every facet the caller is allowed to use
// (or, if types is non-empty, just those of the requested facets the
// caller is allowed to use), capping each facet at perFacetLimit rows.
//
// Response-shape judgment call: when a facet is explicitly requested via
// types, its key is always present in the result (even an empty slice) so
// the caller can tell "you asked, there's nothing" from "this facet wasn't
// searched." When types is empty (search everything the caller can reach),
// only facets that actually returned a hit get a key — an all-facets
// overview response isn't cluttered with empty arrays for facets that just
// didn't match.
func (s *Service) Search(ctx context.Context, actorUserID uuid.UUID, actorRole platform.Role, actorFacilityID *uuid.UUID, actorProviderID *uuid.UUID, query string, types []string) (map[string][]Result, error) {
	explicit := len(types) > 0
	out := make(map[string][]Result)

	include := func(facet string, results []Result) {
		if explicit || len(results) > 0 {
			out[facet] = results
		}
	}

	if wantsFacet(types, FacetPatient) && canSearchPatients(actorRole) {
		patients, err := s.records.SearchPatients(ctx, actorRole, actorFacilityID, query, perFacetLimit)
		if err != nil {
			return nil, fmt.Errorf("search patients: %w", err)
		}
		include(FacetPatient, patientResults(patients))
	}

	if wantsFacet(types, FacetDoctor) {
		providers, err := s.facilities.SearchProviders(ctx, query, perFacetLimit)
		if err != nil {
			return nil, fmt.Errorf("search doctors: %w", err)
		}
		include(FacetDoctor, doctorResults(providers))
	}

	if wantsFacet(types, FacetHospital) {
		facs, err := s.facilities.SearchFacilities(ctx, query, perFacetLimit)
		if err != nil {
			return nil, fmt.Errorf("search hospitals: %w", err)
		}
		include(FacetHospital, hospitalResults(facs))
	}

	if wantsFacet(types, FacetReferral) && canSearchReferrals(actorRole, actorFacilityID) {
		var refs []referrals.Referral
		var err error
		if actorRole == platform.RoleSystemAdmin {
			refs, err = s.referrals.SearchAll(ctx, query, perFacetLimit)
		} else {
			refs, err = s.referrals.SearchForFacility(ctx, *actorFacilityID, query, perFacetLimit)
		}
		if err != nil {
			return nil, fmt.Errorf("search referrals: %w", err)
		}
		include(FacetReferral, referralResults(refs))
	}

	if wantsFacet(types, FacetConsultation) && canSearchConsultations(actorRole, actorProviderID) {
		var cs []consults.Consultation
		var err error
		if actorRole == platform.RoleSystemAdmin {
			cs, err = s.consults.SearchAll(ctx, query, perFacetLimit)
		} else {
			cs, err = s.consults.SearchForProvider(ctx, *actorProviderID, query, perFacetLimit)
		}
		if err != nil {
			return nil, fmt.Errorf("search consultations: %w", err)
		}
		include(FacetConsultation, consultationResults(cs))
	}

	return out, nil
}

func patientResults(patients []records.Patient) []Result {
	out := make([]Result, len(patients))
	for i, p := range patients {
		subtitle := ""
		switch {
		case p.NationalID != nil:
			subtitle = *p.NationalID
		case p.Phone != nil:
			subtitle = *p.Phone
		}
		out[i] = Result{Type: FacetPatient, ID: p.ID, Title: p.FullName, Subtitle: subtitle, CreatedAt: p.CreatedAt}
	}
	return out
}

func doctorResults(providers []facilities.ProviderSearchResult) []Result {
	out := make([]Result, len(providers))
	for i, p := range providers {
		title := ""
		if p.UserFullName != nil {
			title = *p.UserFullName
		}
		subtitle := ""
		if p.Specialty != nil {
			subtitle = *p.Specialty
		}
		out[i] = Result{Type: FacetDoctor, ID: p.ProviderID, Title: title, Subtitle: subtitle, CreatedAt: p.CreatedAt}
	}
	return out
}

func hospitalResults(facs []facilities.Facility) []Result {
	out := make([]Result, len(facs))
	for i, f := range facs {
		subtitle := ""
		switch {
		case f.Region != nil && f.District != nil:
			subtitle = *f.Region + ", " + *f.District
		case f.Region != nil:
			subtitle = *f.Region
		case f.District != nil:
			subtitle = *f.District
		}
		out[i] = Result{Type: FacetHospital, ID: f.ID, Title: f.Name, Subtitle: subtitle, CreatedAt: f.CreatedAt}
	}
	return out
}

func referralResults(refs []referrals.Referral) []Result {
	out := make([]Result, len(refs))
	for i, r := range refs {
		out[i] = Result{Type: FacetReferral, ID: r.ID, Title: r.Reason, Subtitle: string(r.Status), CreatedAt: r.CreatedAt}
	}
	return out
}

func consultationResults(cs []consults.Consultation) []Result {
	out := make([]Result, len(cs))
	for i, c := range cs {
		out[i] = Result{Type: FacetConsultation, ID: c.ID, Title: c.Reason, Subtitle: string(c.Status), CreatedAt: c.CreatedAt}
	}
	return out
}

// SaveSearch persists a named query for the caller. filters is stored
// as-is (raw JSON, no shape validation for this pass) — an empty/nil
// filters means "no filters," stored as "{}" rather than SQL NULL, matching
// the saved_searches.filters column's NOT NULL DEFAULT '{}'::jsonb.
func (s *Service) SaveSearch(ctx context.Context, userID uuid.UUID, name, query string, filters []byte) (SavedSearch, error) {
	if len(filters) == 0 {
		filters = []byte("{}")
	}
	return s.repo.CreateSavedSearch(ctx, userID, name, query, filters)
}

func (s *Service) ListSavedSearches(ctx context.Context, userID uuid.UUID) ([]SavedSearch, error) {
	return s.repo.ListSavedSearchesForUser(ctx, userID)
}

// ErrNotSavedSearchOwner is returned by DeleteSavedSearch when id exists
// but belongs to a different user — kept distinct from
// ErrSavedSearchNotFound so the handler can tell 404 (no such row) apart
// from 403 (exists, not yours).
var ErrNotSavedSearchOwner = fmt.Errorf("saved search does not belong to this user")

// DeleteSavedSearch deletes id, but only if it belongs to userID —
// ErrSavedSearchNotFound if no such row exists at all, ErrNotSavedSearchOwner
// if it exists but belongs to someone else.
func (s *Service) DeleteSavedSearch(ctx context.Context, userID, id uuid.UUID) error {
	existing, err := s.repo.FindSavedSearchByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.UserID != userID {
		return ErrNotSavedSearchOwner
	}
	_, err = s.repo.DeleteSavedSearch(ctx, id, userID)
	return err
}
