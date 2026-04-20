package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/SheykoWk/event-streaming-and-audit/internal/application/consume"
	"github.com/SheykoWk/event-streaming-and-audit/internal/config"
	"github.com/SheykoWk/event-streaming-and-audit/internal/infrastructure/elasticsearch"
	"github.com/SheykoWk/event-streaming-and-audit/internal/infrastructure/kafka"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	indexer, err := elasticsearch.NewIndexer(cfg.Elasticsearch)
	if err != nil {
		log.Error("failed to init elasticsearch indexer", "error", err)
		os.Exit(1)
	}
	defer indexer.Close() //nolint:errcheck

	consumer := kafka.NewConsumer(cfg.Kafka, log)
	defer consumer.Close() //nolint:errcheck

	dlq := kafka.NewDLQProducer(cfg.Kafka)
	defer dlq.Close() //nolint:errcheck

	svc := consume.NewService(consumer, indexer, dlq, log)

	log.Info("consumer-service started",
		"topic", cfg.Kafka.Topic,
		"dlq_topic", cfg.Kafka.DLQTopic,
		"group", cfg.Kafka.GroupID,
		"brokers", cfg.Kafka.Brokers,
	)

	if err := svc.Run(ctx); err != nil {
		log.Error("consumer-service error", "error", err)
		os.Exit(1)
	}

	log.Info("consumer-service stopped")
}
