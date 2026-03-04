package ingestion

import (
	"fmt"
	"log"
	"strconv"
)

// Resolver maps WordPress post IDs to human-readable references.
type Resolver struct {
	// wpIDToDocID maps a WP post ID to a doc_id like "article_6"
	wpIDToDocID map[int]string
	// wpIDToTitle maps a WP post ID to the rendered title
	wpIDToTitle map[int]string
	// articleChapter maps a doc_id to its chapter Roman numeral
	articleChapter map[string]string
	// articleSection maps a doc_id to its section title
	articleSection map[string]string
}

// NewResolver builds a Resolver from chapters, sections, articles, recitals, and annexes.
func NewResolver(chapters, sections, articles, recitals, annexes []WPItem) *Resolver {
	r := &Resolver{
		wpIDToDocID:    make(map[int]string),
		wpIDToTitle:    make(map[int]string),
		articleChapter: make(map[string]string),
		articleSection: make(map[string]string),
	}

	// Index all items by WP ID
	for _, items := range []struct {
		docType string
		list    []WPItem
	}{
		{"chapter", chapters},
		{"section", sections},
		{"article", articles},
		{"recital", recitals},
		{"annex", annexes},
	} {
		for _, item := range items.list {
			docID := fmt.Sprintf("%s_%s", items.docType, item.Slug)
			r.wpIDToDocID[item.ID] = docID
			r.wpIDToTitle[item.ID] = item.Title.Rendered
		}
	}

	// Build chapter map: WP ID → chapter title (Roman numeral)
	chapterTitle := make(map[int]string)
	for _, ch := range chapters {
		chapterTitle[ch.ID] = ch.Title.Rendered
	}

	// Resolve article → chapter via sections
	for _, sec := range sections {
		mb := sec.MetaBox

		// Find parent chapter of this section
		var chapTitle string
		for _, chIDStr := range mb.TitleChapterFrom {
			if chID, err := strconv.Atoi(chIDStr); err == nil {
				if t, ok := chapterTitle[chID]; ok {
					chapTitle = t
					break
				}
			}
		}

		// Assign chapter and section to each article in this section
		for _, artIDStr := range mb.ChapterArticleTo {
			artID, err := strconv.Atoi(artIDStr)
			if err != nil {
				continue
			}
			docID, ok := r.wpIDToDocID[artID]
			if !ok {
				continue
			}
			if chapTitle != "" {
				r.articleChapter[docID] = chapTitle
			}
			r.articleSection[docID] = sec.Title.Rendered
		}
	}

	// Fallback: assign chapter via chapters' title-article_to field
	// This catches articles not in any section
	for _, ch := range chapters {
		for _, artIDStr := range ch.MetaBox.TitleArticleTo {
			artID, err := strconv.Atoi(artIDStr)
			if err != nil {
				continue
			}
			docID, ok := r.wpIDToDocID[artID]
			if !ok {
				continue
			}
			if _, already := r.articleChapter[docID]; !already {
				r.articleChapter[docID] = ch.Title.Rendered
			}
		}
	}

	log.Printf("Resolver: %d ID mappings, %d article-chapter mappings, %d article-section mappings",
		len(r.wpIDToDocID), len(r.articleChapter), len(r.articleSection))

	return r
}

// Chapter returns the chapter Roman numeral for a doc_id.
func (r *Resolver) Chapter(docID string) string {
	return r.articleChapter[docID]
}

// Section returns the section title for a doc_id.
func (r *Resolver) Section(docID string) string {
	return r.articleSection[docID]
}

// DocID returns the human-readable doc_id for a WP post ID.
func (r *Resolver) DocID(wpID int) string {
	return r.wpIDToDocID[wpID]
}
