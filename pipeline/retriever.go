package pipeline

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/prasanthmj/eu-ai-act-rag/rag"
)

const (
	annexCollection   = "eu_ai_act_annexes"
	articleCollection  = "eu_ai_act_articles"
	recitalCollection  = "eu_ai_act_recitals"
)

// EmbedFn is a function that embeds a query string into a dense vector.
type EmbedFn func(ctx context.Context, text string) ([]float32, error)

// SparseEmbedFn is a function that encodes a query into a sparse vector.
type SparseEmbedFn func(text string) *rag.SparseQuery

// Retrieve runs Stage 2: multi-hop retrieval from Qdrant.
// Hop 1: hybrid search annexes, Hop 2: follow cross_refs to articles, Hop 3: follow to recitals.
func Retrieve(ctx context.Context, searcher *rag.Searcher, embedFn EmbedFn, sparseEmbedFn SparseEmbedFn, classification *ClassifyResult, description string) ([]RetrievedChunk, error) {
	// Build search query from description + classification context
	query := description
	if classification.Domain != "" && classification.Domain != "unknown" {
		query = fmt.Sprintf("%s (domain: %s, risk: %s)", description, classification.Domain, strings.Join(classification.RiskTiers, ", "))
	}

	// Hop 1: Hybrid search annexes (dense + sparse)
	vector, err := embedFn(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	var sparse *rag.SparseQuery
	if sparseEmbedFn != nil {
		sparse = sparseEmbedFn(query)
	}

	annexResults, err := searcher.HybridSearch(ctx, annexCollection, vector, sparse, 5)
	if err != nil {
		return nil, fmt.Errorf("hop 1 (annexes): %w", err)
	}
	log.Printf("Retriever hop 1: %d annex results", len(annexResults))

	var chunks []RetrievedChunk
	articleIDs := map[string]bool{"article_3": true} // Always include definitions

	for _, r := range annexResults {
		chunks = append(chunks, RetrievedChunk{SearchResult: r, Hop: 1})
		for _, ref := range r.CrossRefs {
			if strings.HasPrefix(ref, "article_") {
				articleIDs[ref] = true
			}
		}
	}

	// Hop 2: Lookup linked articles
	artIDs := mapKeys(articleIDs)
	articleResults, err := searcher.LookupByDocIDs(ctx, articleCollection, artIDs)
	if err != nil {
		return nil, fmt.Errorf("hop 2 (articles): %w", err)
	}
	log.Printf("Retriever hop 2: %d article results", len(articleResults))

	recitalIDs := map[string]bool{}
	for _, r := range articleResults {
		chunks = append(chunks, RetrievedChunk{SearchResult: r, Hop: 2})
		for _, ref := range r.CrossRefs {
			if strings.HasPrefix(ref, "recital_") {
				recitalIDs[ref] = true
			}
		}
	}

	// Hop 3: Lookup linked recitals
	recIDs := mapKeys(recitalIDs)
	if len(recIDs) > 0 {
		recitalResults, err := searcher.LookupByDocIDs(ctx, recitalCollection, recIDs)
		if err != nil {
			return nil, fmt.Errorf("hop 3 (recitals): %w", err)
		}
		log.Printf("Retriever hop 3: %d recital results", len(recitalResults))

		for _, r := range recitalResults {
			chunks = append(chunks, RetrievedChunk{SearchResult: r, Hop: 3})
		}
	}

	return chunks, nil
}

func mapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
