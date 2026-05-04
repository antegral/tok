// Package provider defines the Provider interface for token counting and the
// Resolve registry that maps a model spec string to a concrete Provider.
package provider

import "context"

// Provider counts tokens for a given text under a given model.
//
// The model parameter does NOT include the provider prefix (e.g. "gpt-4o",
// not "openai/gpt-4o"). For HuggingFace, however, the model parameter is the
// full "<org>/<repo>" spec — this is intentional and matches the upstream
// tokenizer library's expected format.
type Provider interface {
	// Count returns the token count for text under the given model.
	Count(ctx context.Context, model string, text string) (int, error)
	// Name returns the provider's stable identifier (e.g. "openai", "hf").
	Name() string
}
