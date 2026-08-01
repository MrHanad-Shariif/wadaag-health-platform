package dashboard

import (
	"net/http"

	"github.com/go-chi/chi/v5"
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
	r.Use(platform.RequirePermission("dashboard", "read"))
	r.Get("/summary", h.summary)
	return r
}

type summaryResponse struct {
	TotalPatients     int64            `json:"total_patients"`
	TotalEncounters   int64            `json:"total_encounters"`
	TotalFacilities   int64            `json:"total_facilities"`
	TotalUsers        int64            `json:"total_users"`
	ReferralsByStatus map[string]int64 `json:"referrals_by_status"`
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.service.GetSummary(r.Context())
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to load dashboard summary")
		return
	}

	platform.WriteJSON(w, http.StatusOK, summaryResponse{
		TotalPatients: summary.TotalPatients, TotalEncounters: summary.TotalEncounters,
		TotalFacilities: summary.TotalFacilities, TotalUsers: summary.TotalUsers,
		ReferralsByStatus: summary.ReferralsByStatus,
	})
}
