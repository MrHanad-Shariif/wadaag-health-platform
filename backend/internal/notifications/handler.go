package notifications

import (
	"encoding/json"
	"net/http"
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

// Routes is mounted at "/notifications" (see server.NewRouter). Every route
// here is scoped purely to the authenticated caller's own user_id — there is
// no other authorization concept in this package (nothing is patient-scoped,
// so no consent/audit wiring), matching messaging's "just RequireAuth"
// convention.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(platform.RequireAuth(h.tm))

	r.Get("/", h.list)
	r.Get("/unread-count", h.unreadCount)
	r.Post("/{notificationID}/read", h.markRead)
	r.Post("/read-all", h.markAllRead)
	r.Get("/preferences", h.listPreferences)
	r.Patch("/preferences", h.setPreference)

	return r
}

type notificationResponse struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	ReadAt    *string         `json:"read_at,omitempty"`
	CreatedAt string          `json:"created_at"`
}

func toNotificationResponse(n Notification) notificationResponse {
	resp := notificationResponse{
		ID: n.ID.String(), Type: n.Type, Payload: json.RawMessage(n.Payload),
		CreatedAt: n.CreatedAt.Format(time.RFC3339),
	}
	if n.ReadAt != nil {
		s := n.ReadAt.Format(time.RFC3339)
		resp.ReadAt = &s
	}
	return resp
}

type preferenceResponse struct {
	Channel string `json:"channel"`
	Enabled bool   `json:"enabled"`
}

func toPreferenceResponse(p Preference) preferenceResponse {
	return preferenceResponse{Channel: p.Channel, Enabled: p.Enabled}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	list, err := h.service.ListForUser(r.Context(), claims.UserID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to list notifications")
		return
	}
	out := make([]notificationResponse, len(list))
	for i, n := range list {
		out[i] = toNotificationResponse(n)
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) unreadCount(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	count, err := h.service.CountUnread(r.Context(), claims.UserID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to count unread notifications")
		return
	}
	platform.WriteJSON(w, http.StatusOK, map[string]int64{"count": count})
}

func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	notificationID, err := uuid.Parse(chi.URLParam(r, "notificationID"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid notification id")
		return
	}

	if err := h.service.MarkRead(r.Context(), claims.UserID, notificationID); err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to mark notification read")
		return
	}
	platform.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) markAllRead(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	if err := h.service.MarkAllRead(r.Context(), claims.UserID); err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to mark all notifications read")
		return
	}
	platform.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) listPreferences(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	list, err := h.service.GetPreferences(r.Context(), claims.UserID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to load notification preferences")
		return
	}
	out := make([]preferenceResponse, len(list))
	for i, p := range list {
		out[i] = toPreferenceResponse(p)
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

type setPreferenceRequest struct {
	Channel string `json:"channel"`
	Enabled bool   `json:"enabled"`
}

// setPreference upserts a single (channel, enabled) preference row for the
// caller — deliberately kept singular rather than batch, mirroring the
// roadmap's guidance that there's no Settings UI to drive a batch update yet.
func (h *Handler) setPreference(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	var req setPreferenceRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Channel == "" {
		platform.WriteError(w, http.StatusBadRequest, "channel is required")
		return
	}

	pref, err := h.service.SetPreference(r.Context(), claims.UserID, req.Channel, req.Enabled)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to set notification preference")
		return
	}
	platform.WriteJSON(w, http.StatusOK, toPreferenceResponse(pref))
}
