package ingestion

// WPItem represents a WordPress post for any content type (article, recital, annex, chapter, section).
type WPItem struct {
	ID      int     `json:"id"`
	Slug    string  `json:"slug"`
	Title   WPField `json:"title"`
	Content WPField `json:"content"`
	MetaBox MetaBox `json:"meta_box"`
}

// WPField is the standard WordPress rendered field.
type WPField struct {
	Rendered string `json:"rendered"`
}

// MetaBox holds the custom meta fields from the WP REST API.
// WP relationship fields return string arrays of post IDs.
// Different content types use different subsets of these fields.
type MetaBox struct {
	// Chapter → articles (used by chapters/titles)
	TitleArticleTo []string `json:"title-article_to,omitempty"`

	// Section → articles and parent chapter
	ChapterArticleTo []string `json:"chapter-article_to,omitempty"` // child article WP IDs
	TitleChapterFrom []string `json:"title-chapter_from,omitempty"` // parent chapter WP IDs

	// Article → parent chapter/section
	ChapterArticleFrom []string `json:"chapter-article_from,omitempty"` // parent chapter WP IDs
}

// Chunk is the final processed unit ready for embedding and storage.
type Chunk struct {
	DocType     string   `json:"doc_type"`    // "article", "recital", "annex"
	DocID       string   `json:"doc_id"`      // e.g. "article_6", "recital_29"
	Title       string   `json:"title"`       // rendered title text
	Chapter     string   `json:"chapter"`     // e.g. "III"
	Section     string   `json:"section"`     // section title if available
	CrossRefs   []string `json:"cross_refs"`  // e.g. ["article_27", "recital_29"]
	Content     string   `json:"content"`     // plain text content
	Version     string   `json:"version"`     // "2024-08-01" (EU AI Act final)
	LastUpdated string   `json:"last_updated"`
}

// ChunkWithEmbedding pairs a Chunk with its embedding vector.
type ChunkWithEmbedding struct {
	Chunk     Chunk     `json:"chunk"`
	Embedding []float32 `json:"embedding"`
}
