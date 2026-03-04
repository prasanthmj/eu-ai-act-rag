package ingestion

import (
	"fmt"
	"time"
)

const euAIActVersion = "2024-08-01"

// BuildChunks converts WPItems into Chunks using the resolver for metadata enrichment.
func BuildChunks(docType string, items []WPItem, resolver *Resolver) ([]Chunk, error) {
	now := time.Now().Format("2006-01-02")
	var chunks []Chunk

	for _, item := range items {
		docID := fmt.Sprintf("%s_%s", docType, item.Slug)

		content, err := HTMLToPlainText(item.Content.Rendered)
		if err != nil {
			return nil, fmt.Errorf("parse content for %s: %w", docID, err)
		}

		crossRefs, err := ExtractCrossRefs(item.Content.Rendered)
		if err != nil {
			return nil, fmt.Errorf("extract cross-refs for %s: %w", docID, err)
		}

		chunk := Chunk{
			DocType:     docType,
			DocID:       docID,
			Title:       CleanTitle(item.Title.Rendered),
			Chapter:     resolver.Chapter(docID),
			Section:     resolver.Section(docID),
			CrossRefs:   crossRefs,
			Content:     content,
			Version:     euAIActVersion,
			LastUpdated: now,
		}

		chunks = append(chunks, chunk)
	}

	return chunks, nil
}
