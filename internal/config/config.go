package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPAddr string
	GRPCAddr string
	Postgres      PostgresConfig
	Kafka         KafkaConfig
	Elasticsearch ElasticsearchConfig
}

type PostgresConfig struct {
	DSN string
}

type KafkaConfig struct {
	Brokers  []string
	Topic    string
	GroupID  string
	DLQTopic string
}

type ElasticsearchConfig struct {
	Addresses []string
	Index     string
}

func Load() *Config {
	return &Config{
		HTTPAddr: getEnv("HTTP_ADDR", ":8080"),
		GRPCAddr: getEnv("GRPC_ADDR", ":50051"),
		Postgres: PostgresConfig{
			DSN: getEnv("POSTGRES_DSN", "postgres://events:events@localhost:5432/events?sslmode=disable"),
		},
		Kafka: KafkaConfig{
			// 9094 = EXTERNAL listener exposed by docker-compose for host access
			Brokers:  strings.Split(getEnv("KAFKA_BROKERS", "localhost:9094"), ","),
			Topic:    getEnv("KAFKA_TOPIC", "events"),
			GroupID:  getEnv("KAFKA_GROUP_ID", "consumer-service"),
			DLQTopic: getEnv("KAFKA_DLQ_TOPIC", "events-dlq"),
		},
		Elasticsearch: ElasticsearchConfig{
			Addresses: strings.Split(getEnv("ELASTICSEARCH_ADDRS", "http://localhost:9200"), ","),
			Index:     getEnv("ELASTICSEARCH_INDEX", "events"),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
