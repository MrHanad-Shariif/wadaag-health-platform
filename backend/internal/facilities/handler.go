package facilities

import (
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

	r.Group(func(admin chi.Router) {
		admin.Use(platform.RequireRoles(platform.RoleSystemAdmin, platform.RoleHospitalAdmin))
		admin.Post("/", h.create)
		admin.Post("/{facilityID}/providers", h.addProvider)
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
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Type     Type    `json:"type"`
	Region   *string `json:"region,omitempty"`
	District *string `json:"district,omitempty"`
	Phone    *string `json:"phone,omitempty"`
	Address  *string `json:"address,omitempty"`
}

func toFacilityResponse(f Facility) facilityResponse {
	return facilityResponse{
		ID:       f.ID.String(),
		Name:     f.Name,
		Type:     f.Type,
		Region:   f.Region,
		District: f.District,
		Phone:    f.Phone,
		Address:  f.Address,
	}
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
	ID            string  `json:"id"`
	UserID        string  `json:"user_id"`
	FacilityID    string  `json:"facility_id"`
	Specialty     *string `json:"specialty,omitempty"`
	LicenseNumber *string `json:"license_number,omitempty"`
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

	platform.WriteJSON(w, http.StatusCreated, providerResponse{
		ID: provider.ID.String(), UserID: provider.UserID.String(), FacilityID: provider.FacilityID.String(),
		Specialty: provider.Specialty, LicenseNumber: provider.LicenseNumber,
	})
}
