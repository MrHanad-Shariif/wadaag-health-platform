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

	"github.com/wadaag/health-platform/backend/internal/audit"
	"github.com/wadaag/health-platform/backend/internal/consent"
	"github.com/wadaag/health-platform/backend/internal/facilities"
	"github.com/wadaag/health-platform/backend/internal/identity"
	"github.com/wadaag/health-platform/backend/internal/platform"
	"github.com/wadaag/health-platform/backend/internal/records"
	"github.com/wadaag/health-platform/backend/internal/referrals"
	"github.com/wadaag/health-platform/backend/internal/server"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
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

	// Wiring order follows dependency direction: facilities and audit have
	// no dependencies on other domain modules; consent depends on audit;
	// records and referrals depend on consent + facilities; identity
	// depends on facilities (to resolve a provider's facility claim) even
	// though facilities is "below" it in the module table, via the
	// FacilityResolver interface rather than a direct import cycle.
	facilitiesRepo := facilities.NewRepository(db)
	facilitiesService := facilities.NewService(facilitiesRepo)
	facilitiesHandler := facilities.NewHandler(facilitiesService, tokenManager)

	auditRepo := audit.NewRepository(db)
	auditLogger := audit.NewLogger(auditRepo)

	consentRepo := consent.NewRepository(db)
	consentChecker := consent.NewChecker(consentRepo)

	identityRepo := identity.NewRepository(db)
	identityService := identity.NewService(identityRepo, tokenManager, facilitiesService)
	identityHandler := identity.NewHandler(identityService, tokenManager)

	recordsRepo := records.NewRepository(db)
	recordsService := records.NewService(recordsRepo, consentChecker, facilitiesService)
	recordsHandler := records.NewHandler(recordsService, consentChecker, auditLogger, tokenManager)

	referralsRepo := referrals.NewRepository(db)
	referralsService := referrals.NewService(referralsRepo, consentChecker, facilitiesService)
	referralsHandler := referrals.NewHandler(referralsService, consentChecker, auditLogger, recordsService, tokenManager)

	consentHandler := consent.NewHandler(consentChecker, tokenManager, recordsService)
	auditHandler := audit.NewHandler(auditLogger, tokenManager, recordsService)

	router := server.NewRouter(server.Modules{
		Identity:   identityHandler,
		Facilities: facilitiesHandler,
		Records:    recordsHandler,
		Referrals:  referralsHandler,
		Consent:    consentHandler,
		Audit:      auditHandler,
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
