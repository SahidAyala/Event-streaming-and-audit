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

	"github.com/SheykoWk/event-streaming-and-audit/internal/application/ingest"
	"github.com/SheykoWk/event-streaming-and-audit/internal/application/query"
	"github.com/SheykoWk/event-streaming-and-audit/internal/config"
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

	router := httpserver.NewRouter(ingestSvc, querySvc, log)

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("ingest-api started", "addr", cfg.HTTPAddr)
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
