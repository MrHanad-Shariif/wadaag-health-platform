package server

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/wadaag/health-platform/backend/internal/appointments"
	"github.com/wadaag/health-platform/backend/internal/attachments"
	"github.com/wadaag/health-platform/backend/internal/audit"
	"github.com/wadaag/health-platform/backend/internal/authz"
	"github.com/wadaag/health-platform/backend/internal/consent"
	"github.com/wadaag/health-platform/backend/internal/consults"
	"github.com/wadaag/health-platform/backend/internal/dashboard"
	"github.com/wadaag/health-platform/backend/internal/facilities"
	"github.com/wadaag/health-platform/backend/internal/identity"
	"github.com/wadaag/health-platform/backend/internal/messaging"
	"github.com/wadaag/health-platform/backend/internal/notifications"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"github.com/wadaag/health-platform/backend/internal/records"
	"github.com/wadaag/health-platform/backend/internal/referrals"
	"github.com/wadaag/health-platform/backend/internal/search"
)

type Modules struct {
	Identity      *identity.Handler
	Facilities    *facilities.Handler
	Records       *records.Handler
	Referrals     *referrals.Handler
	Consent       *consent.Handler
	Audit         *audit.Handler
	Authz         *authz.Handler
	Dashboard     *dashboard.Handler
	Attachments   *attachments.Handler
	Consults      *consults.Handler
	Messaging     *messaging.Handler
	Notifications *notifications.Handler
	Search        *search.Handler
	Appointments  *appointments.Handler

	// RateLimiter is applied to the whole /api/v1 surface (see NewRouter).
	// Health checks under /healthz are deliberately excluded — they're
	// infra probes, not an abuse target, and rate-limiting them would just
	// risk breaking uptime monitoring.
	RateLimiter *platform.RateLimiter

	// UploadDir must match platform.Config.UploadDir, the directory
	// identity.Service writes profile photos under (at {UploadDir}/
	// profile-photos/) and attachments.Service writes clinical attachments
	// under (at {UploadDir}/attachments/{patientID}/). Only the
	// profile-photos subtree is served statically (see below) — attachments
	// are never reachable through the static mount, only through
	// attachments.Handler's authenticated route.
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

	// Profile photos are served plainly, outside the /api/v1 group and its
	// auth/rate-limit middleware, deliberately — they need to be viewable
	// via a bare <img src> without a bearer token, same as any other
	// publicly-linkable static asset.
	//
	// SECURITY: this mount's root MUST stay scoped to exactly the
	// profile-photos subdirectory, never modules.UploadDir itself. Clinical
	// attachments (internal/attachments) are written under
	// {UploadDir}/attachments/{patientID}/... — mounting the whole
	// UploadDir here would serve every patient's uploaded medical documents
	// to anyone, unauthenticated, plus (via http.FileServer's default
	// directory listing) let an anonymous caller enumerate every patient ID
	// that has attachments. Attachment downloads instead go through
	// attachments.Handler's authenticated, consent-gated streaming route —
	// see internal/attachments/handler.go.
	photoDir := filepath.Join(modules.UploadDir, "profile-photos")
	r.Handle("/uploads/profile-photos/*", http.StripPrefix("/uploads/profile-photos/", http.FileServer(noListingFileSystem{http.Dir(photoDir)})))

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
		api.Mount("/attachments", modules.Attachments.Routes())
		api.Mount("/consults", modules.Consults.Routes())
		api.Mount("/conversations", modules.Messaging.Routes())
		api.Mount("/notifications", modules.Notifications.Routes())
		api.Mount("/search", modules.Search.Routes())
		api.Mount("/appointments", modules.Appointments.Routes())
	})

	return r
}

// noListingFileSystem wraps an http.FileSystem to return "not found" for any
// directory that has no index.html, instead of http.FileServer's default
// behavior of rendering a directory listing — so filenames under a public
// static mount (e.g. other users' profile-photo filenames) aren't
// enumerable by an anonymous caller.
type noListingFileSystem struct {
	http.FileSystem
}

func (fs noListingFileSystem) Open(name string) (http.File, error) {
	f, err := fs.FileSystem.Open(name)
	if err != nil {
		return nil, err
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	if stat.IsDir() {
		index := filepath.Join(name, "index.html")
		if _, err := fs.FileSystem.Open(index); err != nil {
			f.Close()
			return nil, os.ErrNotExist
		}
	}

	return f, nil
}
