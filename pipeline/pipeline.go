package pipeline

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/prasanthmj/eu-ai-act-rag/llm"
	"github.com/prasanthmj/eu-ai-act-rag/rag"
)

// Pipeline orchestrates the 5-stage RAG pipeline.
type Pipeline struct {
	searcher  *rag.Searcher
	llmClient *llm.Client
	embedFn   EmbedFn
}

// NewPipeline creates a Pipeline with all dependencies.
func NewPipeline(searcher *rag.Searcher, llmClient *llm.Client, embedFn EmbedFn) *Pipeline {
	return &Pipeline{
		searcher:  searcher,
		llmClient: llmClient,
		embedFn:   embedFn,
	}
}

// RunFull runs all 5 stages and returns the complete result.
func (p *Pipeline) RunFull(ctx context.Context, description, domainHint string) (*PipelineResult, error) {
	// Stage 1: Classify
	log.Println("Pipeline stage 1: Classifying AI system...")
	classification, err := Classify(ctx, p.llmClient, description, domainHint)
	if err != nil {
		return nil, fmt.Errorf("stage 1 (classify): %w", err)
	}
	log.Printf("Classification: domain=%s, risk=%v", classification.Domain, classification.RiskTiers)

	// Stage 2: Retrieve
	log.Println("Pipeline stage 2: Retrieving relevant legal text...")
	chunks, err := Retrieve(ctx, p.searcher, p.embedFn, classification, description)
	if err != nil {
		return nil, fmt.Errorf("stage 2 (retrieve): %w", err)
	}
	log.Printf("Retrieved %d chunks across 3 hops", len(chunks))

	// Stage 3: Map obligations
	log.Println("Pipeline stage 3: Mapping obligations...")
	mapper, err := MapObligations(ctx, p.llmClient, classification, chunks)
	if err != nil {
		return nil, fmt.Errorf("stage 3 (map): %w", err)
	}
	log.Printf("Mapped %d obligations, risk_tier=%s", len(mapper.Obligations), mapper.RiskTier)

	// Stage 4: Score confidence
	log.Println("Pipeline stage 4: Scoring confidence...")
	scorer := Score(classification, chunks, mapper)
	log.Printf("Confidence: %.0f%%", scorer.OverallConfidence)

	// Stage 5: Generate checklist
	log.Println("Pipeline stage 5: Generating checklist...")
	checklist := GenerateChecklist(classification, mapper, scorer, description)

	return &PipelineResult{
		Classification: *classification,
		Chunks:         chunks,
		Obligations:    *mapper,
		Confidence:     *scorer,
		Checklist:      checklist,
	}, nil
}

// ClassifySystem runs Stage 1 only.
func (p *Pipeline) ClassifySystem(ctx context.Context, description, domainHint string) (*ClassifyResult, error) {
	return Classify(ctx, p.llmClient, description, domainHint)
}

// GetObligations runs retrieval + obligation mapping for a given risk tier and domain.
func (p *Pipeline) GetObligations(ctx context.Context, riskTier, domain string) (*MapperResult, error) {
	// Build a synthetic classification to drive retrieval
	classification := &ClassifyResult{
		Domain:    domain,
		RiskTiers: []string{riskTier},
		Reasoning: fmt.Sprintf("Direct query for %s obligations", riskTier),
	}

	description := fmt.Sprintf("%s AI system in %s domain", riskTier, domain)
	chunks, err := Retrieve(ctx, p.searcher, p.embedFn, classification, description)
	if err != nil {
		return nil, fmt.Errorf("retrieve for obligations: %w", err)
	}

	return MapObligations(ctx, p.llmClient, classification, chunks)
}

// CheckProhibited checks if a description matches Article 5 prohibited practices.
func (p *Pipeline) CheckProhibited(ctx context.Context, description string) (string, error) {
	userMsg := fmt.Sprintf("Assess whether this AI system involves prohibited practices under Article 5:\n\n%s", description)

	result, err := p.llmClient.Complete(ctx, prohibitedCheckPrompt, userMsg)
	if err != nil {
		return "", fmt.Errorf("check prohibited: %w", err)
	}
	return result, nil
}

// LookupArticle retrieves a specific article, recital, or annex by reference.
func (p *Pipeline) LookupArticle(ctx context.Context, reference string) (*rag.SearchResult, error) {
	// Determine collection from reference prefix
	collection := ""
	switch {
	case strings.HasPrefix(reference, "article_"):
		collection = articleCollection
	case strings.HasPrefix(reference, "recital_"):
		collection = recitalCollection
	case strings.HasPrefix(reference, "annex_"):
		collection = annexCollection
	default:
		return nil, fmt.Errorf("unknown reference format: %q (expected article_N, recital_N, or annex_N)", reference)
	}

	return p.searcher.LookupByDocID(ctx, collection, reference)
}
