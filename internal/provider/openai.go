package provider

import (
	"context"
	"fmt"

	"github.com/tiktoken-go/tokenizer"
)

// openaiProvider counts tokens locally via tiktoken-go. No API key required.
type openaiProvider struct{}

func (p *openaiProvider) Name() string { return "openai" }

func (p *openaiProvider) Count(_ context.Context, model string, text string) (int, error) {
	enc, err := tokenizer.ForModel(tokenizer.Model(model))
	if err != nil {
		return 0, fmt.Errorf("openai: unknown model %q", model)
	}
	n, err := enc.Count(text)
	if err != nil {
		return 0, fmt.Errorf("openai: count failed: %w", err)
	}
	return n, nil
}
