package pipeline

import (
	"context"
	"fmt"
	"strings"

	"github.com/prasanthmj/eu-ai-act-rag/llm"
)

// Score runs Stage 4: LLM-based verification of the classification and obligations
// against the retrieved legal text.
func Score(ctx context.Context, llmClient *llm.Client, classification *ClassifyResult, chunks []RetrievedChunk, mapper *MapperResult) (*ScorerResult, error) {
	// Build the retrieved text context
	var chunkTexts []string
	for i, c := range chunks {
		chunkTexts = append(chunkTexts, fmt.Sprintf("[%d] %s — %s\n%s", i+1, c.DocID, c.Title, c.Content))
	}

	// Build the obligations summary
	var obligationLines []string
	for _, ob := range mapper.Obligations {
		obligationLines = append(obligationLines, fmt.Sprintf("- %s (%s): %s [Priority: %s]", ob.Article, ob.Title, ob.Summary, ob.Priority))
	}

	userMsg := fmt.Sprintf(`CLASSIFICATION:
- Domain: %s
- Risk Tier: %s
- Classification Basis: %s
- Reasoning: %s

MAPPED OBLIGATIONS:
%s

RETRIEVED LEGAL TEXT:
%s`,
		classification.Domain,
		strings.Join(classification.RiskTiers, ", "),
		strings.Join(mapper.ClassificationBasis, ", "),
		classification.Reasoning,
		strings.Join(obligationLines, "\n"),
		strings.Join(chunkTexts, "\n\n"),
	)

	var result ScorerResult
	if err := llmClient.CompleteJSON(ctx, scorerSystemPrompt, userMsg, &result); err != nil {
		return nil, fmt.Errorf("score confidence: %w", err)
	}

	// Compute citation accuracy from verifications
	if len(result.Verifications) > 0 {
		verified := 0
		for _, v := range result.Verifications {
			if v.Status == "verified" {
				verified++
			}
		}
		result.CitationAccuracy = float64(verified) / float64(len(result.Verifications)) * 100
	}

	return &result, nil
}
