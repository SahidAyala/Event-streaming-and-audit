// Package main is the entrypoint for the ingest-api service.
//
// @title          Event Streaming and Audit API
// @version        1.0
// @description    HTTP API for ingesting and querying events in the Event Streaming & Audit Platform.
// @description
// @description    ## Authentication
// @description    All /events endpoints require authentication. /health and /swagger are public.
// @description    Set AUTH_MODE=simple (default) to use X-API-Key, or AUTH_MODE=jwt for Bearer tokens.
// @description
// @description    ## Data consistency
// @description    Write path: PostgreSQL (source of truth) → Kafka → Elasticsearch (read model).
// @description    GET /events/{streamID} is served from Elasticsearch and is eventually consistent.
//
// @host        localhost:8080
// @BasePath    /
// @schemes     http https
//
// @securityDefinitions.apikey  ApiKeyAuth
// @in                          header
// @name                        X-API-Key
// @description                 Static API key. Default value for local dev: dev-api-key. Configured via AUTH_API_KEY env var.
//
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 JWT Bearer token. Format: "Bearer {token}". Enable with AUTH_MODE=jwt and AUTH_JWT_SECRET env var.
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

	_ "github.com/SheykoWk/event-streaming-and-audit/docs"

	appauth "github.com/SheykoWk/event-streaming-and-audit/internal/application/auth"
	"github.com/SheykoWk/event-streaming-and-audit/internal/application/ingest"
	"github.com/SheykoWk/event-streaming-and-audit/internal/application/query"
	"github.com/SheykoWk/event-streaming-and-audit/internal/config"
	infraauth "github.com/SheykoWk/event-streaming-and-audit/internal/infrastructure/auth"
	"github.com/SheykoWk/event-streaming-and-audit/internal/infrastructure/auth/jwt"
	"github.com/SheykoWk/event-streaming-and-audit/internal/infrastructure/auth/simple"
	"github.com/SheykoWk/event-streaming-and-audit/internal/infrastructure/elasticsearch"
	"github.com/SheykoWk/event-streaming-and-audit/internal/infrastructure/httpserver"
	"github.com/SheykoWk/event-streaming-and-audit/internal/infrastructure/kafka"
	"github.com/SheykoWk/event-streaming-and-audit/internal/infrastructure/postgres"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Write-side: PostgreSQL event store (source of truth).
	store, err := postgres.NewEventStore(ctx, cfg.Postgres)
	if err != nil {
		log.Error("failed to init event store", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	// Write-side: Kafka producer (best-effort async publish).
	publisher := kafka.NewProducer(cfg.Kafka)
	defer publisher.Close() //nolint:errcheck

	// Read-side: Elasticsearch indexer also acts as the Searcher port.
	esIndexer, err := elasticsearch.NewIndexer(cfg.Elasticsearch)
	if err != nil {
		log.Error("failed to init elasticsearch", "error", err)
		os.Exit(1)
	}

	ingestSvc := ingest.NewService(store, publisher, log)
	querySvc := query.NewService(esIndexer, log)

	authenticator := buildAuthenticator(cfg.Auth, log)
	issuer := buildIssuer(cfg.Auth, log)

	router := httpserver.NewRouter(ingestSvc, querySvc, authenticator, issuer, cfg.Auth.AdminKey, log)

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("ingest-api started",
			"addr", cfg.HTTPAddr,
			"auth_mode", cfg.Auth.Mode,
			"swagger_ui", cfg.HTTPAddr+"/swagger/index.html",
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
	}
	log.Info("ingest-api stopped")
}

// buildIssuer constructs a JWT Issuer when AUTH_JWT_SECRET is set.
// Returns nil otherwise — the /auth/issue endpoint will respond with 501.
func buildIssuer(cfg config.AuthConfig, log *slog.Logger) appauth.Issuer {
	if cfg.JWTSecret == "" {
		log.Info("token issuance disabled — set AUTH_JWT_SECRET to enable POST /auth/issue")
		return nil
	}
	log.Info("token issuance enabled (JWT/HMAC)")
	return jwt.New(cfg.JWTSecret)
}

// buildAuthenticator selects and constructs the configured Authenticator.
// Defaults to simple if mode is unrecognised.
func buildAuthenticator(cfg config.AuthConfig, log *slog.Logger) infraauth.Authenticator {
	switch cfg.Mode {
	case "jwt":
		if cfg.JWTSecret == "" {
			log.Error("AUTH_JWT_SECRET must be set when AUTH_MODE=jwt")
			os.Exit(1)
		}
		log.Info("auth mode: jwt (multi-tenant)")
		return jwt.New(cfg.JWTSecret)
	default:
		log.Info("auth mode: simple (api-key)", "mode", cfg.Mode)
		return simple.New(cfg.APIKey)
	}
}
