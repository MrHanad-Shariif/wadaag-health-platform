package consults

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/audit"
	"github.com/wadaag/health-platform/backend/internal/facilities"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"github.com/wadaag/health-platform/backend/internal/records"
)

type Handler struct {
	service *Service
	audit   *audit.Logger
	tm      *platform.TokenManager
}

func NewHandler(service *Service, auditLogger *audit.Logger, tm *platform.TokenManager) *Handler {
	return &Handler{service: service, audit: auditLogger, tm: tm}
}

// Routes is mounted at "/consults" (see server.NewRouter), so the final
// paths are:
//
//	POST /api/v1/consults
//	GET  /api/v1/consults/inbox
//	GET  /api/v1/consults/{consultationID}
//	GET  /api/v1/consults/{consultationID}/messages
//	POST /api/v1/consults/{consultationID}/messages
//	POST /api/v1/consults/{consultationID}/close
//
// Unlike referrals/attachments, no consent.Middleware is used anywhere in
// this tree — a consult never grants patient-record access via the consent
// module, so authorization is the simpler "requesting/target provider or
// system_admin" party check performed inline in each handler below (see
// Service.IsParty), with audit.Logger.Record called directly on every
// view/create/message/close, allowed or denied.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(platform.RequireAuth(h.tm))

	r.Get("/inbox", h.inbox)

	r.Group(func(create chi.Router) {
		create.Use(platform.RequireRoles(platform.RolePhysician, platform.RoleHospitalAdmin))
		create.Post("/", h.create)
	})

	r.Get("/{consultationID}", h.get)
	r.Get("/{consultationID}/messages", h.listMessages)
	r.Post("/{consultationID}/messages", h.postMessage)
	r.Post("/{consultationID}/close", h.close)

	return r
}

type consultationResponse struct {
	ID                   string  `json:"id"`
	PatientID            string  `json:"patient_id"`
	RequestingProviderID string  `json:"requesting_provider_id"`
	TargetProviderID     string  `json:"target_provider_id"`
	Reason               string  `json:"reason"`
	Status               string  `json:"status"`
	Direction            *string `json:"direction,omitempty"` // "incoming"|"outgoing", only set in inbox list responses
	CreatedAt            string  `json:"created_at"`
	UpdatedAt            string  `json:"updated_at"`
}

func toConsultationResponse(c Consultation) consultationResponse {
	return consultationResponse{
		ID: c.ID.String(), PatientID: c.PatientID.String(), RequestingProviderID: c.RequestingProviderID.String(),
		TargetProviderID: c.TargetProviderID.String(), Reason: c.Reason, Status: string(c.Status),
		CreatedAt: c.CreatedAt.Format(time.RFC3339), UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
	}
}

type consultMessageResponse struct {
	ID             string   `json:"id"`
	ConsultationID string   `json:"consultation_id"`
	SenderID       string   `json:"sender_id"`
	Body           string   `json:"body"`
	AttachmentIDs  []string `json:"attachment_ids"`
	CreatedAt      string   `json:"created_at"`
}

func toMessageResponse(m Message) consultMessageResponse {
	attachmentIDs := make([]string, len(m.AttachmentIDs))
	for i, id := range m.AttachmentIDs {
		attachmentIDs[i] = id.String()
	}
	return consultMessageResponse{
		ID: m.ID.String(), ConsultationID: m.ConsultationID.String(), SenderID: m.SenderID.String(),
		Body: m.Body, AttachmentIDs: attachmentIDs, CreatedAt: m.CreatedAt.Format(time.RFC3339),
	}
}

type createRequest struct {
	PatientID        string `json:"patient_id"`
	TargetProviderID string `json:"target_provider_id"`
	Reason           string `json:"reason"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	var req createRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Reason == "" {
		platform.WriteError(w, http.StatusBadRequest, "reason is required")
		return
	}
	patientID, err := uuid.Parse(req.PatientID)
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid patient_id")
		return
	}
	targetProviderID, err := uuid.Parse(req.TargetProviderID)
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid target_provider_id")
		return
	}

	consultation, err := h.service.Create(r.Context(), claims.UserID, CreateInput{
		PatientID: patientID, TargetProviderID: targetProviderID, Reason: req.Reason,
	})
	result := audit.ResultAllowed
	if err != nil {
		result = audit.ResultDenied
	}
	h.audit.Record(r.Context(), audit.RecordInput{
		ActorUserID: claims.UserID, ActorRole: claims.Role, Action: audit.ActionCreateConsult,
		ResourceType: "consultation", PatientID: &patientID, Result: result,
	})
	if err != nil {
		switch {
		case errors.Is(err, records.ErrPatientNotFound):
			platform.WriteError(w, http.StatusNotFound, "patient not found")
		case errors.Is(err, ErrTargetProviderInvalid), errors.Is(err, facilities.ErrProviderNotFound):
			platform.WriteError(w, http.StatusBadRequest, err.Error())
		default:
			platform.WriteError(w, http.StatusInternalServerError, "failed to create consultation")
		}
		return
	}

	platform.WriteJSON(w, http.StatusCreated, toConsultationResponse(consultation))
}

func (h *Handler) inbox(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	// system_admin has no provider identity of its own to scope by — give
	// it the platform-wide oversight list instead of 403ing. "direction" is
	// caller-relative (incoming/outgoing from one provider's perspective),
	// so it's left unset on every item in this list rather than computed.
	var list []Consultation
	var providerID uuid.UUID
	var err error
	isAdmin := claims.Role == platform.RoleSystemAdmin
	if isAdmin {
		list, err = h.service.ListAll(r.Context())
	} else {
		providerID, err = h.service.ResolveProviderID(r.Context(), claims.UserID)
		if err != nil {
			platform.WriteError(w, http.StatusForbidden, "actor has no provider profile")
			return
		}
		list, err = h.service.ListForProvider(r.Context(), providerID)
	}
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to list consultations")
		return
	}

	// One audit entry for the list access as a whole — patient_id is
	// omitted since this spans multiple patients, not a single-patient
	// access.
	h.audit.Record(r.Context(), audit.RecordInput{
		ActorUserID: claims.UserID, ActorRole: claims.Role, Action: audit.ActionViewConsult,
		ResourceType: "consultation", Result: audit.ResultAllowed,
	})

	out := make([]consultationResponse, len(list))
	for i, c := range list {
		resp := toConsultationResponse(c)
		if !isAdmin {
			direction := "outgoing"
			if c.TargetProviderID == providerID {
				direction = "incoming"
			}
			resp.Direction = &direction
		}
		out[i] = resp
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

// loadParty resolves consultationID from the URL, loads the consultation,
// verifies the actor is a party to it (or system_admin), and records an
// audit entry for action either way — mirroring consent.Middleware's
// pattern of logging both allowed and denied outcomes, just without going
// through the generic consent module (see the package/Routes doc comment
// for why).
func (h *Handler) loadParty(w http.ResponseWriter, r *http.Request, action audit.Action) (Consultation, bool) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return Consultation{}, false
	}

	consultationID, err := uuid.Parse(chi.URLParam(r, "consultationID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid consultation id")
		return Consultation{}, false
	}

	consultation, err := h.service.Get(r.Context(), consultationID)
	if err != nil {
		if errors.Is(err, ErrConsultationNotFound) {
			platform.WriteError(w, http.StatusNotFound, "consultation not found")
			return Consultation{}, false
		}
		platform.WriteError(w, http.StatusInternalServerError, "failed to load consultation")
		return Consultation{}, false
	}

	allowed, err := h.service.IsParty(r.Context(), consultation, claims.UserID, claims.Role)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to verify access")
		return Consultation{}, false
	}

	result := audit.ResultDenied
	if allowed {
		result = audit.ResultAllowed
	}
	h.audit.Record(r.Context(), audit.RecordInput{
		ActorUserID: claims.UserID, ActorRole: claims.Role, Action: action,
		ResourceType: "consultation", ResourceID: &consultation.ID, PatientID: &consultation.PatientID, Result: result,
	})

	if !allowed {
		platform.WriteError(w, http.StatusForbidden, ErrNotConsultParty.Error())
		return Consultation{}, false
	}
	return consultation, true
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	consultation, ok := h.loadParty(w, r, audit.ActionViewConsult)
	if !ok {
		return
	}
	platform.WriteJSON(w, http.StatusOK, toConsultationResponse(consultation))
}

func (h *Handler) listMessages(w http.ResponseWriter, r *http.Request) {
	consultation, ok := h.loadParty(w, r, audit.ActionViewConsult)
	if !ok {
		return
	}
	list, err := h.service.ListMessages(r.Context(), consultation.ID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to list consultation messages")
		return
	}
	out := make([]consultMessageResponse, len(list))
	for i, m := range list {
		out[i] = toMessageResponse(m)
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

type postMessageRequest struct {
	Body          string   `json:"body"`
	AttachmentIDs []string `json:"attachment_ids"`
}

func (h *Handler) postMessage(w http.ResponseWriter, r *http.Request) {
	claims, _ := platform.ClaimsFromContext(r.Context())
	consultation, ok := h.loadParty(w, r, audit.ActionCreateConsult)
	if !ok {
		return
	}

	var req postMessageRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Body == "" {
		platform.WriteError(w, http.StatusBadRequest, "body is required")
		return
	}

	attachmentIDs := make([]uuid.UUID, len(req.AttachmentIDs))
	for i, s := range req.AttachmentIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			platform.WriteError(w, http.StatusBadRequest, "invalid attachment_ids entry")
			return
		}
		attachmentIDs[i] = id
	}

	message, err := h.service.PostMessage(r.Context(), claims.UserID, consultation.ID, req.Body, attachmentIDs)
	if err != nil {
		switch {
		case errors.Is(err, ErrConsultClosed):
			platform.WriteError(w, http.StatusConflict, err.Error())
		case errors.Is(err, ErrConsultationNotFound):
			platform.WriteError(w, http.StatusNotFound, err.Error())
		default:
			platform.WriteError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	platform.WriteJSON(w, http.StatusCreated, toMessageResponse(message))
}

func (h *Handler) close(w http.ResponseWriter, r *http.Request) {
	consultation, ok := h.loadParty(w, r, audit.ActionViewConsult)
	if !ok {
		return
	}
	closed, err := h.service.Close(r.Context(), consultation.ID)
	if err != nil {
		if errors.Is(err, ErrConsultationNotFound) {
			platform.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "failed to close consultation")
		return
	}
	platform.WriteJSON(w, http.StatusOK, toConsultationResponse(closed))
}
