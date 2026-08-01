package referrals

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/audit"
	"github.com/wadaag/health-platform/backend/internal/consent"
	"github.com/wadaag/health-platform/backend/internal/platform"
)

// PatientLookup resolves a patient's linked user_id (if any) — needed to
// evaluate the "patient viewing their own record" branch of a consent
// check. Implemented by records.Service, wired in main.go.
type PatientLookup interface {
	GetPatientUserID(ctx context.Context, patientID uuid.UUID) (*uuid.UUID, error)
}

type Handler struct {
	service  *Service
	consent  *consent.Checker
	audit    *audit.Logger
	patients PatientLookup
	tm       *platform.TokenManager
}

func NewHandler(service *Service, consentChecker *consent.Checker, auditLogger *audit.Logger, patients PatientLookup, tm *platform.TokenManager) *Handler {
	return &Handler{service: service, consent: consentChecker, audit: auditLogger, patients: patients, tm: tm}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(platform.RequireAuth(h.tm))

	r.Get("/inbox", h.inbox)

	r.Group(func(patient chi.Router) {
		patient.Use(consent.Middleware(h.consent, h.audit, audit.ActionViewPatient, h.resolvePatientRoute))
		patient.Get("/patients/{patientID}", h.listForPatient)
	})

	r.Group(func(create chi.Router) {
		create.Use(platform.RequireRoles(platform.RolePhysician, platform.RoleHospitalAdmin))
		create.Post("/", h.create)
	})

	r.Group(func(ref chi.Router) {
		ref.Use(consent.Middleware(h.consent, h.audit, audit.ActionViewPatient, h.resolveReferralRoute))
		ref.Get("/{referralID}", h.get)
		ref.Get("/{referralID}/events", h.events)
	})

	r.Group(func(action chi.Router) {
		action.Use(platform.RequireRoles(platform.RolePhysician, platform.RoleHospitalAdmin))
		action.Post("/{referralID}/accept", h.accept)
		action.Post("/{referralID}/decline", h.decline)
		action.Post("/{referralID}/start", h.start)
		action.Post("/{referralID}/complete", h.complete)
		action.Post("/{referralID}/cancel", h.cancel)
	})

	return r
}

func (h *Handler) resolvePatientRoute(r *http.Request) (consent.ResourceInfo, error) {
	patientID, err := uuid.Parse(chi.URLParam(r, "patientID"))
	if err != nil {
		return consent.ResourceInfo{}, err
	}
	patientUserID, err := h.patients.GetPatientUserID(r.Context(), patientID)
	if err != nil {
		return consent.ResourceInfo{}, err
	}
	return consent.ResourceInfo{PatientID: patientID, PatientUserID: patientUserID, ResourceType: "patient", ResourceID: &patientID}, nil
}

func (h *Handler) resolveReferralRoute(r *http.Request) (consent.ResourceInfo, error) {
	referralID, err := uuid.Parse(chi.URLParam(r, "referralID"))
	if err != nil {
		return consent.ResourceInfo{}, err
	}
	referral, err := h.service.Get(r.Context(), referralID)
	if err != nil {
		return consent.ResourceInfo{}, err
	}
	patientUserID, err := h.patients.GetPatientUserID(r.Context(), referral.PatientID)
	if err != nil {
		return consent.ResourceInfo{}, err
	}
	return consent.ResourceInfo{PatientID: referral.PatientID, PatientUserID: patientUserID, ResourceType: "referral", ResourceID: &referralID}, nil
}

type createRequest struct {
	PatientID                  string  `json:"patient_id"`
	ReceivingFacilityID        string  `json:"receiving_facility_id"`
	SpecialtyRequested         string  `json:"specialty_requested"`
	Urgency                    Urgency `json:"urgency"`
	Reason                     string  `json:"reason"`
	ClinicalSummaryEncounterID *string `json:"clinical_summary_encounter_id"`
}

type referralResponse struct {
	ID                  string  `json:"id"`
	PatientID           string  `json:"patient_id"`
	ReferringProviderID string  `json:"referring_provider_id"`
	ReferringFacilityID string  `json:"referring_facility_id"`
	ReceivingFacilityID *string `json:"receiving_facility_id,omitempty"`
	ReceivingProviderID *string `json:"receiving_provider_id,omitempty"`
	SpecialtyRequested  string  `json:"specialty_requested"`
	Urgency             Urgency `json:"urgency"`
	Status              Status  `json:"status"`
	Reason              string  `json:"reason"`
	Version             int32   `json:"version"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
}

func toReferralResponse(ref Referral) referralResponse {
	var recvFacility, recvProvider *string
	if ref.ReceivingFacilityID != nil {
		s := ref.ReceivingFacilityID.String()
		recvFacility = &s
	}
	if ref.ReceivingProviderID != nil {
		s := ref.ReceivingProviderID.String()
		recvProvider = &s
	}
	return referralResponse{
		ID: ref.ID.String(), PatientID: ref.PatientID.String(), ReferringProviderID: ref.ReferringProviderID.String(),
		ReferringFacilityID: ref.ReferringFacilityID.String(), ReceivingFacilityID: recvFacility, ReceivingProviderID: recvProvider,
		SpecialtyRequested: ref.SpecialtyRequested, Urgency: ref.Urgency, Status: ref.Status, Reason: ref.Reason,
		Version: ref.Version, CreatedAt: ref.CreatedAt.Format(time.RFC3339), UpdatedAt: ref.UpdatedAt.Format(time.RFC3339),
	}
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok || claims.FacilityID == nil {
		platform.WriteError(w, http.StatusForbidden, "actor has no facility affiliation")
		return
	}

	var req createRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	patientID, err := uuid.Parse(req.PatientID)
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid patient_id")
		return
	}
	receivingFacilityID, err := uuid.Parse(req.ReceivingFacilityID)
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid receiving_facility_id")
		return
	}

	var summaryEncounterID *uuid.UUID
	if req.ClinicalSummaryEncounterID != nil {
		id, err := uuid.Parse(*req.ClinicalSummaryEncounterID)
		if err != nil {
			platform.WriteError(w, http.StatusBadRequest, "invalid clinical_summary_encounter_id")
			return
		}
		summaryEncounterID = &id
	}

	// The referring provider must already have access to this patient
	// (their own facility's patient, or a prior referral grant) before
	// they can refer them onward — same rule as reading the record, just
	// checked inline here since patient_id lives in the body, not the URL.
	patientUserID, err := h.patients.GetPatientUserID(r.Context(), patientID)
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "patient not found")
		return
	}
	allowed, err := h.consent.HasAccess(r.Context(), claims.UserID, claims.Role, claims.FacilityID, patientUserID, patientID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "consent check failed")
		return
	}
	result := audit.ResultDenied
	if allowed {
		result = audit.ResultAllowed
	}
	h.audit.Record(r.Context(), audit.RecordInput{
		ActorUserID: claims.UserID, ActorRole: claims.Role, Action: audit.ActionCreateReferral,
		ResourceType: "patient", ResourceID: &patientID, PatientID: &patientID, Result: result,
	})
	if !allowed {
		platform.WriteError(w, http.StatusForbidden, "not authorized to refer this patient")
		return
	}

	referral, err := h.service.Create(r.Context(), claims.UserID, *claims.FacilityID, CreateReferralInput{
		PatientID: patientID, ReceivingFacilityID: receivingFacilityID, SpecialtyRequested: req.SpecialtyRequested,
		Urgency: req.Urgency, Reason: req.Reason, ClinicalSummaryEncounterID: summaryEncounterID,
	})
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	platform.WriteJSON(w, http.StatusCreated, toReferralResponse(referral))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	referralID, _ := uuid.Parse(chi.URLParam(r, "referralID"))
	referral, err := h.service.Get(r.Context(), referralID)
	if err != nil {
		if errors.Is(err, ErrReferralNotFound) {
			platform.WriteError(w, http.StatusNotFound, "referral not found")
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "failed to load referral")
		return
	}
	platform.WriteJSON(w, http.StatusOK, toReferralResponse(referral))
}

func (h *Handler) listForPatient(w http.ResponseWriter, r *http.Request) {
	patientID, _ := uuid.Parse(chi.URLParam(r, "patientID"))
	list, err := h.service.ListForPatient(r.Context(), patientID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to list referrals")
		return
	}
	out := make([]referralResponse, len(list))
	for i, ref := range list {
		out[i] = toReferralResponse(ref)
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) inbox(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok || claims.FacilityID == nil {
		platform.WriteError(w, http.StatusForbidden, "actor has no facility affiliation")
		return
	}

	list, err := h.service.ListForFacility(r.Context(), *claims.FacilityID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to list referrals")
		return
	}
	out := make([]referralResponse, len(list))
	for i, ref := range list {
		out[i] = toReferralResponse(ref)
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) events(w http.ResponseWriter, r *http.Request) {
	referralID, _ := uuid.Parse(chi.URLParam(r, "referralID"))
	events, err := h.service.ListStatusEvents(r.Context(), referralID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to list referral events")
		return
	}

	type eventResponse struct {
		FromStatus *Status `json:"from_status,omitempty"`
		ToStatus   Status  `json:"to_status"`
		ActorID    string  `json:"actor_user_id"`
		Note       *string `json:"note,omitempty"`
		OccurredAt string  `json:"occurred_at"`
	}
	out := make([]eventResponse, len(events))
	for i, e := range events {
		out[i] = eventResponse{FromStatus: e.FromStatus, ToStatus: e.ToStatus, ActorID: e.ActorUserID.String(), Note: e.Note, OccurredAt: e.OccurredAt.Format(time.RFC3339)}
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

type actionRequest struct {
	Version int32   `json:"version"`
	Note    *string `json:"note"`
}

func (h *Handler) actorContext(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok || claims.FacilityID == nil {
		platform.WriteError(w, http.StatusForbidden, "actor has no facility affiliation")
		return uuid.UUID{}, uuid.UUID{}, false
	}
	return claims.UserID, *claims.FacilityID, true
}

func (h *Handler) writeTransitionResult(w http.ResponseWriter, referral Referral, err error) {
	if err != nil {
		switch {
		case errors.Is(err, ErrNotReceivingFacility), errors.Is(err, ErrNotReferralParty):
			platform.WriteError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, ErrInvalidTransition), errors.Is(err, ErrStaleVersion):
			platform.WriteError(w, http.StatusConflict, err.Error())
		case errors.Is(err, ErrReferralNotFound):
			platform.WriteError(w, http.StatusNotFound, err.Error())
		default:
			platform.WriteError(w, http.StatusInternalServerError, "failed to update referral")
		}
		return
	}
	platform.WriteJSON(w, http.StatusOK, toReferralResponse(referral))
}

func (h *Handler) accept(w http.ResponseWriter, r *http.Request) {
	referralID, _ := uuid.Parse(chi.URLParam(r, "referralID"))
	userID, facilityID, ok := h.actorContext(w, r)
	if !ok {
		return
	}
	var req actionRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	referral, err := h.service.Accept(r.Context(), userID, facilityID, referralID, req.Version)
	h.writeTransitionResult(w, referral, err)
}

func (h *Handler) decline(w http.ResponseWriter, r *http.Request) {
	h.runTransition(w, r, Transition{To: StatusDeclined, AllowedFrom: StatusRouted, RequireFacility: ReceivingFacility})
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	h.runTransition(w, r, Transition{To: StatusInProgress, AllowedFrom: StatusAccepted, RequireFacility: ReceivingFacility})
}

func (h *Handler) complete(w http.ResponseWriter, r *http.Request) {
	h.runTransition(w, r, Transition{To: StatusCompleted, AllowedFrom: StatusInProgress, RequireFacility: ReceivingFacility})
}

func (h *Handler) runTransition(w http.ResponseWriter, r *http.Request, t Transition) {
	referralID, _ := uuid.Parse(chi.URLParam(r, "referralID"))
	userID, facilityID, ok := h.actorContext(w, r)
	if !ok {
		return
	}
	var req actionRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	referral, err := h.service.Transition(r.Context(), userID, facilityID, referralID, req.Version, t, req.Note)
	h.writeTransitionResult(w, referral, err)
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	referralID, _ := uuid.Parse(chi.URLParam(r, "referralID"))
	userID, facilityID, ok := h.actorContext(w, r)
	if !ok {
		return
	}
	var req actionRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	referral, err := h.service.Cancel(r.Context(), userID, facilityID, referralID, req.Version, req.Note)
	h.writeTransitionResult(w, referral, err)
}
