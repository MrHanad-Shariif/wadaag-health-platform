package audit

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/platform"
)

// PatientIdentityResolver answers "is this user the patient behind this
// patient_id" — implemented by records.Service and wired in main.go, so
// this module can gate the audit-trail view without importing records.
type PatientIdentityResolver interface {
	IsPatientUser(ctx context.Context, patientID, userID uuid.UUID) (bool, error)
}

// Handler exposes the "who accessed my data" view — patients can see every
// access to their own record; nobody else can query another patient's log
// through this route (system_admin access is still recorded here like any
// other actor, but reading someone else's audit trail isn't exposed).
type Handler struct {
	logger   *Logger
	tm       *platform.TokenManager
	patients PatientIdentityResolver
}

func NewHandler(logger *Logger, tm *platform.TokenManager, patients PatientIdentityResolver) *Handler {
	return &Handler{logger: logger, tm: tm, patients: patients}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(platform.RequireAuth(h.tm))
	r.Get("/patients/{patientID}", h.listForPatient)
	return r
}

type entryResponse struct {
	ID           string  `json:"id"`
	ActorUserID  string  `json:"actor_user_id"`
	ActorRole    string  `json:"actor_role"`
	Action       string  `json:"action"`
	ResourceType string  `json:"resource_type"`
	ResourceID   *string `json:"resource_id,omitempty"`
	Result       string  `json:"result"`
	OccurredAt   string  `json:"occurred_at"`
}

func (h *Handler) listForPatient(w http.ResponseWriter, r *http.Request) {
	patientID, err := uuid.Parse(chi.URLParam(r, "patientID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid patient id")
		return
	}

	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	if claims.Role != platform.RoleSystemAdmin {
		isSelf, err := h.patients.IsPatientUser(r.Context(), patientID, claims.UserID)
		if err != nil {
			platform.WriteError(w, http.StatusInternalServerError, "failed to verify patient identity")
			return
		}
		if !isSelf {
			platform.WriteError(w, http.StatusForbidden, "cannot view another patient's audit trail")
			return
		}
	}

	entries, err := h.logger.ListForPatient(r.Context(), patientID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to load audit trail")
		return
	}

	out := make([]entryResponse, len(entries))
	for i, e := range entries {
		var resourceID *string
		if e.ResourceID != nil {
			s := e.ResourceID.String()
			resourceID = &s
		}
		out[i] = entryResponse{
			ID: e.ID.String(), ActorUserID: e.ActorUserID.String(), ActorRole: string(e.ActorRole),
			Action: string(e.Action), ResourceType: e.ResourceType, ResourceID: resourceID,
			Result: string(e.Result), OccurredAt: e.OccurredAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	platform.WriteJSON(w, http.StatusOK, out)
}
