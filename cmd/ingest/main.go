package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/prasanthmj/eu-ai-act-rag/ingestion"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	ctx := context.Background()

	// Step 1: Ensure data directories exist
	if err := ingestion.EnsureDataDirs(); err != nil {
		log.Fatalf("Failed to create data dirs: %v", err)
	}

	fetcher := ingestion.NewFetcher()

	// Step 2: Fetch chapters and sections (needed for resolver)
	log.Println("=== Fetching chapters and sections ===")
	chapters, err := fetcher.FetchAll("chapter")
	if err != nil {
		log.Fatalf("Fetch chapters: %v", err)
	}
	if err := ingestion.SaveRawJSON("chapters.json", chapters); err != nil {
		log.Fatalf("Save chapters: %v", err)
	}

	sections, err := fetcher.FetchAll("section")
	if err != nil {
		log.Fatalf("Fetch sections: %v", err)
	}
	if err := ingestion.SaveRawJSON("sections.json", sections); err != nil {
		log.Fatalf("Save sections: %v", err)
	}

	// Step 3: Fetch articles, recitals, annexes
	log.Println("=== Fetching articles, recitals, annexes ===")
	articles, err := fetcher.FetchAll("article")
	if err != nil {
		log.Fatalf("Fetch articles: %v", err)
	}
	if err := ingestion.SaveRawJSON("articles.json", articles); err != nil {
		log.Fatalf("Save articles: %v", err)
	}

	recitals, err := fetcher.FetchAll("recital")
	if err != nil {
		log.Fatalf("Fetch recitals: %v", err)
	}
	if err := ingestion.SaveRawJSON("recitals.json", recitals); err != nil {
		log.Fatalf("Save recitals: %v", err)
	}

	annexes, err := fetcher.FetchAll("annex")
	if err != nil {
		log.Fatalf("Fetch annexes: %v", err)
	}
	if err := ingestion.SaveRawJSON("annexes.json", annexes); err != nil {
		log.Fatalf("Save annexes: %v", err)
	}

	// Step 4: Build resolver
	log.Println("=== Building resolver ===")
	resolver := ingestion.NewResolver(chapters, sections, articles, recitals, annexes)

	// Step 5: Build chunks
	log.Println("=== Building chunks ===")
	articleChunks, err := ingestion.BuildChunks("article", articles, resolver)
	if err != nil {
		log.Fatalf("Build article chunks: %v", err)
	}
	if err := ingestion.SaveProcessedJSON("article_chunks.json", articleChunks); err != nil {
		log.Fatalf("Save article chunks: %v", err)
	}

	recitalChunks, err := ingestion.BuildChunks("recital", recitals, resolver)
	if err != nil {
		log.Fatalf("Build recital chunks: %v", err)
	}
	if err := ingestion.SaveProcessedJSON("recital_chunks.json", recitalChunks); err != nil {
		log.Fatalf("Save recital chunks: %v", err)
	}

	annexChunks, err := ingestion.BuildChunks("annex", annexes, resolver)
	if err != nil {
		log.Fatalf("Build annex chunks: %v", err)
	}
	if err := ingestion.SaveProcessedJSON("annex_chunks.json", annexChunks); err != nil {
		log.Fatalf("Save annex chunks: %v", err)
	}

	// Step 6: Build sparse encoder from all chunks
	log.Println("=== Building sparse encoder (BM25) ===")
	allChunks := make([]string, 0, len(articleChunks)+len(recitalChunks)+len(annexChunks))
	for _, c := range articleChunks {
		allChunks = append(allChunks, c.Title+"\n\n"+c.Content)
	}
	for _, c := range recitalChunks {
		allChunks = append(allChunks, c.Title+"\n\n"+c.Content)
	}
	for _, c := range annexChunks {
		allChunks = append(allChunks, c.Title+"\n\n"+c.Content)
	}
	sparseEnc := ingestion.NewSparseEncoder()
	sparseEnc.Fit(allChunks)
	log.Printf("Sparse encoder fitted on %d documents", len(allChunks))

	if err := sparseEnc.Save("ingestion/data/processed/sparse_encoder.json"); err != nil {
		log.Fatalf("Save sparse encoder: %v", err)
	}
	log.Println("Saved sparse encoder to ingestion/data/processed/sparse_encoder.json")

	// Step 7: Generate dense + sparse embeddings
	log.Println("=== Generating embeddings ===")
	embedder := ingestion.NewEmbedder()

	articleEmbedded, err := embedder.EmbedChunks(ctx, articleChunks, sparseEnc)
	if err != nil {
		log.Fatalf("Embed articles: %v", err)
	}

	recitalEmbedded, err := embedder.EmbedChunks(ctx, recitalChunks, sparseEnc)
	if err != nil {
		log.Fatalf("Embed recitals: %v", err)
	}

	annexEmbedded, err := embedder.EmbedChunks(ctx, annexChunks, sparseEnc)
	if err != nil {
		log.Fatalf("Embed annexes: %v", err)
	}

	// Step 8: Connect to Qdrant and create collections
	log.Println("=== Setting up Qdrant ===")
	qdrantHost := envOrDefault("QDRANT_HOST", "localhost")
	qdrantPort, _ := strconv.Atoi(envOrDefault("QDRANT_PORT", "6334"))

	store, err := ingestion.NewStore(qdrantHost, qdrantPort)
	if err != nil {
		log.Fatalf("Connect to Qdrant: %v", err)
	}
	defer store.Close()

	collections := map[string][]ingestion.ChunkWithEmbedding{
		"eu_ai_act_articles": articleEmbedded,
		"eu_ai_act_recitals": recitalEmbedded,
		"eu_ai_act_annexes":  annexEmbedded,
	}

	for name, chunks := range collections {
		if err := store.RecreateCollection(ctx, name); err != nil {
			log.Fatalf("Recreate collection %s: %v", name, err)
		}
		if err := store.UpsertChunks(ctx, name, chunks); err != nil {
			log.Fatalf("Upsert to %s: %v", name, err)
		}
	}

	// Step 9: Summary
	fmt.Printf("\n=== Ingestion Complete ===\n")
	fmt.Printf("Ingested %d articles, %d recitals, %d annexes\n",
		len(articleChunks), len(recitalChunks), len(annexChunks))
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
