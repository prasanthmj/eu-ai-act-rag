package rag

import (
	"context"
	"fmt"

	pb "github.com/qdrant/go-client/qdrant"
)

// SearchResult represents a document retrieved from Qdrant.
type SearchResult struct {
	DocType   string
	DocID     string
	Title     string
	Content   string
	Chapter   string
	Section   string
	CrossRefs []string
	Score     float32
}

// Searcher queries Qdrant collections.
type Searcher struct {
	client *pb.Client
}

// NewSearcher connects to Qdrant at the given address.
func NewSearcher(host string, port int) (*Searcher, error) {
	client, err := pb.NewClient(&pb.Config{
		Host: host,
		Port: port,
	})
	if err != nil {
		return nil, fmt.Errorf("connect to qdrant %s:%d: %w", host, port, err)
	}
	return &Searcher{client: client}, nil
}

// Close closes the Qdrant connection.
func (s *Searcher) Close() error {
	return s.client.Close()
}

// SparseQuery holds sparse vector data for hybrid search.
type SparseQuery struct {
	Indices []uint32
	Values  []float32
}

// HybridSearch performs dense + sparse vector search with RRF fusion.
func (s *Searcher) HybridSearch(ctx context.Context, collection string, denseVector []float32, sparse *SparseQuery, limit uint64) ([]SearchResult, error) {
	denseUsing := "dense"
	sparseUsing := "sparse"
	prefetchLimit := uint64(100)

	prefetch := []*pb.PrefetchQuery{
		{
			Query: pb.NewQueryDense(denseVector),
			Using: &denseUsing,
			Limit: &prefetchLimit,
		},
	}

	// Add sparse prefetch if we have sparse data
	if sparse != nil && len(sparse.Indices) > 0 {
		prefetch = append(prefetch, &pb.PrefetchQuery{
			Query: pb.NewQuerySparse(sparse.Indices, sparse.Values),
			Using: &sparseUsing,
			Limit: &prefetchLimit,
		})
	}

	points, err := s.client.Query(ctx, &pb.QueryPoints{
		CollectionName: collection,
		Prefetch:       prefetch,
		Query:          pb.NewQueryFusion(pb.Fusion_RRF),
		Limit:          &limit,
		WithPayload:    pb.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("hybrid search %s: %w", collection, err)
	}

	results := make([]SearchResult, len(points))
	for i, p := range points {
		results[i] = extractSearchResult(p.Payload, p.Score)
	}
	return results, nil
}

// LookupByDocID fetches a single document by its doc_id.
func (s *Searcher) LookupByDocID(ctx context.Context, collection, docID string) (*SearchResult, error) {
	results, err := s.LookupByDocIDs(ctx, collection, []string{docID})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("doc_id %q not found in %s", docID, collection)
	}
	return &results[0], nil
}

// LookupByDocIDs fetches multiple documents by their doc_ids.
func (s *Searcher) LookupByDocIDs(ctx context.Context, collection string, docIDs []string) ([]SearchResult, error) {
	if len(docIDs) == 0 {
		return nil, nil
	}

	points, err := s.client.Scroll(ctx, &pb.ScrollPoints{
		CollectionName: collection,
		Filter: &pb.Filter{
			Must: []*pb.Condition{
				pb.NewMatchKeywords("doc_id", docIDs...),
			},
		},
		WithPayload: pb.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("lookup doc_ids in %s: %w", collection, err)
	}

	results := make([]SearchResult, len(points))
	for i, p := range points {
		results[i] = extractSearchResult(p.Payload, 0)
	}
	return results, nil
}

func extractSearchResult(payload map[string]*pb.Value, score float32) SearchResult {
	r := SearchResult{Score: score}
	if v, ok := payload["doc_type"]; ok {
		r.DocType = v.GetStringValue()
	}
	if v, ok := payload["doc_id"]; ok {
		r.DocID = v.GetStringValue()
	}
	if v, ok := payload["title"]; ok {
		r.Title = v.GetStringValue()
	}
	if v, ok := payload["content"]; ok {
		r.Content = v.GetStringValue()
	}
	if v, ok := payload["chapter"]; ok {
		r.Chapter = v.GetStringValue()
	}
	if v, ok := payload["section"]; ok {
		r.Section = v.GetStringValue()
	}
	if v, ok := payload["cross_refs"]; ok {
		if list := v.GetListValue(); list != nil {
			for _, item := range list.Values {
				r.CrossRefs = append(r.CrossRefs, item.GetStringValue())
			}
		}
	}
	return r
}
