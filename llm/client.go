package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
)

// Client wraps OpenAI chat completions.
type Client struct {
	api openai.Client
}

// NewClient creates a Client. Uses OPENAI_API_KEY env var automatically.
func NewClient() *Client {
	return &Client{api: openai.NewClient()}
}

// Complete sends a system + user message and returns the assistant's text response.
func (c *Client) Complete(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	resp, err := c.api.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: shared.ChatModelGPT4oMini,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userMessage),
		},
	})
	if err != nil {
		return "", fmt.Errorf("chat completion: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("chat completion: no choices returned")
	}
	return resp.Choices[0].Message.Content, nil
}

// CompleteJSON sends a system + user message, expecting JSON output, and unmarshals into target.
func (c *Client) CompleteJSON(ctx context.Context, systemPrompt, userMessage string, target any) error {
	text, err := c.Complete(ctx, systemPrompt, userMessage)
	if err != nil {
		return err
	}

	// Strip markdown code fences if present
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		lines := strings.Split(text, "\n")
		// Remove first and last lines (```json and ```)
		if len(lines) > 2 {
			lines = lines[1 : len(lines)-1]
		}
		text = strings.Join(lines, "\n")
	}

	if err := json.Unmarshal([]byte(text), target); err != nil {
		return fmt.Errorf("unmarshal LLM JSON response: %w\nraw response: %s", err, text)
	}
	return nil
}
