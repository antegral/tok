package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
)

// anthropicProvider counts tokens via the Anthropic Messages.CountTokens API.
// An API key is required (ANTHROPIC_API_KEY env var).
type anthropicProvider struct{}

func (p *anthropicProvider) Name() string { return "anthropic" }

func (p *anthropicProvider) Count(ctx context.Context, model string, text string) (int, error) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		return 0, fmt.Errorf("ANTHROPIC_API_KEY environment variable is required for Claude models")
	}

	client := anthropic.NewClient()

	count, err := client.Messages.CountTokens(ctx, anthropic.MessageCountTokensParams{
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(text)),
		},
		Model: anthropic.Model(model),
	})
	if err != nil {
		return 0, fmt.Errorf("anthropic: request failed: %w", err)
	}

	return int(count.InputTokens), nil
}
