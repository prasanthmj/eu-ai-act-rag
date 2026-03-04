package ingestion

import (
	"strings"
	"testing"
)

// These tests fetch live data from the WordPress API to verify
// that the content is reachable and our parsing works correctly.
// Run with: go test -v ./ingestion/ -run TestLive -count=1

func TestLiveFetchArticle(t *testing.T) {
	fetcher := NewFetcher()

	items, err := fetcher.FetchBySlug("article", "6")
	if err != nil {
		t.Fatalf("Failed to fetch article 6: %v", err)
	}
	art := items[0]

	if art.ID == 0 {
		t.Error("Article ID should not be zero")
	}
	if art.Slug != "6" {
		t.Errorf("Expected slug '6', got %q", art.Slug)
	}
	if !strings.Contains(art.Title.Rendered, "Article 6") {
		t.Errorf("Title should contain 'Article 6', got %q", art.Title.Rendered)
	}
	if art.Content.Rendered == "" {
		t.Error("Content should not be empty")
	}

	t.Logf("Article 6: ID=%d, Title=%q", art.ID, art.Title.Rendered)
	t.Logf("MetaBox chapter-article_from: %v", art.MetaBox.ChapterArticleFrom)
}

func TestLiveFetchChapter(t *testing.T) {
	fetcher := NewFetcher()

	items, err := fetcher.FetchBySlug("chapter", "3")
	if err != nil {
		t.Fatalf("Failed to fetch chapter 3: %v", err)
	}
	ch := items[0]

	if ch.ID == 0 {
		t.Error("Chapter ID should not be zero")
	}
	if !strings.Contains(ch.Title.Rendered, "III") {
		t.Errorf("Chapter 3 title should contain 'III', got %q", ch.Title.Rendered)
	}

	t.Logf("Chapter 3: ID=%d, Title=%q", ch.ID, ch.Title.Rendered)
}

func TestLiveFetchSection(t *testing.T) {
	fetcher := NewFetcher()

	// Section slugs use "{chapter}-{section}" format, e.g. "3-1"
	items, err := fetcher.FetchBySlug("section", "3-1")
	if err != nil {
		t.Fatalf("Failed to fetch section 3-1: %v", err)
	}
	sec := items[0]

	if sec.ID == 0 {
		t.Error("Section ID should not be zero")
	}
	if len(sec.MetaBox.ChapterArticleTo) == 0 {
		t.Error("Section should have chapter-article_to references")
	}
	if len(sec.MetaBox.TitleChapterFrom) == 0 {
		t.Error("Section should have title-chapter_from references")
	}

	t.Logf("Section 3-1: ID=%d, Title=%q", sec.ID, sec.Title.Rendered)
	t.Logf("  chapter-article_to: %v", sec.MetaBox.ChapterArticleTo)
	t.Logf("  title-chapter_from: %v", sec.MetaBox.TitleChapterFrom)
}

func TestLiveFetchRecital(t *testing.T) {
	fetcher := NewFetcher()

	items, err := fetcher.FetchBySlug("recital", "47")
	if err != nil {
		t.Fatalf("Failed to fetch recital 47: %v", err)
	}
	rec := items[0]

	if rec.ID == 0 {
		t.Error("Recital ID should not be zero")
	}
	if !strings.Contains(rec.Title.Rendered, "Recital 47") {
		t.Errorf("Title should contain 'Recital 47', got %q", rec.Title.Rendered)
	}
	if rec.Content.Rendered == "" {
		t.Error("Content should not be empty")
	}

	t.Logf("Recital 47: ID=%d, Title=%q", rec.ID, rec.Title.Rendered)
}

func TestLiveParseArticle6(t *testing.T) {
	fetcher := NewFetcher()

	items, err := fetcher.FetchBySlug("article", "6")
	if err != nil {
		t.Fatalf("Failed to fetch article 6: %v", err)
	}
	art := items[0]

	// Test cross-reference extraction
	crossRefs, err := ExtractCrossRefs(art.Content.Rendered)
	if err != nil {
		t.Fatalf("ExtractCrossRefs failed: %v", err)
	}

	if len(crossRefs) == 0 {
		t.Error("Article 6 should have cross-references")
	}

	// Article 6 references Annex I, Annex III, Article 49, Article 96, Article 97, Article 7
	// and recitals 47, 48, 50-63
	hasAnnex := false
	hasArticle := false
	hasRecital := false
	for _, ref := range crossRefs {
		if strings.HasPrefix(ref, "annex_") {
			hasAnnex = true
		}
		if strings.HasPrefix(ref, "article_") {
			hasArticle = true
		}
		if strings.HasPrefix(ref, "recital_") {
			hasRecital = true
		}
	}

	if !hasAnnex {
		t.Error("Article 6 should reference annexes")
	}
	if !hasArticle {
		t.Error("Article 6 should reference other articles")
	}
	if !hasRecital {
		t.Error("Article 6 should reference recitals")
	}

	t.Logf("Cross-refs (%d): %v", len(crossRefs), crossRefs)

	// Test HTML to plain text
	plainText, err := HTMLToPlainText(art.Content.Rendered)
	if err != nil {
		t.Fatalf("HTMLToPlainText failed: %v", err)
	}

	if plainText == "" {
		t.Error("Plain text should not be empty")
	}

	// Should contain the actual regulation text
	if !strings.Contains(plainText, "high-risk") {
		t.Error("Plain text should contain 'high-risk'")
	}

	// Should have indented sub-items (padding-left: 40px → 2 spaces)
	if !strings.Contains(plainText, "  (a)") {
		t.Error("Plain text should have indented sub-items starting with '  (a)'")
	}

	// Recital reference spans should be stripped
	if strings.Contains(plainText, "aia-recital-ref") {
		t.Error("Plain text should not contain recital reference HTML classes")
	}

	// Should NOT contain HTML tags
	if strings.Contains(plainText, "<p>") || strings.Contains(plainText, "<a ") {
		t.Error("Plain text should not contain HTML tags")
	}

	t.Logf("Plain text length: %d chars", len(plainText))
	t.Logf("First 300 chars:\n%s", truncate(plainText, 300))
}

func TestLiveFetchPagination(t *testing.T) {
	fetcher := NewFetcher()

	// Articles have >100 items, so this tests pagination
	articles, err := fetcher.FetchAll("article")
	if err != nil {
		t.Fatalf("FetchAll articles failed: %v", err)
	}

	if len(articles) < 100 {
		t.Errorf("Expected >100 articles, got %d", len(articles))
	}

	t.Logf("Fetched %d articles (pagination working)", len(articles))
}

func TestLiveResolverIntegration(t *testing.T) {
	fetcher := NewFetcher()

	chapters, err := fetcher.FetchAll("chapter")
	if err != nil {
		t.Fatalf("Fetch chapters: %v", err)
	}

	sections, err := fetcher.FetchAll("section")
	if err != nil {
		t.Fatalf("Fetch sections: %v", err)
	}

	articles, err := fetcher.FetchAll("article")
	if err != nil {
		t.Fatalf("Fetch articles: %v", err)
	}

	resolver := NewResolver(chapters, sections, articles, nil, nil)

	// Article 6 should have a chapter assignment
	chap := resolver.Chapter("article_6")
	if chap == "" {
		t.Error("Article 6 should have a chapter assignment")
	} else {
		t.Logf("Article 6 chapter: %q", chap)
	}

	sec := resolver.Section("article_6")
	if sec == "" {
		t.Error("Article 6 should have a section assignment")
	} else {
		t.Logf("Article 6 section: %q", sec)
	}

	// Check that most articles got chapter assignments
	assigned := 0
	for _, art := range articles {
		docID := "article_" + art.Slug
		if resolver.Chapter(docID) != "" {
			assigned++
		}
	}
	t.Logf("Articles with chapter: %d / %d", assigned, len(articles))

	if float64(assigned)/float64(len(articles)) < 0.8 {
		t.Errorf("Expected at least 80%% of articles to have chapters, got %d/%d", assigned, len(articles))
	}
}

func TestLiveChunkBuilding(t *testing.T) {
	fetcher := NewFetcher()

	items, err := fetcher.FetchBySlug("article", "6")
	if err != nil {
		t.Fatalf("Failed to fetch article 6: %v", err)
	}

	// Minimal resolver
	resolver := NewResolver(nil, nil, items, nil, nil)

	chunks, err := BuildChunks("article", items, resolver)
	if err != nil {
		t.Fatalf("BuildChunks failed: %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("Expected 1 chunk, got %d", len(chunks))
	}

	chunk := chunks[0]
	if chunk.DocType != "article" {
		t.Errorf("Expected doc_type 'article', got %q", chunk.DocType)
	}
	if chunk.DocID != "article_6" {
		t.Errorf("Expected doc_id 'article_6', got %q", chunk.DocID)
	}
	if chunk.Title == "" {
		t.Error("Chunk title should not be empty")
	}
	if chunk.Content == "" {
		t.Error("Chunk content should not be empty")
	}
	if len(chunk.CrossRefs) == 0 {
		t.Error("Chunk should have cross-references")
	}
	if chunk.Version != euAIActVersion {
		t.Errorf("Expected version %q, got %q", euAIActVersion, chunk.Version)
	}

	t.Logf("Chunk: doc_id=%s, title=%q, cross_refs=%d, content_len=%d",
		chunk.DocID, chunk.Title, len(chunk.CrossRefs), len(chunk.Content))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
