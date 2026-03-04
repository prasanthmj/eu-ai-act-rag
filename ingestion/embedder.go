package ingestion

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/openai/openai-go"
)

const (
	batchSize  = 100
	maxRetries = 3
)

// Embedder generates embeddings via the OpenAI API.
type Embedder struct {
	client openai.Client
}

// NewEmbedder creates an Embedder. Uses OPENAI_API_KEY env var automatically.
func NewEmbedder() *Embedder {
	return &Embedder{
		client: openai.NewClient(),
	}
}

// EmbedChunks generates embeddings for all chunks and returns ChunkWithEmbeddings.
func (e *Embedder) EmbedChunks(ctx context.Context, chunks []Chunk) ([]ChunkWithEmbedding, error) {
	// Prepare texts: title + content
	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.Title + "\n\n" + c.Content
	}

	// Process in batches
	allEmbeddings := make([][]float32, len(chunks))

	for start := 0; start < len(texts); start += batchSize {
		end := start + batchSize
		if end > len(texts) {
			end = len(texts)
		}
		batch := texts[start:end]
		log.Printf("Embedding batch %d-%d of %d...", start+1, end, len(texts))

		embeddings, err := e.embedBatchWithRetry(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("embed batch %d-%d: %w", start, end, err)
		}

		for i, emb := range embeddings {
			allEmbeddings[start+i] = emb
		}
	}

	// Combine chunks with embeddings
	results := make([]ChunkWithEmbedding, len(chunks))
	for i, c := range chunks {
		results[i] = ChunkWithEmbedding{
			Chunk:     c,
			Embedding: allEmbeddings[i],
		}
	}

	return results, nil
}

func (e *Embedder) embedBatchWithRetry(ctx context.Context, texts []string) ([][]float32, error) {
	for attempt := range maxRetries {
		resp, err := e.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
			Model: openai.EmbeddingModelTextEmbedding3Small,
			Input: openai.EmbeddingNewParamsInputUnion{
				OfArrayOfStrings: texts,
			},
		})
		if err == nil {
			embeddings := make([][]float32, len(resp.Data))
			for i, d := range resp.Data {
				embeddings[i] = toFloat32(d.Embedding)
			}
			return embeddings, nil
		}

		if attempt < maxRetries-1 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			log.Printf("Embedding API error (attempt %d/%d), retrying in %v: %v", attempt+1, maxRetries, backoff, err)
			time.Sleep(backoff)
			continue
		}
		return nil, err
	}

	return nil, fmt.Errorf("unreachable")
}

func toFloat32(f64s []float64) []float32 {
	f32s := make([]float32, len(f64s))
	for i, v := range f64s {
		f32s[i] = float32(v)
	}
	return f32s
}
