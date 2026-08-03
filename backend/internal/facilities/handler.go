package facilities

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/platform"
)

type Handler struct {
	service *Service
	tm      *platform.TokenManager
}

func NewHandler(service *Service, tm *platform.TokenManager) *Handler {
	return &Handler{service: service, tm: tm}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(platform.RequireAuth(h.tm))

	r.Get("/", h.list)
	r.Get("/{facilityID}", h.get)
	r.Get("/{facilityID}/branches", h.listBranches)
	r.Get("/{facilityID}/departments", h.listDepartments)

	// "/providers" is a static path segment, so chi routes it ahead of the
	// "/{facilityID}" wildcard above rather than trying to parse "providers"
	// as a facility id — same reasoning identity.Handler.Routes relies on for
	// "/me/..." alongside any future "/{userID}"-shaped route.
	r.Get("/providers/me", h.getOwnProviderProfile)
	r.Patch("/providers/me", h.updateOwnProviderProfile)
	r.Get("/providers/{providerID}", h.getProvider)

	r.Group(func(admin chi.Router) {
		admin.Use(platform.RequireRoles(platform.RoleSystemAdmin, platform.RoleHospitalAdmin))
		admin.Post("/", h.create)
		admin.Post("/{facilityID}/providers", h.addProvider)
		admin.Patch("/{facilityID}", h.updateFacility)
		admin.Patch("/providers/{providerID}/verification-status", h.updateProviderVerificationStatus)
		admin.Post("/{facilityID}/branches", h.createBranch)
		admin.Patch("/branches/{branchID}", h.updateBranch)
		admin.Delete("/branches/{branchID}", h.deleteBranch)
		admin.Post("/{facilityID}/departments", h.createDepartment)
		admin.Patch("/departments/{departmentID}", h.updateDepartment)
		admin.Delete("/departments/{departmentID}", h.deleteDepartment)
	})

	return r
}

type facilityRequest struct {
	Name     string  `json:"name"`
	Type     Type    `json:"type"`
	Region   *string `json:"region"`
	District *string `json:"district"`
	Phone    *string `json:"phone"`
	Address  *string `json:"address"`
}

type facilityResponse struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Type         Type            `json:"type"`
	Region       *string         `json:"region,omitempty"`
	District     *string         `json:"district,omitempty"`
	Phone        *string         `json:"phone,omitempty"`
	Address      *string         `json:"address,omitempty"`
	LogoURL      *string         `json:"logo_url,omitempty"`
	WorkingHours json.RawMessage `json:"working_hours,omitempty"`
}

func toFacilityResponse(f Facility) facilityResponse {
	return facilityResponse{
		ID:           f.ID.String(),
		Name:         f.Name,
		Type:         f.Type,
		Region:       f.Region,
		District:     f.District,
		Phone:        f.Phone,
		Address:      f.Address,
		LogoURL:      f.LogoURL,
		WorkingHours: f.WorkingHours,
	}
}

// updateFacilityRequest is PATCH /{facilityID}'s body — logo_url and
// working_hours only, matching Service.UpdateFacilityInput's partial-update
// contract (a nil/omitted field leaves the current value unchanged).
type updateFacilityRequest struct {
	LogoURL      *string         `json:"logo_url"`
	WorkingHours json.RawMessage `json:"working_hours"`
}

func (h *Handler) updateFacility(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "facilityID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid facility id")
		return
	}

	var req updateFacilityRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	in := UpdateFacilityInput{LogoURL: req.LogoURL}
	if req.WorkingHours != nil {
		wh := []byte(req.WorkingHours)
		in.WorkingHours = &wh
	}

	facility, err := h.service.UpdateFacility(r.Context(), id, in)
	if err != nil {
		if errors.Is(err, ErrFacilityNotFound) {
			platform.WriteError(w, http.StatusNotFound, "facility not found")
			return
		}
		platform.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	platform.WriteJSON(w, http.StatusOK, toFacilityResponse(facility))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req facilityRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	facility, err := h.service.CreateFacility(r.Context(), CreateFacilityInput{
		Name: req.Name, Type: req.Type, Region: req.Region, District: req.District, Phone: req.Phone, Address: req.Address,
	})
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	platform.WriteJSON(w, http.StatusCreated, toFacilityResponse(facility))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.service.ListFacilities(r.Context())
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to list facilities")
		return
	}

	out := make([]facilityResponse, len(list))
	for i, f := range list {
		out[i] = toFacilityResponse(f)
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "facilityID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid facility id")
		return
	}

	facility, err := h.service.GetFacility(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrFacilityNotFound) {
			platform.WriteError(w, http.StatusNotFound, "facility not found")
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "failed to load facility")
		return
	}

	platform.WriteJSON(w, http.StatusOK, toFacilityResponse(facility))
}

type addProviderRequest struct {
	UserID        string  `json:"user_id"`
	Specialty     *string `json:"specialty"`
	LicenseNumber *string `json:"license_number"`
}

type providerResponse struct {
	ID                 string          `json:"id"`
	UserID             string          `json:"user_id"`
	FacilityID         string          `json:"facility_id"`
	Specialty          *string         `json:"specialty,omitempty"`
	LicenseNumber      *string         `json:"license_number,omitempty"`
	Qualifications     []string        `json:"qualifications,omitempty"`
	YearsExperience    *int            `json:"years_experience,omitempty"`
	ConsultationFee    *string         `json:"consultation_fee,omitempty"`
	VerificationStatus string          `json:"verification_status"`
	Languages          []string        `json:"languages,omitempty"`
	Certificates       json.RawMessage `json:"certificates,omitempty"`
	AreasOfExpertise   []string        `json:"areas_of_expertise,omitempty"`
}

func toProviderResponse(p Provider) providerResponse {
	return providerResponse{
		ID:                 p.ID.String(),
		UserID:             p.UserID.String(),
		FacilityID:         p.FacilityID.String(),
		Specialty:          p.Specialty,
		LicenseNumber:      p.LicenseNumber,
		Qualifications:     p.Qualifications,
		YearsExperience:    p.YearsExperience,
		ConsultationFee:    p.ConsultationFee,
		VerificationStatus: p.VerificationStatus,
		Languages:          p.Languages,
		Certificates:       p.Certificates,
		AreasOfExpertise:   p.AreasOfExpertise,
	}
}

func (h *Handler) addProvider(w http.ResponseWriter, r *http.Request) {
	facilityID, err := uuid.Parse(chi.URLParam(r, "facilityID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid facility id")
		return
	}

	var req addProviderRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	provider, err := h.service.CreateProvider(r.Context(), CreateProviderInput{
		UserID: userID, FacilityID: facilityID, Specialty: req.Specialty, LicenseNumber: req.LicenseNumber,
	})
	if err != nil {
		if errors.Is(err, ErrDuplicateProvider) {
			platform.WriteError(w, http.StatusConflict, err.Error())
			return
		}
		platform.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	platform.WriteJSON(w, http.StatusCreated, toProviderResponse(provider))
}

// getProvider fetches a provider profile by its id. Any authenticated
// caller can look up any provider by id (matching the public-directory
// nature of the {facilityID}/{providerID} read paths elsewhere in this
// handler) — this is a read, not the self/admin-gated write paths below.
func (h *Handler) getProvider(w http.ResponseWriter, r *http.Request) {
	providerID, err := uuid.Parse(chi.URLParam(r, "providerID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid provider id")
		return
	}

	provider, err := h.service.GetProvider(r.Context(), providerID)
	if err != nil {
		if errors.Is(err, ErrProviderNotFound) {
			platform.WriteError(w, http.StatusNotFound, "provider not found")
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "failed to load provider")
		return
	}

	platform.WriteJSON(w, http.StatusOK, toProviderResponse(provider))
}

// getOwnProviderProfile returns the caller's own provider profile —
// mirrors identity.Handler.getProfile's shape (pull the subject from
// claims, not the URL).
func (h *Handler) getOwnProviderProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	provider, err := h.service.GetOwnProviderProfile(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, ErrProviderNotFound) {
			platform.WriteError(w, http.StatusNotFound, "provider profile not found")
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "failed to load provider profile")
		return
	}

	platform.WriteJSON(w, http.StatusOK, toProviderResponse(provider))
}

// updateProviderProfileRequest is PATCH /providers/me's body, matching
// Service.UpdateProviderProfileInput's partial-update contract (a
// nil/omitted field leaves the current value unchanged). Deliberately has
// no verification_status/specialty/license_number fields — those aren't
// editable through this endpoint.
type updateProviderProfileRequest struct {
	Qualifications   *[]string       `json:"qualifications"`
	YearsExperience  *int            `json:"years_experience"`
	ConsultationFee  *string         `json:"consultation_fee"`
	Languages        *[]string       `json:"languages"`
	Certificates     json.RawMessage `json:"certificates"`
	AreasOfExpertise *[]string       `json:"areas_of_expertise"`
}

// updateOwnProviderProfile lets a provider edit their own profile
// (qualifications, years of experience, consultation fee, languages,
// certificates, areas of expertise). Which provider is "own" comes purely
// from claims.UserID, never a client-supplied id — see
// Service.UpdateOwnProviderProfile.
func (h *Handler) updateOwnProviderProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	var req updateProviderProfileRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	in := UpdateProviderProfileInput{
		Qualifications:   req.Qualifications,
		YearsExperience:  req.YearsExperience,
		ConsultationFee:  req.ConsultationFee,
		Languages:        req.Languages,
		AreasOfExpertise: req.AreasOfExpertise,
	}
	if req.Certificates != nil {
		c := []byte(req.Certificates)
		in.Certificates = &c
	}

	provider, err := h.service.UpdateOwnProviderProfile(r.Context(), claims.UserID, in)
	if err != nil {
		if errors.Is(err, ErrProviderNotFound) {
			platform.WriteError(w, http.StatusNotFound, "provider profile not found")
			return
		}
		platform.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	platform.WriteJSON(w, http.StatusOK, toProviderResponse(provider))
}

type updateProviderVerificationStatusRequest struct {
	VerificationStatus string `json:"verification_status"`
}

// updateProviderVerificationStatus is the admin-only counterpart to
// updateOwnProviderProfile — gated by the same
// platform.RequireRoles(RoleSystemAdmin, RoleHospitalAdmin) group as every
// other admin route in this handler (addProvider, updateFacility, etc.).
func (h *Handler) updateProviderVerificationStatus(w http.ResponseWriter, r *http.Request) {
	providerID, err := uuid.Parse(chi.URLParam(r, "providerID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid provider id")
		return
	}

	var req updateProviderVerificationStatusRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	provider, err := h.service.UpdateProviderVerificationStatus(r.Context(), providerID, req.VerificationStatus)
	if err != nil {
		if errors.Is(err, ErrProviderNotFound) {
			platform.WriteError(w, http.StatusNotFound, "provider not found")
			return
		}
		platform.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	platform.WriteJSON(w, http.StatusOK, toProviderResponse(provider))
}

type branchRequest struct {
	Name    string  `json:"name"`
	Address *string `json:"address"`
	Phone   *string `json:"phone"`
}

type branchResponse struct {
	ID         string  `json:"id"`
	FacilityID string  `json:"facility_id"`
	Name       string  `json:"name"`
	Address    *string `json:"address,omitempty"`
	Phone      *string `json:"phone,omitempty"`
}

func toBranchResponse(b Branch) branchResponse {
	return branchResponse{
		ID: b.ID.String(), FacilityID: b.FacilityID.String(),
		Name: b.Name, Address: b.Address, Phone: b.Phone,
	}
}

func (h *Handler) createBranch(w http.ResponseWriter, r *http.Request) {
	facilityID, err := uuid.Parse(chi.URLParam(r, "facilityID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid facility id")
		return
	}

	var req branchRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	branch, err := h.service.CreateBranch(r.Context(), CreateBranchInput{
		FacilityID: facilityID, Name: req.Name, Address: req.Address, Phone: req.Phone,
	})
	if err != nil {
		if errors.Is(err, ErrFacilityNotFound) {
			platform.WriteError(w, http.StatusNotFound, "facility not found")
			return
		}
		platform.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	platform.WriteJSON(w, http.StatusCreated, toBranchResponse(branch))
}

func (h *Handler) listBranches(w http.ResponseWriter, r *http.Request) {
	facilityID, err := uuid.Parse(chi.URLParam(r, "facilityID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid facility id")
		return
	}

	list, err := h.service.ListBranchesByFacility(r.Context(), facilityID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to list branches")
		return
	}

	out := make([]branchResponse, len(list))
	for i, b := range list {
		out[i] = toBranchResponse(b)
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) updateBranch(w http.ResponseWriter, r *http.Request) {
	branchID, err := uuid.Parse(chi.URLParam(r, "branchID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid branch id")
		return
	}

	var req branchRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	branch, err := h.service.UpdateBranch(r.Context(), branchID, UpdateBranchInput{
		Name: req.Name, Address: req.Address, Phone: req.Phone,
	})
	if err != nil {
		if errors.Is(err, ErrBranchNotFound) {
			platform.WriteError(w, http.StatusNotFound, "branch not found")
			return
		}
		platform.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	platform.WriteJSON(w, http.StatusOK, toBranchResponse(branch))
}

func (h *Handler) deleteBranch(w http.ResponseWriter, r *http.Request) {
	branchID, err := uuid.Parse(chi.URLParam(r, "branchID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid branch id")
		return
	}

	if err := h.service.DeleteBranch(r.Context(), branchID); err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to delete branch")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type departmentRequest struct {
	Name string `json:"name"`
}

type departmentResponse struct {
	ID         string `json:"id"`
	FacilityID string `json:"facility_id"`
	Name       string `json:"name"`
}

func toDepartmentResponse(d Department) departmentResponse {
	return departmentResponse{ID: d.ID.String(), FacilityID: d.FacilityID.String(), Name: d.Name}
}

func (h *Handler) createDepartment(w http.ResponseWriter, r *http.Request) {
	facilityID, err := uuid.Parse(chi.URLParam(r, "facilityID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid facility id")
		return
	}

	var req departmentRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	department, err := h.service.CreateDepartment(r.Context(), CreateDepartmentInput{
		FacilityID: facilityID, Name: req.Name,
	})
	if err != nil {
		if errors.Is(err, ErrFacilityNotFound) {
			platform.WriteError(w, http.StatusNotFound, "facility not found")
			return
		}
		platform.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	platform.WriteJSON(w, http.StatusCreated, toDepartmentResponse(department))
}

func (h *Handler) listDepartments(w http.ResponseWriter, r *http.Request) {
	facilityID, err := uuid.Parse(chi.URLParam(r, "facilityID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid facility id")
		return
	}

	list, err := h.service.ListDepartmentsByFacility(r.Context(), facilityID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to list departments")
		return
	}

	out := make([]departmentResponse, len(list))
	for i, d := range list {
		out[i] = toDepartmentResponse(d)
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) updateDepartment(w http.ResponseWriter, r *http.Request) {
	departmentID, err := uuid.Parse(chi.URLParam(r, "departmentID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid department id")
		return
	}

	var req departmentRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	department, err := h.service.UpdateDepartment(r.Context(), departmentID, req.Name)
	if err != nil {
		if errors.Is(err, ErrDepartmentNotFound) {
			platform.WriteError(w, http.StatusNotFound, "department not found")
			return
		}
		platform.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	platform.WriteJSON(w, http.StatusOK, toDepartmentResponse(department))
}

func (h *Handler) deleteDepartment(w http.ResponseWriter, r *http.Request) {
	departmentID, err := uuid.Parse(chi.URLParam(r, "departmentID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid department id")
		return
	}

	if err := h.service.DeleteDepartment(r.Context(), departmentID); err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to delete department")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
