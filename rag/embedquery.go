package rag

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
)

// EmbedQuery embeds a single query string using text-embedding-3-small.
func EmbedQuery(ctx context.Context, client openai.Client, text string) ([]float32, error) {
	resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: openai.EmbeddingModelTextEmbedding3Small,
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: param.NewOpt(text),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("embed query: no embedding returned")
	}

	// Convert float64 → float32
	f64s := resp.Data[0].Embedding
	f32s := make([]float32, len(f64s))
	for i, v := range f64s {
		f32s[i] = float32(v)
	}
	return f32s, nil
}
