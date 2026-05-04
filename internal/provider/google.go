package provider

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/genai"
	"google.golang.org/genai/tokenizer"
)

// newLocalTokenizerSilent calls tokenizer.NewLocalTokenizer while suppressing
// the SDK's experimental warning that is unconditionally printed to os.Stdout.
// The warning fires via sync.Once on the first call, before any model-name
// validation, so we must redirect stdout around the call itself.
func newLocalTokenizerSilent(model string) (*tokenizer.LocalTokenizer, error) {
	// Redirect os.Stdout to /dev/null for the duration of this call so the
	// SDK's fmt.Println warning doesn't pollute the token-count output.
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		// If we can't open /dev/null just call through — worst case the warning
		// appears, which is better than silently failing.
		return tokenizer.NewLocalTokenizer(model)
	}
	orig := os.Stdout
	os.Stdout = devNull
	tok, tokErr := tokenizer.NewLocalTokenizer(model)
	os.Stdout = orig
	devNull.Close()
	return tok, tokErr
}

// googleProvider counts tokens for Google Gemini models using a hybrid strategy:
//   1. Try local tokenizer (no network/key required for supported models).
//   2. If local tokenizer is unavailable or fails, fall back to remote API.
//      Remote fallback requires GEMINI_API_KEY or GOOGLE_API_KEY.
type googleProvider struct{}

func (p *googleProvider) Name() string { return "google" }

func (p *googleProvider) Count(ctx context.Context, model string, text string) (int, error) {
	contents := []*genai.Content{genai.NewContentFromText(text, genai.RoleUser)}

	// 1. Attempt local tokenization (warning suppressed — SDK prints to stdout
	//    unconditionally on first call, which would corrupt the numeric output).
	tok, err := newLocalTokenizerSilent(model)
	if err == nil {
		result, err := tok.CountTokens(contents, nil)
		if err == nil {
			return int(result.TotalTokens), nil
		}
		// CountTokens failed — fall through to remote.
	}
	// Local tokenizer creation failed or CountTokens failed — fall through to remote.

	// 2. Remote fallback: check for API key.
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return 0, fmt.Errorf("GEMINI_API_KEY (or GOOGLE_API_KEY) environment variable is required for Gemini remote tokenization")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return 0, fmt.Errorf("google: request failed: %w", err)
	}

	result, err := client.Models.CountTokens(ctx, model, contents, nil)
	if err != nil {
		return 0, fmt.Errorf("google: request failed: %w", err)
	}

	return int(result.TotalTokens), nil
}
