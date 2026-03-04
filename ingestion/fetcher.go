package ingestion

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

const (
	baseURL     = "https://artificialintelligenceact.eu/wp-json/wp/v2"
	rateLimit   = 1 * time.Second
	perPage     = 100
)

// Fetcher is a rate-limited HTTP client for the WordPress REST API.
type Fetcher struct {
	client   *http.Client
	lastCall time.Time
}

// NewFetcher creates a new rate-limited fetcher.
func NewFetcher() *Fetcher {
	return &Fetcher{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// FetchAll fetches all pages of a given WP content type and returns all items.
func (f *Fetcher) FetchAll(contentType string) ([]WPItem, error) {
	var all []WPItem
	page := 1

	for {
		f.waitForRateLimit()

		url := fmt.Sprintf("%s/%s?per_page=%d&page=%d", baseURL, contentType, perPage, page)
		log.Printf("Fetching %s (page %d)...", contentType, page)

		resp, err := f.client.Get(url)
		if err != nil {
			return nil, fmt.Errorf("GET %s: %w", url, err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read body %s: %w", url, err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
		}

		var items []WPItem
		if err := json.Unmarshal(body, &items); err != nil {
			return nil, fmt.Errorf("decode %s: %w", url, err)
		}

		all = append(all, items...)

		totalPages, _ := strconv.Atoi(resp.Header.Get("X-WP-TotalPages"))
		if page >= totalPages {
			break
		}
		page++
	}

	log.Printf("Fetched %d %s items", len(all), contentType)
	return all, nil
}

// FetchBySlug fetches a single item of the given content type by slug.
func (f *Fetcher) FetchBySlug(contentType, slug string) ([]WPItem, error) {
	f.waitForRateLimit()

	url := fmt.Sprintf("%s/%s?slug=%s", baseURL, contentType, slug)
	log.Printf("Fetching %s slug=%s...", contentType, slug)

	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read body %s: %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}

	var items []WPItem
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("decode %s: %w", url, err)
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no %s found with slug %q", contentType, slug)
	}

	return items, nil
}

func (f *Fetcher) waitForRateLimit() {
	elapsed := time.Since(f.lastCall)
	if elapsed < rateLimit {
		time.Sleep(rateLimit - elapsed)
	}
	f.lastCall = time.Now()
}
