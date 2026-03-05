package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gomcpgo/mcp/pkg/handler"
	"github.com/gomcpgo/mcp/pkg/server"
	"github.com/openai/openai-go"
	"github.com/prasanthmj/eu-ai-act-rag/api"
	"github.com/prasanthmj/eu-ai-act-rag/ingestion"
	"github.com/prasanthmj/eu-ai-act-rag/llm"
	"github.com/prasanthmj/eu-ai-act-rag/pipeline"
	"github.com/prasanthmj/eu-ai-act-rag/rag"
	"github.com/prasanthmj/eu-ai-act-rag/tools"
)

func main() {
	mode := flag.String("mode", "http", "Run mode: 'mcp' for MCP server, 'http' for HTTP API")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	qdrantHost := envOrDefault("QDRANT_HOST", "localhost")
	qdrantPort, _ := strconv.Atoi(envOrDefault("QDRANT_PORT", "6334"))
	httpPort := envOrDefault("HTTP_PORT", "8552")

	// Initialize Qdrant searcher
	searcher, err := rag.NewSearcher(qdrantHost, qdrantPort)
	if err != nil {
		log.Fatalf("Connect to Qdrant: %v", err)
	}
	defer searcher.Close()

	// Initialize LLM client
	llmClient := llm.NewClient()

	// Initialize embedding function
	openaiClient := openai.NewClient()
	embedFn := func(ctx context.Context, text string) ([]float32, error) {
		return rag.EmbedQuery(ctx, openaiClient, text)
	}

	// Load sparse encoder for hybrid search
	sparseEncPath := envOrDefault("SPARSE_ENCODER_PATH", "ingestion/data/processed/sparse_encoder.json")
	sparseEnc, err := ingestion.LoadSparseEncoder(sparseEncPath)
	if err != nil {
		log.Printf("Warning: could not load sparse encoder (%v) — falling back to dense-only search", err)
	}

	var sparseEmbedFn pipeline.SparseEmbedFn
	if sparseEnc != nil {
		sparseEmbedFn = func(text string) *rag.SparseQuery {
			sv := sparseEnc.Encode(text)
			return &rag.SparseQuery{Indices: sv.Indices, Values: sv.Values}
		}
	}

	// Create pipeline
	p := pipeline.NewPipeline(searcher, llmClient, embedFn, sparseEmbedFn)

	switch *mode {
	case "mcp":
		log.Println("Starting MCP server (stdio)...")
		registry := handler.NewHandlerRegistry()
		registry.RegisterToolHandler(tools.NewHandler(p))
		srv := server.New(server.Options{
			Name:     "eu-ai-act-mcp",
			Version:  "1.0.0",
			Registry: registry,
		})
		if err := srv.Run(); err != nil {
			log.Fatalf("MCP server error: %v", err)
		}

	case "http":
		router := api.NewRouter(p)
		log.Printf("Starting HTTP server on :%s...", httpPort)
		if err := http.ListenAndServe(":"+httpPort, router); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}

	default:
		log.Fatalf("Unknown mode: %s (use 'mcp' or 'http')", *mode)
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
