package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	"github.com/wadaag/health-platform/backend/internal/server"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

// dashboardDoctorActivityAdapter satisfies dashboard.DoctorActivityCounter
// by delegating to referrals.Service.MostActiveDoctors and converting its
// []referrals.ProviderReferralActivity result to []dashboard.DoctorActivity
// — see referrals.ProviderReferralActivity's doc comment for why this
// conversion lives here instead of either module importing the other.
type dashboardDoctorActivityAdapter struct {
	referrals *referrals.Service
}

func (a dashboardDoctorActivityAdapter) MostActiveDoctors(ctx context.Context, limit int) ([]dashboard.DoctorActivity, error) {
	rows, err := a.referrals.MostActiveDoctors(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]dashboard.DoctorActivity, len(rows))
	for i, row := range rows {
		out[i] = dashboard.DoctorActivity{ProviderID: row.ProviderID, FullName: row.FullName, ReferralCount: row.ReferralCount}
	}
	return out, nil
}

func run() error {
	cfg, err := platform.LoadConfig()
	if err != nil {
		return err
	}

	logger := platform.NewLogger(cfg.Env)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := platform.NewDBPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	tokenManager := platform.NewTokenManager(cfg)
	rateLimiter := platform.NewDefaultRateLimiter()
	loginLockout := platform.NewDefaultLoginLockout()

	// Wiring order follows dependency direction: facilities and audit have
	// no dependencies on other domain modules; consent depends on audit;
	// records and referrals depend on consent + facilities; identity
	// depends on facilities (to resolve a provider's facility claim) even
	// though facilities is "below" it in the module table, via the
	// FacilityResolver interface rather than a direct import cycle.
	//
	// identity <-> authz is a genuine circular dependency at the wiring
	// level (authz creates accounts via identity; identity resolves
	// permissions via authz) — broken with SetPermissionResolver after
	// both are constructed. See identity.Service.SetPermissionResolver.
	facilitiesRepo := facilities.NewRepository(db)
	facilitiesService := facilities.NewService(facilitiesRepo)
	facilitiesHandler := facilities.NewHandler(facilitiesService, tokenManager)

	// notificationsService has no dependencies on the other domain modules,
	// so it's constructed early — its concrete *notifications.Service is
	// then handed to referrals/consults/messaging/records below via each
	// package's own SetNotificationPublisher setter (they each define a
	// narrow local Publisher interface rather than importing this package
	// directly; *notifications.Service satisfies each structurally).
	notificationsRepo := notifications.NewRepository(db)
	notificationsService := notifications.NewService(notificationsRepo)
	notificationsHandler := notifications.NewHandler(notificationsService, tokenManager)

	auditRepo := audit.NewRepository(db)
	auditLogger := audit.NewLogger(auditRepo)

	consentRepo := consent.NewRepository(db)
	consentChecker := consent.NewChecker(consentRepo)

	identityRepo := identity.NewRepository(db)
	identityService := identity.NewService(identityRepo, tokenManager, facilitiesService, cfg.UploadDir)
	identityHandler := identity.NewHandler(identityService, tokenManager, loginLockout)

	authzRepo := authz.NewRepository(db)
	authzService := authz.NewService(authzRepo, identityService, facilitiesService)
	authzHandler := authz.NewHandler(authzService, tokenManager)
	identityService.SetPermissionResolver(authzService)

	recordsRepo := records.NewRepository(db)
	recordsService := records.NewService(recordsRepo, consentChecker, facilitiesService)
	recordsHandler := records.NewHandler(recordsService, consentChecker, auditLogger, tokenManager)
	recordsService.SetNotificationPublisher(notificationsService)

	referralsRepo := referrals.NewRepository(db)
	referralsService := referrals.NewService(referralsRepo, consentChecker, facilitiesService)
	referralsHandler := referrals.NewHandler(referralsService, consentChecker, auditLogger, recordsService, tokenManager)
	referralsService.SetProviderLookup(facilitiesService)
	referralsService.SetNotificationPublisher(notificationsService)

	consentHandler := consent.NewHandler(consentChecker, tokenManager, recordsService)
	auditHandler := audit.NewHandler(auditLogger, tokenManager, recordsService)

	attachmentsRepo := attachments.NewRepository(db)
	attachmentsService := attachments.NewService(attachmentsRepo, recordsService, cfg.UploadDir)
	attachmentsHandler := attachments.NewHandler(attachmentsService, consentChecker, auditLogger, tokenManager)

	// dashboardDoctorActivity adapts referrals.Service.MostActiveDoctors
	// (which returns referrals' own ProviderReferralActivity type) to
	// dashboard.DoctorActivityCounter's []dashboard.DoctorActivity — kept
	// here rather than as a method on either service so neither package
	// needs to import the other (see referrals.ProviderReferralActivity's
	// doc comment).
	dashboardService := dashboard.NewService(
		recordsService, recordsService, facilitiesService, authzService, referralsService,
		dashboardDoctorActivityAdapter{referrals: referralsService}, referralsService, recordsService,
	)
	dashboardHandler := dashboard.NewHandler(dashboardService, tokenManager, facilitiesService)

	consultsRepo := consults.NewRepository(db)
	consultsService := consults.NewService(consultsRepo, facilitiesService, facilitiesService, recordsService, attachmentsService)
	consultsHandler := consults.NewHandler(consultsService, auditLogger, tokenManager)
	consultsService.SetNotificationPublisher(notificationsService)
	// consults is constructed after dashboard (it depends on
	// recordsService/attachmentsService, which dashboard doesn't need to
	// wait for), so dashboard's optional consult-pending dependency is
	// wired via setter rather than a NewService parameter — see
	// dashboard.Service.SetConsultPendingCounter.
	dashboardService.SetConsultPendingCounter(consultsService)

	// messagingHub has no dependencies (a plain in-memory registry), so it's
	// constructed early, same as everything else with no wiring
	// prerequisites — see internal/messaging/hub.go.
	messagingHub := messaging.NewHub()
	messagingRepo := messaging.NewRepository(db)
	messagingService := messaging.NewService(messagingRepo)
	messagingHandler := messaging.NewHandler(messagingService, messagingHub, tokenManager)
	messagingService.SetNotificationPublisher(notificationsService)

	// search is constructed last among the domain modules — it only reads
	// through records/facilities/referrals/consults' existing service
	// methods (see internal/search/service.go), so every dependency it
	// needs must already exist.
	searchRepo := search.NewRepository(db)
	searchService := search.NewService(searchRepo, recordsService, facilitiesService, referralsService, consultsService)
	searchHandler := search.NewHandler(searchService, facilitiesService, tokenManager)

	appointmentsRepo := appointments.NewRepository(db)
	appointmentsService := appointments.NewService(appointmentsRepo, facilitiesService, facilitiesService, recordsService)
	appointmentsHandler := appointments.NewHandler(appointmentsService, tokenManager)
	appointmentsService.SetNotificationPublisher(notificationsService)

	// Reminder delivery is a lightweight cron-style ticker rather than a
	// separate job queue/scheduler (see the design doc for Phase 8) — it's
	// tied to the same shutdown context as the HTTP server below so it
	// stops cleanly instead of leaking a goroutine past shutdown.
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sent, err := appointmentsService.SendDueReminders(context.Background())
				if err != nil {
					slog.Error("failed to send appointment reminders", "error", err)
					continue
				}
				if sent > 0 {
					slog.Info("sent appointment reminders", "count", sent)
				}
			}
		}
	}()

	router := server.NewRouter(server.Modules{
		Identity:      identityHandler,
		Facilities:    facilitiesHandler,
		Records:       recordsHandler,
		Referrals:     referralsHandler,
		Consent:       consentHandler,
		Audit:         auditHandler,
		Authz:         authzHandler,
		Dashboard:     dashboardHandler,
		Attachments:   attachmentsHandler,
		Consults:      consultsHandler,
		Messaging:     messagingHandler,
		Notifications: notificationsHandler,
		Search:        searchHandler,
		Appointments:  appointmentsHandler,
		RateLimiter:   rateLimiter,
		UploadDir:     cfg.UploadDir,
	})

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", "port", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutting down")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return httpServer.Shutdown(shutdownCtx)
}
