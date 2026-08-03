package search

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wadaag/health-platform/backend/internal/platform"
)

// ProviderResolver maps the calling user to their provider identity — used
// to scope the consultation facet the same way consults.Handler.inbox
// does. Satisfied by *facilities.Service, wired in main.go.
type ProviderResolver interface {
	ResolveProviderID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

type Handler struct {
	service  *Service
	provider ProviderResolver
	tm       *platform.TokenManager
}

func NewHandler(service *Service, provider ProviderResolver, tm *platform.TokenManager) *Handler {
	return &Handler{service: service, provider: provider, tm: tm}
}

// Routes is mounted at "/search" (see server.NewRouter), so the final paths
// are:
//
//	GET    /api/v1/search?q=...&type=patient,doctor
//	POST   /api/v1/search/saved
//	GET    /api/v1/search/saved
//	DELETE /api/v1/search/saved/{id}
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(platform.RequireAuth(h.tm))

	r.Get("/", h.search)
	r.Post("/saved", h.saveSearch)
	r.Get("/saved", h.listSaved)
	r.Delete("/saved/{id}", h.deleteSaved)

	return r
}

type resultResponse struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	Title     string `json:"title"`
	Subtitle  string `json:"subtitle,omitempty"`
	CreatedAt string `json:"created_at"`
}

func toResultResponse(r Result) resultResponse {
	return resultResponse{
		Type: r.Type, ID: r.ID.String(), Title: r.Title, Subtitle: r.Subtitle,
		CreatedAt: r.CreatedAt.Format(time.RFC3339),
	}
}

func parseTypes(raw string) []string {
	if raw == "" {
		return nil
	}
	var types []string
	for _, t := range strings.Split(raw, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			types = append(types, t)
		}
	}
	return types
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		platform.WriteError(w, http.StatusBadRequest, "q is required")
		return
	}
	types := parseTypes(r.URL.Query().Get("type"))

	// actorProviderID is resolved best-effort: a caller with no provider
	// profile (patient, system_admin, lab_tech, ...) simply won't get the
	// consultation facet populated — that's not treated as an error here
	// the way it is on consults' own inbox route, since a missing provider
	// profile shouldn't fail an otherwise-valid multi-facet search.
	var providerID *uuid.UUID
	if id, err := h.provider.ResolveProviderID(r.Context(), claims.UserID); err == nil {
		providerID = &id
	}

	results, err := h.service.Search(r.Context(), claims.UserID, claims.Role, claims.FacilityID, providerID, query, types)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "search failed")
		return
	}

	out := make(map[string][]resultResponse, len(results))
	for facet, list := range results {
		converted := make([]resultResponse, len(list))
		for i, res := range list {
			converted[i] = toResultResponse(res)
		}
		out[facet] = converted
	}

	platform.WriteJSON(w, http.StatusOK, out)
}

type saveSearchRequest struct {
	Name    string          `json:"name"`
	Query   string          `json:"query"`
	Filters json.RawMessage `json:"filters"`
}

type savedSearchResponse struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Query     string          `json:"query"`
	Filters   json.RawMessage `json:"filters"`
	CreatedAt string          `json:"created_at"`
}

func toSavedSearchResponse(s SavedSearch) savedSearchResponse {
	return savedSearchResponse{
		ID: s.ID.String(), Name: s.Name, Query: s.Query,
		Filters: json.RawMessage(s.Filters), CreatedAt: s.CreatedAt.Format(time.RFC3339),
	}
}

func (h *Handler) saveSearch(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	var req saveSearchRequest
	if err := platform.DecodeJSON(r, &req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Query == "" {
		platform.WriteError(w, http.StatusBadRequest, "name and query are required")
		return
	}

	saved, err := h.service.SaveSearch(r.Context(), claims.UserID, req.Name, req.Query, req.Filters)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to save search")
		return
	}
	platform.WriteJSON(w, http.StatusCreated, toSavedSearchResponse(saved))
}

func (h *Handler) listSaved(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	list, err := h.service.ListSavedSearches(r.Context(), claims.UserID)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "failed to list saved searches")
		return
	}
	out := make([]savedSearchResponse, len(list))
	for i, s := range list {
		out[i] = toSavedSearchResponse(s)
	}
	platform.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) deleteSaved(w http.ResponseWriter, r *http.Request) {
	claims, ok := platform.ClaimsFromContext(r.Context())
	if !ok {
		platform.WriteError(w, http.StatusUnauthorized, "missing auth context")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.service.DeleteSavedSearch(r.Context(), claims.UserID, id); err != nil {
		switch {
		case errors.Is(err, ErrSavedSearchNotFound):
			platform.WriteError(w, http.StatusNotFound, "saved search not found")
		case errors.Is(err, ErrNotSavedSearchOwner):
			platform.WriteError(w, http.StatusForbidden, "not your saved search")
		default:
			platform.WriteError(w, http.StatusInternalServerError, "failed to delete saved search")
		}
		return
	}
	platform.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
