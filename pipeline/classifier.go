package pipeline

import (
	"context"
	"fmt"

	"github.com/prasanthmj/eu-ai-act-rag/llm"
)

// Classify runs Stage 1: classifies an AI system description into domain and risk tier.
func Classify(ctx context.Context, llmClient *llm.Client, description, domainHint string) (*ClassifyResult, error) {
	userMsg := fmt.Sprintf("Classify this AI system:\n\n%s", description)
	if domainHint != "" {
		userMsg += fmt.Sprintf("\n\nDomain hint: %s", domainHint)
	}

	var result ClassifyResult
	if err := llmClient.CompleteJSON(ctx, classifierSystemPrompt, userMsg, &result); err != nil {
		return nil, fmt.Errorf("classify: %w", err)
	}
	return &result, nil
}
