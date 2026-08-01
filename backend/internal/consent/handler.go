package consent

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/platform"
)

// PatientIdentityResolver answers "is this user the patient behind this
// patient_id" — implemented by records.Service, wired in main.go, mirroring
// the same pattern used by audit.Handler.
type PatientIdentityResolver interface {
	IsPatientUser(ctx context.Context, patientID, userID uuid.UUID) (bool, error)
}

// Handler exposes patient-facing consent visibility and revocation. Grants
// are otherwise created implicitly by other modules (records, referrals) —
// a patient-initiated "grant access to a specific provider" UI is staged
// in for a later phase.
type Handler struct {
	checker  *Checker
	tm       *platform.TokenManager
	patients PatientIdentityResolver
}

func NewHandler(checker *Checker, tm *platform.TokenManager, patients PatientIdentityResolver) *Handler {
	return &Handler{checker: checker, tm: tm, patients: patients}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(platform.RequireAuth(h.tm))
	r.Get("/patients/{patientID}/grants", h.list)
	r.Delete("/patients/{patientID}/grants/{grantID}", h.revoke)
	return r
}

func (h *Handler) authorizeSelfOrAdmin(r *http.Request, patientID uuid.UUID) (bool, error) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		return false, nil
	}
	if claims.Role == platform.RoleSystemAdmin {
		return true, nil
	}
	return h.patients.IsPatientUser(r.Context(), patientID, claims.UserID)
}

type grantResponse struct {
	ID          string  `json:"id"`
	GranteeType string  `json:"grantee_type"`
	GranteeID   string  `json:"grantee_id"`
	Scope       string  `json:"scope"`
	Status      string  `json:"status"`
	GrantedVia  string  `json:"granted_via"`
	GrantedAt   string  `json:"granted_at"`
	ExpiresAt   *string `json:"expires_at,omitempty"`
}

func toGrantResponse(g Grant) grantResponse {
	var expires *string
	if g.ExpiresAt != nil {
		s := g.ExpiresAt.Format("2006-01-02T15:04:05Z07:00")
		expires = &s
	}
	return grantResponse{
		ID: g.ID.String(), GranteeType: string(g.GranteeType), GranteeID: g.GranteeID.String(),
		Scope: string(g.Scope), Status: string(g.Status), GrantedVia: string(g.GrantedVia),
		GrantedAt: g.GrantedAt.Format("2006-01-02T15:04:05Z07:00"), ExpiresAt: expires,
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	patientID, err := uuid.Parse(chi.URLParam(r, "patientID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid patient id")
		return
	}

	allowed, err := h.authorizeSelfOrAdmin(r, patientID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to verify patient identity")
		return
	}
	if !allowed {
		platform.WriteError(w, http.StatusForbidden, "cannot view another patient's consent grants")
		return
	}

	grants, err := h.checker.ListForPatient(r.Context(), patientID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to list consent grants")
		return
	}

	out := make([]grantResponse, len(grants))
	for i, g := range grants {
		out[i] = toGrantResponse(g)
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) revoke(w http.ResponseWriter, r *http.Request) {
	patientID, err := uuid.Parse(chi.URLParam(r, "patientID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid patient id")
		return
	}
	grantID, err := uuid.Parse(chi.URLParam(r, "grantID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid grant id")
		return
	}

	allowed, err := h.authorizeSelfOrAdmin(r, patientID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to verify patient identity")
		return
	}
	if !allowed {
		platform.WriteError(w, http.StatusForbidden, "cannot revoke another patient's consent grants")
		return
	}

	grant, err := h.checker.Revoke(r.Context(), grantID, patientID)
	if err != nil {
		if errors.Is(err, ErrGrantNotFound) {
			platform.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		platform.WriteError(w, http.StatusInternalServerError, "failed to revoke consent grant")
		return
	}

	platform.WriteJSON(w, http.StatusOK, toGrantResponse(grant))
}
