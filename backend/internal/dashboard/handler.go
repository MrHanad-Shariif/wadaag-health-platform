package dashboard

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/platform"
)

// ProviderResolver maps the calling user to their provider identity — same
// interface shape as referrals.ProviderResolver / consults.ProviderResolver.
// Used only to resolve a physician caller's own provider id ahead of
// Service.GetSummary; satisfied by *facilities.Service, wired in main.go.
type ProviderResolver interface {
	ResolveProviderID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

type Handler struct {
	service   *Service
	tm        *platform.TokenManager
	providers ProviderResolver
}

func NewHandler(service *Service, tm *platform.TokenManager, providers ProviderResolver) *Handler {
	return &Handler{service: service, tm: tm, providers: providers}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(platform.RequireAuth(h.tm))
	r.Use(platform.RequirePermission("dashboard", "read"))
	r.Get("/summary", h.summary)
	return r
}

type doctorActivityResponse struct {
	ProviderID    string  `json:"provider_id"`
	FullName      *string `json:"full_name,omitempty"`
	ReferralCount int64   `json:"referral_count"`
}

type summaryResponse struct {
	TotalPatients     int64                    `json:"total_patients"`
	TotalEncounters   int64                    `json:"total_encounters"`
	TotalFacilities   int64                    `json:"total_facilities"`
	TotalUsers        int64                    `json:"total_users"`
	ReferralsByStatus map[string]int64         `json:"referrals_by_status"`
	MostActiveDoctors []doctorActivityResponse `json:"most_active_doctors"`

	// The following three are only meaningful (and only ever non-zero) for
	// a physician caller — see Service.GetSummary. Left as plain zero-value
	// ints (not pointers) for every other role: a dashboard consumer that
	// isn't a physician simply has no such concept, and the frontend is
	// expected to only render this panel for the physician role, same as
	// it already branches per-role for the rest of the dashboard.
	PendingConsultCount     int64 `json:"pending_consult_count"`
	MyReferralsPendingCount int64 `json:"my_referrals_pending_count"`
	PatientsTodayCount      int64 `json:"patients_today_count"`
}

func toSummaryResponse(s Summary) summaryResponse {
	doctors := make([]doctorActivityResponse, len(s.MostActiveDoctors))
	for i, d := range s.MostActiveDoctors {
		doctors[i] = doctorActivityResponse{ProviderID: d.ProviderID.String(), FullName: d.FullName, ReferralCount: d.ReferralCount}
	}
	return summaryResponse{
		TotalPatients: s.TotalPatients, TotalEncounters: s.TotalEncounters,
		TotalFacilities: s.TotalFacilities, TotalUsers: s.TotalUsers,
		ReferralsByStatus: s.ReferralsByStatus, MostActiveDoctors: doctors,
		PendingConsultCount: s.PendingConsultCount, MyReferralsPendingCount: s.MyReferralsPendingCount,
		PatientsTodayCount: s.PatientsTodayCount,
	}
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	// Only a physician caller needs their own provider id resolved — every
	// other role (including system_admin, who has no provider profile of
	// their own) gets the admin-only fields with actorProviderID left nil.
	// A resolution failure (e.g. no providers row for this user, which
	// shouldn't normally happen for a physician) is tolerated rather than
	// failing the whole summary — the physician-specific fields are simply
	// left at zero.
	var actorProviderID *uuid.UUID
	if claims.Role == platform.RolePhysician {
		if id, err := h.providers.ResolveProviderID(r.Context(), claims.UserID); err == nil {
			actorProviderID = &id
		}
	}

	summary, err := h.service.GetSummary(r.Context(), claims.UserID, claims.Role, actorProviderID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to load dashboard summary")
		return
	}

	platform.WriteJSON(w, http.StatusOK, toSummaryResponse(summary))
}
