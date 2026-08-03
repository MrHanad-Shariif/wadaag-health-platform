package appointments

import (
	"errors"
	"net/http"
	"strconv"
	"time"

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

// Routes is mounted at "/appointments" (see server.NewRouter), so the final
// paths are:
//
//	POST   /api/v1/appointments/availability
//	GET    /api/v1/appointments/availability/{providerID}
//	GET    /api/v1/appointments/availability/{providerID}/slots
//	DELETE /api/v1/appointments/availability/{ruleID}
//	POST   /api/v1/appointments
//	GET    /api/v1/appointments/mine
//	GET    /api/v1/appointments/{appointmentID}
//	PATCH  /api/v1/appointments/{appointmentID}/reschedule
//	POST   /api/v1/appointments/{appointmentID}/cancel
//	POST   /api/v1/appointments/{appointmentID}/complete
//	POST   /api/v1/appointments/{appointmentID}/no-show
//
// Unlike records/referrals, there's no consent.Middleware anywhere in this
// tree — an appointment is a scheduling record, not the kind of clinical
// data the consent module gates, so authorization is the simpler party
// check performed directly in the service (see Service.isPartyOrAdmin/
// isProviderOrAdmin), same reasoning as consults.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(platform.RequireAuth(h.tm))

	r.Group(func(w chi.Router) {
		// RequireRoles' system_admin bypass lets the administrator reach
		// this too, alongside hospital_admin managing on a provider's
		// behalf — see Service.resolveManageableProviderID for exactly who
		// may set availability for which provider.
		w.Use(platform.RequireRoles(platform.RolePhysician, platform.RoleHospitalAdmin))
		w.Post("/availability", h.setAvailability)
	})

	// Viewing a provider's schedule (and generated slots) is open to any
	// authenticated user — a patient needs to see this before booking.
	r.Get("/availability/{providerID}", h.listAvailability)
	r.Get("/availability/{providerID}/slots", h.listSlots)

	r.Group(func(w chi.Router) {
		// The owning provider only — no hospital_admin/system_admin
		// delegation here, unlike setAvailability above.
		w.Use(platform.RequireRoles(platform.RolePhysician))
		w.Delete("/availability/{ruleID}", h.deleteAvailability)
	})

	r.Post("/", h.book)
	r.Get("/mine", h.mine)
	r.Get("/{appointmentID}", h.get)
	r.Patch("/{appointmentID}/reschedule", h.reschedule)
	r.Post("/{appointmentID}/cancel", h.cancel)

	r.Group(func(w chi.Router) {
		w.Use(platform.RequireRoles(platform.RolePhysician, platform.RoleHospitalAdmin))
		w.Post("/{appointmentID}/complete", h.complete)
		w.Post("/{appointmentID}/no-show", h.noShow)
	})

	return r
}

type appointmentResponse struct {
	ID         string  `json:"id"`
	PatientID  string  `json:"patient_id"`
	ProviderID string  `json:"provider_id"`
	FacilityID string  `json:"facility_id"`
	StartAt    string  `json:"start_at"`
	EndAt      string  `json:"end_at"`
	Status     string  `json:"status"`
	Reason     *string `json:"reason,omitempty"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

func toAppointmentResponse(a Appointment) appointmentResponse {
	return appointmentResponse{
		ID: a.ID.String(), PatientID: a.PatientID.String(), ProviderID: a.ProviderID.String(),
		FacilityID: a.FacilityID.String(), StartAt: a.StartAt.Format(time.RFC3339), EndAt: a.EndAt.Format(time.RFC3339),
		Status: string(a.Status), Reason: a.Reason,
		CreatedAt: a.CreatedAt.Format(time.RFC3339), UpdatedAt: a.UpdatedAt.Format(time.RFC3339),
	}
}

type availabilityResponse struct {
	ID         string `json:"id"`
	ProviderID string `json:"provider_id"`
	Weekday    int    `json:"weekday"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
}

func toAvailabilityResponse(a AvailabilityRule) availabilityResponse {
	return availabilityResponse{
		ID: a.ID.String(), ProviderID: a.ProviderID.String(), Weekday: int(a.Weekday),
		StartTime: a.StartTime, EndTime: a.EndTime,
	}
}

type slotResponse struct {
	StartAt string `json:"start_at"`
	EndAt   string `json:"end_at"`
}

func toSlotResponse(s Slot) slotResponse {
	return slotResponse{StartAt: s.StartAt.Format(time.RFC3339), EndAt: s.EndAt.Format(time.RFC3339)}
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrAppointmentNotFound):
		platform.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrAppointmentNotModifiable):
		platform.WriteError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrSlotUnavailable):
		platform.WriteError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrNotAuthorized):
		platform.WriteError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrPatientIDRequired), errors.Is(err, ErrProviderIDInvalid),
		errors.Is(err, ErrProviderIDRequired), errors.Is(err, ErrInvalidTimeRange),
		errors.Is(err, ErrInvalidWeekday), errors.Is(err, ErrInvalidTimeOfDay):
		platform.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		platform.WriteError(w, http.StatusInternalServerError, "failed to process appointment request")
	}
}

type setAvailabilityRequest struct {
	ProviderID *string `json:"provider_id"`
	Weekday    int     `json:"weekday"`
	StartTime  string  `json:"start_time"`
	EndTime    string  `json:"end_time"`
}

func (h *Handler) setAvailability(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	var req setAvailabilityRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var targetProviderID *uuid.UUID
	if req.ProviderID != nil && *req.ProviderID != "" {
		parsed, err := uuid.Parse(*req.ProviderID)
		if err != nil {
			platform.WriteError(w, http.StatusBadRequest, "invalid provider_id")
			return
		}
		targetProviderID = &parsed
	}

	rule, err := h.service.SetAvailability(
		r.Context(), claims.UserID, claims.Role, claims.FacilityID, targetProviderID,
		int16(req.Weekday), req.StartTime, req.EndTime,
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	platform.WriteJSON(w, http.StatusCreated, toAvailabilityResponse(rule))
}

func (h *Handler) listAvailability(w http.ResponseWriter, r *http.Request) {
	providerID, err := uuid.Parse(chi.URLParam(r, "providerID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid provider id")
		return
	}

	rules, err := h.service.ListAvailability(r.Context(), providerID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to list availability")
		return
	}

	out := make([]availabilityResponse, len(rules))
	for i, rule := range rules {
		out[i] = toAvailabilityResponse(rule)
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) listSlots(w http.ResponseWriter, r *http.Request) {
	providerID, err := uuid.Parse(chi.URLParam(r, "providerID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid provider id")
		return
	}

	dateStr := r.URL.Query().Get("date")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid or missing date, expected YYYY-MM-DD")
		return
	}

	slotMinutes := 30
	if raw := r.URL.Query().Get("slot_minutes"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			platform.WriteError(w, http.StatusBadRequest, "invalid slot_minutes")
			return
		}
		slotMinutes = parsed
	}

	slots, err := h.service.GetAvailableSlots(r.Context(), providerID, date, slotMinutes)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to compute available slots")
		return
	}

	out := make([]slotResponse, len(slots))
	for i, s := range slots {
		out[i] = toSlotResponse(s)
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) deleteAvailability(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	ruleID, err := uuid.Parse(chi.URLParam(r, "ruleID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid rule id")
		return
	}

	if err := h.service.DeleteAvailability(r.Context(), claims.UserID, ruleID); err != nil {
		platform.WriteError(w, http.StatusForbidden, "actor has no provider profile")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type bookRequest struct {
	PatientID  *string `json:"patient_id"`
	ProviderID string  `json:"provider_id"`
	FacilityID string  `json:"facility_id"`
	StartAt    string  `json:"start_at"`
	EndAt      string  `json:"end_at"`
	Reason     *string `json:"reason"`
}

func (h *Handler) book(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	var req bookRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var patientID *uuid.UUID
	if req.PatientID != nil && *req.PatientID != "" {
		parsed, err := uuid.Parse(*req.PatientID)
		if err != nil {
			platform.WriteError(w, http.StatusBadRequest, "invalid patient_id")
			return
		}
		patientID = &parsed
	}

	providerID, err := uuid.Parse(req.ProviderID)
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid provider_id")
		return
	}
	facilityID, err := uuid.Parse(req.FacilityID)
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid facility_id")
		return
	}
	startAt, err := time.Parse(time.RFC3339, req.StartAt)
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid start_at, expected RFC3339")
		return
	}
	endAt, err := time.Parse(time.RFC3339, req.EndAt)
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid end_at, expected RFC3339")
		return
	}

	appointment, err := h.service.Book(r.Context(), claims.UserID, claims.Role, BookInput{
		PatientID: patientID, ProviderID: providerID, FacilityID: facilityID,
		StartAt: startAt, EndAt: endAt, Reason: req.Reason,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	platform.WriteJSON(w, http.StatusCreated, toAppointmentResponse(appointment))
}

// mine is role-aware:
//   - patient gets their own appointments (self-resolved, same as booking).
//   - physician gets their own upcoming schedule (now through 90 days out —
//     a fixed forward window rather than an "everything ever" list, since
//     this is meant to answer "what's on my schedule," not be a full
//     history browser).
//   - hospital_admin/system_admin: neither role necessarily has a provider
//     profile of their own, and there's no facility-wide appointment list
//     in this phase (the design doc doesn't ask for one, and Phase 9's
//     dashboard is the more natural home for that kind of aggregate view),
//     so they get their own schedule only if they *also* happen to be a
//     registered provider, and an empty list otherwise rather than a 403 —
//     this is a judgment call, noted here rather than in the design doc.
func (h *Handler) mine(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	var list []Appointment
	var err error
	switch claims.Role {
	case platform.RolePatient:
		var patientID uuid.UUID
		patientID, err = h.service.ResolveOwnPatientID(r.Context(), claims.UserID)
		if err != nil {
			platform.WriteError(w, http.StatusForbidden, "actor has no patient profile")
			return
		}
		list, err = h.service.ListForPatient(r.Context(), patientID)
	default:
		var providerID uuid.UUID
		providerID, err = h.service.ResolveProviderID(r.Context(), claims.UserID)
		if err != nil {
			// No provider profile of their own (e.g. a hospital_admin/
			// system_admin who isn't also a registered provider) — an
			// empty schedule, not an error, per the doc comment above.
			platform.WriteJSON(w, http.StatusOK, []appointmentResponse{})
			return
		}
		now := time.Now()
		list, err = h.service.ListForProvider(r.Context(), providerID, now, now.AddDate(0, 0, 90))
	}
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to list appointments")
		return
	}

	out := make([]appointmentResponse, len(list))
	for i, a := range list {
		out[i] = toAppointmentResponse(a)
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) loadAppointment(w http.ResponseWriter, r *http.Request) (Appointment, bool) {
	appointmentID, err := uuid.Parse(chi.URLParam(r, "appointmentID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid appointment id")
		return Appointment{}, false
	}

	appointment, err := h.service.Get(r.Context(), appointmentID)
	if err != nil {
		writeServiceError(w, err)
		return Appointment{}, false
	}
	return appointment, true
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	appointment, ok := h.loadAppointment(w, r)
	if !ok {
		return
	}

	if !h.service.isPartyOrAdmin(r.Context(), appointment, claims.UserID, claims.Role, claims.FacilityID) {
		platform.WriteError(w, http.StatusForbidden, ErrNotAuthorized.Error())
		return
	}

	platform.WriteJSON(w, http.StatusOK, toAppointmentResponse(appointment))
}

type rescheduleRequest struct {
	StartAt string `json:"start_at"`
	EndAt   string `json:"end_at"`
}

func (h *Handler) reschedule(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	appointmentID, err := uuid.Parse(chi.URLParam(r, "appointmentID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid appointment id")
		return
	}

	var req rescheduleRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	startAt, err := time.Parse(time.RFC3339, req.StartAt)
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid start_at, expected RFC3339")
		return
	}
	endAt, err := time.Parse(time.RFC3339, req.EndAt)
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid end_at, expected RFC3339")
		return
	}

	appointment, err := h.service.Reschedule(r.Context(), claims.UserID, claims.Role, claims.FacilityID, appointmentID, startAt, endAt)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	platform.WriteJSON(w, http.StatusOK, toAppointmentResponse(appointment))
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}
	appointmentID, err := uuid.Parse(chi.URLParam(r, "appointmentID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid appointment id")
		return
	}

	appointment, err := h.service.Cancel(r.Context(), claims.UserID, claims.Role, claims.FacilityID, appointmentID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	platform.WriteJSON(w, http.StatusOK, toAppointmentResponse(appointment))
}

func (h *Handler) complete(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}
	appointmentID, err := uuid.Parse(chi.URLParam(r, "appointmentID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid appointment id")
		return
	}

	appointment, err := h.service.Complete(r.Context(), claims.UserID, claims.Role, claims.FacilityID, appointmentID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	platform.WriteJSON(w, http.StatusOK, toAppointmentResponse(appointment))
}

func (h *Handler) noShow(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}
	appointmentID, err := uuid.Parse(chi.URLParam(r, "appointmentID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid appointment id")
		return
	}

	appointment, err := h.service.NoShow(r.Context(), claims.UserID, claims.Role, claims.FacilityID, appointmentID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	platform.WriteJSON(w, http.StatusOK, toAppointmentResponse(appointment))
}
