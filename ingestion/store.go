package ingestion

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"

	"github.com/google/uuid"
	pb "github.com/qdrant/go-client/qdrant"
)

const vectorSize = 1536 // text-embedding-3-small dimension

// Store manages Qdrant collections and point storage.
type Store struct {
	client *pb.Client
}

// NewStore connects to Qdrant at the given address.
func NewStore(host string, port int) (*Store, error) {
	client, err := pb.NewClient(&pb.Config{
		Host: host,
		Port: port,
	})
	if err != nil {
		return nil, fmt.Errorf("connect to qdrant %s:%d: %w", host, port, err)
	}
	return &Store{client: client}, nil
}

// Close closes the Qdrant connection.
func (s *Store) Close() error {
	return s.client.Close()
}

// RecreateCollection deletes (if exists) and creates a collection with named dense + sparse vectors.
func (s *Store) RecreateCollection(ctx context.Context, name string) error {
	// Delete existing collection (ignore errors if it doesn't exist)
	_ = s.client.DeleteCollection(ctx, name)

	err := s.client.CreateCollection(ctx, &pb.CreateCollection{
		CollectionName: name,
		VectorsConfig: pb.NewVectorsConfigMap(map[string]*pb.VectorParams{
			"dense": {
				Size:     vectorSize,
				Distance: pb.Distance_Cosine,
			},
		}),
		SparseVectorsConfig: pb.NewSparseVectorsConfig(map[string]*pb.SparseVectorParams{
			"sparse": {},
		}),
	})
	if err != nil {
		return fmt.Errorf("create collection %s: %w", name, err)
	}

	log.Printf("Created collection: %s (dense + sparse vectors)", name)
	return nil
}

// UpsertChunks stores ChunkWithEmbeddings as Qdrant points with named dense + sparse vectors.
func (s *Store) UpsertChunks(ctx context.Context, collection string, chunks []ChunkWithEmbedding) error {
	points := make([]*pb.PointStruct, len(chunks))

	for i, cwe := range chunks {
		pointID := deterministicUUID(cwe.Chunk.DocID)

		vectors := map[string]*pb.Vector{
			"dense": pb.NewVectorDense(cwe.Embedding),
		}
		if len(cwe.Sparse.Indices) > 0 {
			vectors["sparse"] = pb.NewVectorSparse(cwe.Sparse.Indices, cwe.Sparse.Values)
		}

		points[i] = &pb.PointStruct{
			Id:      pb.NewIDUUID(pointID.String()),
			Vectors: pb.NewVectorsMap(vectors),
			Payload: chunkToPayload(cwe.Chunk),
		}
	}

	// Upsert in batches of 100
	for start := 0; start < len(points); start += 100 {
		end := start + 100
		if end > len(points) {
			end = len(points)
		}

		_, err := s.client.Upsert(ctx, &pb.UpsertPoints{
			CollectionName: collection,
			Points:         points[start:end],
		})
		if err != nil {
			return fmt.Errorf("upsert to %s (batch %d-%d): %w", collection, start, end, err)
		}
	}

	log.Printf("Upserted %d points to %s", len(points), collection)
	return nil
}

func chunkToPayload(c Chunk) map[string]*pb.Value {
	payload := map[string]*pb.Value{
		"doc_type":     pb.NewValueString(c.DocType),
		"doc_id":       pb.NewValueString(c.DocID),
		"title":        pb.NewValueString(c.Title),
		"content":      pb.NewValueString(c.Content),
		"version":      pb.NewValueString(c.Version),
		"last_updated": pb.NewValueString(c.LastUpdated),
	}

	if c.Chapter != "" {
		payload["chapter"] = pb.NewValueString(c.Chapter)
	}
	if c.Section != "" {
		payload["section"] = pb.NewValueString(c.Section)
	}
	if len(c.CrossRefs) > 0 {
		refs := make([]*pb.Value, len(c.CrossRefs))
		for i, r := range c.CrossRefs {
			refs[i] = pb.NewValueString(r)
		}
		payload["cross_refs"] = pb.NewValueFromList(refs...)
	}

	return payload
}

// deterministicUUID generates a UUID5 from a doc_id for idempotent upserts.
func deterministicUUID(docID string) uuid.UUID {
	// Use SHA-256 of doc_id as the "name" in a UUID5-like scheme
	hash := sha256.Sum256([]byte(docID))
	return uuid.NewSHA1(uuid.NameSpaceURL, hash[:])
}
