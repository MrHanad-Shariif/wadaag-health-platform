package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/wadaag/health-platform/backend/internal/audit"
	"github.com/wadaag/health-platform/backend/internal/authz"
	"github.com/wadaag/health-platform/backend/internal/consent"
	"github.com/wadaag/health-platform/backend/internal/dashboard"
	"github.com/wadaag/health-platform/backend/internal/facilities"
	"github.com/wadaag/health-platform/backend/internal/identity"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"github.com/wadaag/health-platform/backend/internal/records"
	"github.com/wadaag/health-platform/backend/internal/referrals"
)

type Modules struct {
	Identity   *identity.Handler
	Facilities *facilities.Handler
	Records    *records.Handler
	Referrals  *referrals.Handler
	Consent    *consent.Handler
	Audit      *audit.Handler
	Authz      *authz.Handler
	Dashboard  *dashboard.Handler

	// RateLimiter is applied to the whole /api/v1 surface (see NewRouter).
	// Health checks under /healthz are deliberately excluded — they're
	// infra probes, not an abuse target, and rate-limiting them would just
	// risk breaking uptime monitoring.
	RateLimiter *platform.RateLimiter

	// UploadDir is served statically at /uploads/* (see below) — must match
	// platform.Config.UploadDir, the directory identity.Service writes
	// uploaded files (e.g. profile photos) under.
	UploadDir string
}

// NewRouter is the one place middleware ordering is decided. Every
// patient-data route added by later modules must follow the same shape:
// RequireAuth -> RequireRoles -> [consent check] -> [audit log] -> handler.
// Getting that order right here is what makes "every access is logged"
// a structural guarantee instead of a convention each handler has to
// remember. records/referrals/consent/audit routes below wire this chain
// via consent.Middleware; see each module's Routes() for specifics.
func NewRouter(modules Modules) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		platform.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Uploaded files (profile photos today) are served plainly, outside the
	// /api/v1 group and its auth/rate-limit middleware, deliberately — they
	// need to be viewable via a bare <img src> without a bearer token, same
	// as any other publicly-linkable static asset.
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir(modules.UploadDir))))

	r.Route("/api/v1", func(api chi.Router) {
		api.Use(modules.RateLimiter.Middleware)
		api.Mount("/auth", modules.Identity.Routes())
		api.Mount("/facilities", modules.Facilities.Routes())
		api.Mount("/records", modules.Records.Routes())
		api.Mount("/referrals", modules.Referrals.Routes())
		api.Mount("/consent", modules.Consent.Routes())
		api.Mount("/audit", modules.Audit.Routes())
		api.Mount("/authz", modules.Authz.Routes())
		api.Mount("/dashboard", modules.Dashboard.Routes())
	})

	return r
}
