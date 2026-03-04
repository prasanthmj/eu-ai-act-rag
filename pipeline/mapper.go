package pipeline

import (
	"context"
	"fmt"
	"strings"

	"github.com/prasanthmj/eu-ai-act-rag/llm"
)

// MapObligations runs Stage 3: maps retrieved chunks to structured compliance obligations.
func MapObligations(ctx context.Context, llmClient *llm.Client, classification *ClassifyResult, chunks []RetrievedChunk) (*MapperResult, error) {
	// Build context from retrieved chunks
	var context_parts []string
	for i, c := range chunks {
		context_parts = append(context_parts, fmt.Sprintf("[%d] %s — %s\n%s", i+1, c.DocID, c.Title, c.Content))
	}

	userMsg := fmt.Sprintf(`AI System Classification:
- Domain: %s
- Risk Tiers: %s
- Reasoning: %s
- Needs Profiling Check: %v
- Exception Candidate: %v

Legal Text Context:
%s

Based on the classification and legal text above, identify all applicable compliance obligations.`,
		classification.Domain,
		strings.Join(classification.RiskTiers, ", "),
		classification.Reasoning,
		classification.NeedsProfiling,
		classification.ExceptionCandidate,
		strings.Join(context_parts, "\n\n"),
	)

	var result MapperResult
	if err := llmClient.CompleteJSON(ctx, mapperSystemPrompt, userMsg, &result); err != nil {
		return nil, fmt.Errorf("map obligations: %w", err)
	}
	return &result, nil
}
