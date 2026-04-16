package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"

	"github.com/SheykoWk/event-streaming-and-audit/internal/config"
	"github.com/SheykoWk/event-streaming-and-audit/internal/domain/event"
)

// mapping defines the Elasticsearch index schema for events.
// keyword fields enable exact-match filtering and aggregations;
// occurred_at is a date so range queries and sorting work correctly.
const mapping = `{
  "mappings": {
    "properties": {
      "id":          { "type": "keyword" },
      "stream_id":   { "type": "keyword" },
      "type":        { "type": "keyword" },
      "source":      { "type": "keyword" },
      "version":     { "type": "long"    },
      "occurred_at": { "type": "date"    },
      "payload":     { "type": "object", "dynamic": true },
      "metadata":    { "type": "object", "dynamic": true }
    }
  }
}`

// Indexer writes events into an Elasticsearch index.
// Index calls are idempotent: the event UUID is used as the document ID,
// so reprocessing a message produces an overwrite, not a duplicate.
type Indexer struct {
	client *elasticsearch.Client
	index  string
}

func NewIndexer(cfg config.ElasticsearchConfig) (*Indexer, error) {
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: cfg.Addresses,
	})
	if err != nil {
		return nil, fmt.Errorf("create elasticsearch client: %w", err)
	}

	idx := &Indexer{client: client, index: cfg.Index}

	initCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := idx.ensureIndex(initCtx); err != nil {
		return nil, fmt.Errorf("ensure index: %w", err)
	}

	return idx, nil
}

// ensureIndex creates the index with the canonical mapping if it does not exist.
func (i *Indexer) ensureIndex(ctx context.Context) error {
	res, err := i.client.Indices.Exists(
		[]string{i.index},
		i.client.Indices.Exists.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("check index: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		return nil // already exists
	}

	createRes, err := i.client.Indices.Create(
		i.index,
		i.client.Indices.Create.WithBody(bytes.NewReader([]byte(mapping))),
		i.client.Indices.Create.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("create index request: %w", err)
	}
	defer createRes.Body.Close()

	if createRes.IsError() {
		b, _ := io.ReadAll(createRes.Body)
		return fmt.Errorf("create index [%s]: %s", createRes.Status(), b)
	}
	return nil
}

// Index upserts the event as a document. Uses op_type=index (create-or-overwrite)
// so at-least-once delivery from Kafka does not produce duplicate documents.
func (i *Indexer) Index(ctx context.Context, e *event.Event) error {
	body, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	req := esapi.IndexRequest{
		Index:      i.index,
		DocumentID: e.ID.String(),
		Body:       bytes.NewReader(body),
		OpType:     "index",
		Refresh:    "false",
	}

	res, err := req.Do(ctx, i.client)
	if err != nil {
		return fmt.Errorf("elasticsearch request: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("elasticsearch [%s]: %s", res.Status(), b)
	}
	return nil
}

// Close is a no-op; elasticsearch.Client uses http.DefaultTransport which
// manages its own connection pool.
func (i *Indexer) Close() error { return nil }
