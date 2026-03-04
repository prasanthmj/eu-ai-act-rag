package ingestion

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var (
	// Match both absolute and relative URLs for articles, annexes, and recitals
	articleHrefRe = regexp.MustCompile(`/article/(\d+)`)
	annexHrefRe   = regexp.MustCompile(`/annex/([a-z0-9]+)`)
	recitalHrefRe = regexp.MustCompile(`/recital/(\d+)`)
)

// ExtractCrossRefs finds all article, annex, and recital references in HTML content.
func ExtractCrossRefs(html string) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var refs []string

	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		var ref string
		if m := articleHrefRe.FindStringSubmatch(href); m != nil {
			ref = "article_" + m[1]
		} else if m := annexHrefRe.FindStringSubmatch(href); m != nil {
			ref = "annex_" + m[1]
		} else if m := recitalHrefRe.FindStringSubmatch(href); m != nil {
			ref = "recital_" + m[1]
		}

		if ref != "" && !seen[ref] {
			seen[ref] = true
			refs = append(refs, ref)
		}
	})

	return refs, nil
}

// HTMLToPlainText converts HTML content to structured plain text.
// Handles paragraph indentation based on padding-left style.
// Strips recital reference spans before text extraction.
func HTMLToPlainText(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	// Remove recital reference spans
	doc.Find("span.aia-recital-ref").Remove()

	var paragraphs []string

	doc.Find("p").Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text == "" {
			return
		}

		// Determine indentation from inline style
		style, _ := s.Attr("style")
		indent := ""
		if strings.Contains(style, "padding-left: 80px") || strings.Contains(style, "padding-left:80px") {
			indent = "    " // 4 spaces for sub-sub-level
		} else if strings.Contains(style, "padding-left: 40px") || strings.Contains(style, "padding-left:40px") {
			indent = "  " // 2 spaces for sub-level
		}

		paragraphs = append(paragraphs, indent+text)
	})

	// If no <p> tags found, fall back to full text
	if len(paragraphs) == 0 {
		text := strings.TrimSpace(doc.Text())
		if text != "" {
			paragraphs = append(paragraphs, text)
		}
	}

	return strings.Join(paragraphs, "\n\n"), nil
}

// CleanTitle strips HTML tags from a title string.
func CleanTitle(rendered string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rendered))
	if err != nil {
		return rendered
	}
	return strings.TrimSpace(doc.Text())
}
