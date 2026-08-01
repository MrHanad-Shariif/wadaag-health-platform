package records

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/audit"
	"github.com/wadaag/health-platform/backend/internal/consent"
	"github.com/wadaag/health-platform/backend/internal/platform"
)

type Handler struct {
	service *Service
	consent *consent.Checker
	audit   *audit.Logger
	tm      *platform.TokenManager
}

func NewHandler(service *Service, consentChecker *consent.Checker, auditLogger *audit.Logger, tm *platform.TokenManager) *Handler {
	return &Handler{service: service, consent: consentChecker, audit: auditLogger, tm: tm}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(platform.RequireAuth(h.tm))

	r.Group(func(create chi.Router) {
		create.Use(platform.RequireRoles(platform.RolePhysician, platform.RoleHospitalAdmin))
		create.Post("/patients", h.createPatient)
	})

	r.Group(func(patient chi.Router) {
		patient.Use(consent.Middleware(h.consent, h.audit, audit.ActionViewPatient, h.resolvePatientRoute))
		patient.Get("/patients/{patientID}", h.getPatient)
		patient.Get("/patients/{patientID}/encounters", h.listEncounters)
	})

	r.Group(func(w chi.Router) {
		w.Use(platform.RequireRoles(platform.RolePhysician, platform.RoleHospitalAdmin))
		w.Use(consent.Middleware(h.consent, h.audit, audit.ActionCreateEncounter, h.resolvePatientRoute))
		w.Post("/patients/{patientID}/encounters", h.createEncounter)
	})

	r.Group(func(enc chi.Router) {
		enc.Use(consent.Middleware(h.consent, h.audit, audit.ActionViewEncounter, h.resolveEncounterRoute))
		enc.Get("/encounters/{encounterID}", h.getEncounter)
		enc.Get("/encounters/{encounterID}/observations", h.listObservations)
	})

	r.Group(func(enc chi.Router) {
		enc.Use(platform.RequireRoles(platform.RolePhysician, platform.RoleHospitalAdmin))
		enc.Use(consent.Middleware(h.consent, h.audit, audit.ActionCreateEncounter, h.resolveEncounterRoute))
		enc.Post("/encounters/{encounterID}/observations", h.createObservation)
	})

	return r
}

func (h *Handler) resolvePatientRoute(r *http.Request) (consent.ResourceInfo, error) {
	patientID, err := uuid.Parse(chi.URLParam(r, "patientID"))
	if err != nil {
		return consent.ResourceInfo{}, err
	}

	patient, err := h.service.GetPatient(r.Context(), patientID)
	if err != nil {
		return consent.ResourceInfo{}, err
	}

	return consent.ResourceInfo{
		PatientID: patientID, PatientUserID: patient.UserID,
		ResourceType: "patient", ResourceID: &patientID,
	}, nil
}

func (h *Handler) resolveEncounterRoute(r *http.Request) (consent.ResourceInfo, error) {
	encounterID, err := uuid.Parse(chi.URLParam(r, "encounterID"))
	if err != nil {
		return consent.ResourceInfo{}, err
	}

	encounter, err := h.service.GetEncounter(r.Context(), encounterID)
	if err != nil {
		return consent.ResourceInfo{}, err
	}

	patient, err := h.service.GetPatient(r.Context(), encounter.PatientID)
	if err != nil {
		return consent.ResourceInfo{}, err
	}

	return consent.ResourceInfo{
		PatientID: encounter.PatientID, PatientUserID: patient.UserID,
		ResourceType: "encounter", ResourceID: &encounterID,
	}, nil
}

type createPatientRequest struct {
	FullName    string  `json:"full_name"`
	DateOfBirth *string `json:"date_of_birth"`
	Sex         *string `json:"sex"`
	NationalID  *string `json:"national_id"`
	Phone       *string `json:"phone"`
	Address     *string `json:"address"`
	NextOfKin   *string `json:"next_of_kin"`
}

type patientResponse struct {
	ID          string  `json:"id"`
	FullName    string  `json:"full_name"`
	DateOfBirth *string `json:"date_of_birth,omitempty"`
	Sex         *string `json:"sex,omitempty"`
	NationalID  *string `json:"national_id,omitempty"`
	Phone       *string `json:"phone,omitempty"`
	Address     *string `json:"address,omitempty"`
	NextOfKin   *string `json:"next_of_kin,omitempty"`
	Version     int32   `json:"version"`
}

func toPatientResponse(p Patient) patientResponse {
	var dob *string
	if p.DateOfBirth != nil {
		s := p.DateOfBirth.Format("2006-01-02")
		dob = &s
	}
	return patientResponse{
		ID: p.ID.String(), FullName: p.FullName, DateOfBirth: dob, Sex: p.Sex,
		NationalID: p.NationalID, Phone: p.Phone, Address: p.Address, NextOfKin: p.NextOfKin, Version: p.Version,
	}
}

func (h *Handler) createPatient(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok || claims.FacilityID == nil {
		platform.WriteError(w, http.StatusForbidden, "actor has no facility affiliation")
		return
	}

	var req createPatientRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var dob *time.Time
	if req.DateOfBirth != nil {
		parsed, err := time.Parse("2006-01-02", *req.DateOfBirth)
		if err != nil {
			platform.WriteError(w, http.StatusBadRequest, "invalid date_of_birth, expected YYYY-MM-DD")
			return
		}
		dob = &parsed
	}

	patient, err := h.service.CreatePatient(r.Context(), claims.UserID, *claims.FacilityID, CreatePatientInput{
		FullName: req.FullName, DateOfBirth: dob, Sex: req.Sex, NationalID: req.NationalID,
		Phone: req.Phone, Address: req.Address, NextOfKin: req.NextOfKin,
	})
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.audit.Record(r.Context(), audit.RecordInput{
		ActorUserID: claims.UserID, ActorRole: claims.Role, Action: audit.ActionCreatePatient,
		ResourceType: "patient", ResourceID: &patient.ID, PatientID: &patient.ID, Result: audit.ResultAllowed,
	})

	platform.WriteJSON(w, http.StatusCreated, toPatientResponse(patient))
}

func (h *Handler) getPatient(w http.ResponseWriter, r *http.Request) {
	patientID, _ := uuid.Parse(chi.URLParam(r, "patientID"))
	patient, err := h.service.GetPatient(r.Context(), patientID)
	if err != nil {
		if errors.Is(err, ErrPatientNotFound) {
			platform.WriteError(w, http.StatusNotFound, "patient not found")
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "failed to load patient")
		return
	}
	platform.WriteJSON(w, http.StatusOK, toPatientResponse(patient))
}

type encounterRequest struct {
	Type       EncounterType `json:"type"`
	Notes      *string       `json:"notes"`
	OccurredAt *string       `json:"occurred_at"`
}

type encounterResponse struct {
	ID         string        `json:"id"`
	PatientID  string        `json:"patient_id"`
	FacilityID string        `json:"facility_id"`
	ProviderID string        `json:"provider_id"`
	Type       EncounterType `json:"type"`
	Notes      *string       `json:"notes,omitempty"`
	OccurredAt string        `json:"occurred_at"`
	Version    int32         `json:"version"`
}

func toEncounterResponse(e Encounter) encounterResponse {
	return encounterResponse{
		ID: e.ID.String(), PatientID: e.PatientID.String(), FacilityID: e.FacilityID.String(),
		ProviderID: e.ProviderID.String(), Type: e.Type, Notes: e.Notes,
		OccurredAt: e.OccurredAt.Format(time.RFC3339), Version: e.Version,
	}
}

func (h *Handler) createEncounter(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok || claims.FacilityID == nil {
		platform.WriteError(w, http.StatusForbidden, "actor has no facility affiliation")
		return
	}

	patientID, _ := uuid.Parse(chi.URLParam(r, "patientID"))

	var req encounterRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	occurredAt := time.Now()
	if req.OccurredAt != nil {
		parsed, err := time.Parse(time.RFC3339, *req.OccurredAt)
		if err != nil {
			platform.WriteError(w, http.StatusBadRequest, "invalid occurred_at, expected RFC3339")
			return
		}
		occurredAt = parsed
	}

	encounter, err := h.service.CreateEncounter(r.Context(), claims.UserID, *claims.FacilityID, CreateEncounterInput{
		PatientID: patientID, Type: req.Type, Notes: req.Notes, OccurredAt: occurredAt,
	})
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	platform.WriteJSON(w, http.StatusCreated, toEncounterResponse(encounter))
}

func (h *Handler) getEncounter(w http.ResponseWriter, r *http.Request) {
	encounterID, _ := uuid.Parse(chi.URLParam(r, "encounterID"))
	encounter, err := h.service.GetEncounter(r.Context(), encounterID)
	if err != nil {
		if errors.Is(err, ErrEncounterNotFound) {
			platform.WriteError(w, http.StatusNotFound, "encounter not found")
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "failed to load encounter")
		return
	}
	platform.WriteJSON(w, http.StatusOK, toEncounterResponse(encounter))
}

func (h *Handler) listEncounters(w http.ResponseWriter, r *http.Request) {
	patientID, _ := uuid.Parse(chi.URLParam(r, "patientID"))
	encounters, err := h.service.ListEncounters(r.Context(), patientID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to list encounters")
		return
	}

	out := make([]encounterResponse, len(encounters))
	for i, e := range encounters {
		out[i] = toEncounterResponse(e)
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

type observationRequest struct {
	Type    ObservationType `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type observationResponse struct {
	ID          string          `json:"id"`
	EncounterID string          `json:"encounter_id"`
	Type        ObservationType `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	RecordedAt  string          `json:"recorded_at"`
}

func toObservationResponse(o ClinicalObservation) observationResponse {
	return observationResponse{
		ID: o.ID.String(), EncounterID: o.EncounterID.String(), Type: o.Type,
		Payload: o.Payload, RecordedAt: o.RecordedAt.Format(time.RFC3339),
	}
}

func (h *Handler) createObservation(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	encounterID, _ := uuid.Parse(chi.URLParam(r, "encounterID"))

	var req observationRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	observation, err := h.service.CreateObservation(r.Context(), claims.UserID, encounterID, req.Type, req.Payload)
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	platform.WriteJSON(w, http.StatusCreated, toObservationResponse(observation))
}

func (h *Handler) listObservations(w http.ResponseWriter, r *http.Request) {
	encounterID, _ := uuid.Parse(chi.URLParam(r, "encounterID"))
	observations, err := h.service.ListObservations(r.Context(), encounterID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to list observations")
		return
	}

	out := make([]observationResponse, len(observations))
	for i, o := range observations {
		out[i] = toObservationResponse(o)
	}
	platform.WriteJSON(w, http.StatusOK, out)
}
